package diagnostics

import (
	"context"
	"errors"
	"testing"
	"time"
)

type checkinStoreStub struct {
	recordErr     error
	recorded      chan Checkin
	cleanupCalled chan int
}

func (s *checkinStoreStub) RecordCheckin(_ context.Context, checkin Checkin) error {
	if s.recorded != nil {
		s.recorded <- checkin
	}
	return s.recordErr
}

func (s *checkinStoreStub) AggregateClosedDatesAndCleanup(_ context.Context, retentionDays int) (int64, error) {
	if s.cleanupCalled != nil {
		s.cleanupCalled <- retentionDays
	}
	return 0, nil
}

func TestRecorderPersistsCheckinSynchronously(t *testing.T) {
	store := &checkinStoreStub{recorded: make(chan Checkin, 1)}
	recorder := NewRecorder(store, 7)
	want := Checkin{Version: "2.0.0"}
	if err := recorder.Record(t.Context(), want); err != nil {
		t.Fatal(err)
	}
	if got := <-store.recorded; got.Version != want.Version {
		t.Fatalf("recorded version = %q, want %q", got.Version, want.Version)
	}
	recorder.Close(t.Context())
}

func TestRecorderReturnsWriteError(t *testing.T) {
	want := errors.New("database unavailable")
	recorder := NewRecorder(&checkinStoreStub{recordErr: want}, 7)
	if err := recorder.Record(t.Context(), Checkin{}); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	recorder.Close(t.Context())
}

func TestRecorderRunsDailyWorkAtStartupWithConfiguredRetention(t *testing.T) {
	called := make(chan int, 1)
	recorder := NewRecorder(&checkinStoreStub{cleanupCalled: called}, 7)
	select {
	case got := <-called:
		if got != 7 {
			t.Fatalf("retention = %d, want 7", got)
		}
	case <-time.After(time.Second):
		t.Fatal("startup aggregation was not called")
	}
	recorder.Close(t.Context())
}
