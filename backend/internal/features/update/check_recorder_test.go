package update

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type blockingRecordStore struct {
	started chan struct{}
	release chan struct{}
	inserts atomic.Int32
}

func (s *blockingRecordStore) InsertUpdateCheck(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, string, string, string, string, string, string, string, *uuid.UUID) error {
	if s.inserts.Add(1) == 1 {
		close(s.started)
	}
	<-s.release
	return nil
}
func TestUpdateCheckRecorderDropsWhenFullWithoutBlocking(t *testing.T) {
	store := &blockingRecordStore{started: make(chan struct{}), release: make(chan struct{})}
	recorder := NewUpdateCheckRecorder(store, 1)
	recorder.Record(UpdateCheckRecord{})
	<-store.started
	recorder.Record(UpdateCheckRecord{})
	start := time.Now()
	recorder.Record(UpdateCheckRecord{})
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("full recorder blocked request path")
	}
	if recorder.dropped.Load() != 1 {
		t.Fatalf("dropped = %d", recorder.dropped.Load())
	}
	close(store.release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	recorder.Close(ctx)
	if store.inserts.Load() != 2 {
		t.Fatalf("inserts = %d", store.inserts.Load())
	}
}
