package update

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/google/uuid"
)

type UpdateCheckRecord struct {
	OrganizationID, ProductID, LicenseID                    uuid.UUID
	Platform, Channel, ProviderKey                          string
	CurrentVersion, CurrentBuild, Arch, OSVersion, Decision string
	SelectedReleaseID                                       *uuid.UUID
}

type checkRecordStore interface {
	InsertUpdateCheck(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string, string, string, string, string, *uuid.UUID) error
}

type UpdateCheckRecorder struct {
	store   checkRecordStore
	records chan UpdateCheckRecord
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	dropped atomic.Uint64
}

func NewUpdateCheckRecorder(store checkRecordStore, capacity int) *UpdateCheckRecorder {
	if capacity < 1 {
		capacity = 256
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &UpdateCheckRecorder{store: store, records: make(chan UpdateCheckRecord, capacity), cancel: cancel}
	r.wg.Add(1)
	go r.run(ctx)
	return r
}

func (r *UpdateCheckRecorder) Record(record UpdateCheckRecord) {
	select {
	case r.records <- record:
	default:
		r.dropped.Add(1)
		observability.CountUpdateCheckTelemetry(context.Background(), "dropped")
	}
}

func (r *UpdateCheckRecorder) run(ctx context.Context) {
	defer r.wg.Done()
	for {
		select {
		case record, ok := <-r.records:
			if !ok {
				return
			}
			if err := r.store.InsertUpdateCheck(ctx, record.OrganizationID, record.ProductID, record.LicenseID,
				record.Platform, record.Channel, record.ProviderKey, record.CurrentVersion, record.CurrentBuild,
				record.Arch, record.OSVersion, record.Decision, record.SelectedReleaseID); err != nil {
				slog.Warn("failed to record update check", "err", err)
				observability.CountUpdateCheckTelemetry(context.Background(), "error")
			} else {
				observability.CountUpdateCheckTelemetry(context.Background(), "recorded")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *UpdateCheckRecorder) Close(ctx context.Context) {
	close(r.records)
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		r.cancel()
		<-done
	}
	r.cancel()
}
