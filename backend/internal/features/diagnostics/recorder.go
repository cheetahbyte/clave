package diagnostics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cheetahbyte/clave/internal/observability"
)

type checkinStore interface {
	RecordCheckin(context.Context, Checkin) error
	AggregateClosedDatesAndCleanup(context.Context, int) (int64, error)
}

type Recorder struct {
	store         checkinStore
	retentionDays int
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewRecorder(store checkinStore, retentionDays int) *Recorder {
	ctx, cancel := context.WithCancel(context.Background())
	recorder := &Recorder{store: store, retentionDays: retentionDays, cancel: cancel}
	recorder.wg.Add(1)
	go recorder.run(ctx)
	return recorder
}

func (r *Recorder) Record(ctx context.Context, checkin Checkin) error {
	if err := r.store.RecordCheckin(ctx, checkin); err != nil {
		observability.CountClientCheckinTelemetry(ctx, "error")
		return err
	}
	observability.CountClientCheckinTelemetry(ctx, "recorded")
	return nil
}

func (r *Recorder) run(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	r.aggregateAndCleanup(ctx)
	for {
		select {
		case <-ticker.C:
			r.aggregateAndCleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Recorder) aggregateAndCleanup(ctx context.Context) {
	if _, err := r.store.AggregateClosedDatesAndCleanup(ctx, r.retentionDays); err != nil && ctx.Err() == nil {
		slog.Warn("failed to aggregate or clean client check-ins", "err", err)
	}
}

func (r *Recorder) Close(ctx context.Context) {
	r.cancel()
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
