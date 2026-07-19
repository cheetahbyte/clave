package diagnostics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeAdoptionRepository struct {
	latest    []LatestCheckin
	daily     []DailyVersion
	latestErr error
	dailyErr  error
}

func (f *fakeAdoptionRepository) ListLatestCheckins(context.Context, uuid.UUID, pgtype.UUID, int) ([]LatestCheckin, error) {
	return f.latest, f.latestErr
}

func (f *fakeAdoptionRepository) ListDailyVersions(context.Context, uuid.UUID, pgtype.UUID, int) ([]DailyVersion, error) {
	return f.daily, f.dailyErr
}

func TestVersionAdoptionAggregatesLatestDevicesAndTrend(t *testing.T) {
	jan1 := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	jan2 := jan1.Add(24 * time.Hour)
	repo := &fakeAdoptionRepository{
		latest: []LatestCheckin{
			{ActivationID: uuid.New(), Version: "2.0.0", CreatedAt: jan2},
			{ActivationID: uuid.New(), Version: "1.9.0", CreatedAt: jan2},
			{ActivationID: uuid.New(), Version: "2.0.0", CreatedAt: jan1},
		},
		daily: []DailyVersion{
			{Date: jan1, Version: "1.9.0", DeviceCount: 2},
			{Date: jan1, Version: "2.0.0", DeviceCount: 1},
			{Date: jan2, Version: "2.0.0", DeviceCount: 3},
		},
	}

	result, err := NewService(repo).VersionAdoption(context.Background(), uuid.New(), pgtype.UUID{}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if result.ActiveDevices != 3 || result.VersionCount != 2 {
		t.Fatalf("summary = %d devices, %d versions", result.ActiveDevices, result.VersionCount)
	}
	if got := result.Distribution[0]; got.Version != "2.0.0" || got.DeviceCount != 2 || got.Percentage != 66.7 {
		t.Fatalf("first distribution = %#v", got)
	}
	if got := result.Distribution[1]; got.Version != "1.9.0" || got.DeviceCount != 1 || got.Percentage != 33.3 {
		t.Fatalf("second distribution = %#v", got)
	}
	if len(result.Trend) != 2 || len(result.Trend[0].Versions) != 2 || result.Trend[1].Date != "2026-01-02" {
		t.Fatalf("trend = %#v", result.Trend)
	}
	if len(result.Devices) != 3 || result.Devices[0].Version != "2.0.0" {
		t.Fatalf("devices = %#v", result.Devices)
	}
}

func TestVersionAdoptionReturnsEmptyCollections(t *testing.T) {
	result, err := NewService(&fakeAdoptionRepository{}).VersionAdoption(context.Background(), uuid.New(), pgtype.UUID{}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if result.Distribution == nil || result.Trend == nil || result.Devices == nil {
		t.Fatalf("empty collections must be arrays: %#v", result)
	}
}

func TestVersionAdoptionPropagatesRepositoryErrors(t *testing.T) {
	want := errors.New("database unavailable")
	_, err := NewService(&fakeAdoptionRepository{latestErr: want}).VersionAdoption(context.Background(), uuid.New(), pgtype.UUID{}, 30)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
