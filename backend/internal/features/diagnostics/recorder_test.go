package diagnostics

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type blockingCheckinStore struct {
	started chan struct{}
	release chan struct{}
	inserts atomic.Int32
}

func (s *blockingCheckinStore) InsertCheckin(context.Context, Checkin) error {
	if s.inserts.Add(1) == 1 {
		close(s.started)
	}
	<-s.release
	return nil
}

func (*blockingCheckinStore) DeleteExpiredCheckins(context.Context, int) (int64, error) {
	return 0, nil
}

func TestRecorderDropsWhenFullWithoutBlocking(t *testing.T) {
	store := &blockingCheckinStore{started: make(chan struct{}), release: make(chan struct{})}
	recorder := NewRecorder(store, 0, 1)
	recorder.Record(Checkin{})
	<-store.started
	recorder.Record(Checkin{})

	start := time.Now()
	recorder.Record(Checkin{})
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("full recorder blocked request path for %s", elapsed)
	}
	if got := recorder.dropped.Load(); got != 1 {
		t.Fatalf("dropped = %d, want 1", got)
	}

	close(store.release)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	recorder.Close(ctx)
	if got := store.inserts.Load(); got != 2 {
		t.Fatalf("inserts = %d, want 2", got)
	}
}
