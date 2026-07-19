package diagnostics

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cheetahbyte/clave/internal/observability"
)

type checkinStore interface {
	InsertCheckin(context.Context, Checkin) error
	DeleteExpiredCheckins(context.Context, int) (int64, error)
}

type Recorder struct {
	store         checkinStore
	retentionDays int
	records       chan Checkin
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	dropped       atomic.Uint64
}

func NewRecorder(store checkinStore, retentionDays, capacity int) *Recorder {
	if capacity < 1 {
		capacity = 256
	}
	ctx, cancel := context.WithCancel(context.Background())
	recorder := &Recorder{
		store: store, retentionDays: retentionDays,
		records: make(chan Checkin, capacity), cancel: cancel,
	}
	recorder.wg.Add(1)
	go recorder.run(ctx)
	return recorder
}

func (r *Recorder) Record(checkin Checkin) {
	select {
	case r.records <- checkin:
	default:
		r.dropped.Add(1)
		observability.CountClientCheckinTelemetry(context.Background(), "dropped")
	}
}

func (r *Recorder) run(ctx context.Context) {
	defer r.wg.Done()
	var cleanup <-chan time.Time
	if r.retentionDays > 0 {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		cleanup = ticker.C
		r.cleanup(ctx)
	}
	for {
		select {
		case record, ok := <-r.records:
			if !ok {
				return
			}
			if err := r.store.InsertCheckin(ctx, record); err != nil {
				slog.Warn("failed to record client check-in", "err", err)
				observability.CountClientCheckinTelemetry(context.Background(), "error")
			} else {
				observability.CountClientCheckinTelemetry(context.Background(), "recorded")
			}
		case <-cleanup:
			r.cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Recorder) cleanup(ctx context.Context) {
	if _, err := r.store.DeleteExpiredCheckins(ctx, r.retentionDays); err != nil {
		slog.Warn("failed to clean expired client check-ins", "err", err)
	}
}

func (r *Recorder) Close(ctx context.Context) {
	close(r.records)
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		r.cancel()
		<-done
	}
	r.cancel()
}
