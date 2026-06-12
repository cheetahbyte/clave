package license

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestTimePtrValid(t *testing.T) {
	now := time.Now().UTC()
	ts := pgtype.Timestamptz{Time: now, Valid: true}
	result := timePtr(ts)
	if result == nil {
		t.Fatal("expected non-nil time pointer")
	}
	if !result.Equal(now) {
		t.Fatalf("expected %v, got %v", now, *result)
	}
}

func TestTimePtrInvalid(t *testing.T) {
	ts := pgtype.Timestamptz{Valid: false}
	result := timePtr(ts)
	if result != nil {
		t.Fatal("expected nil time pointer for invalid timestamp")
	}
}

func TestToAdminLicenseItem(t *testing.T) {
	id := uuid.New()
	created := pgtype.Timestamptz{Time: time.Now().Add(-24 * time.Hour), Valid: true}
	expires := pgtype.Timestamptz{Time: time.Now().Add(30 * 24 * time.Hour), Valid: true}

	item := toAdminLicenseItem(id, "test@example.com", "TestProduct", true, false, 5, 3, created, expires)

	if item.ID != id.String() {
		t.Fatalf("expected ID %s, got %s", id.String(), item.ID)
	}
	if item.CustomerEmail != "test@example.com" {
		t.Fatalf("expected CustomerEmail 'test@example.com', got '%s'", item.CustomerEmail)
	}
	if item.ProductName != "TestProduct" {
		t.Fatalf("expected ProductName 'TestProduct', got '%s'", item.ProductName)
	}
	if !item.IsActive {
		t.Fatal("expected IsActive=true")
	}
	if item.IsTrial {
		t.Fatal("expected IsTrial=false")
	}
	if item.MaxActivations != 5 {
		t.Fatalf("expected MaxActivations=5, got %d", item.MaxActivations)
	}
	if item.ActivationCount != 3 {
		t.Fatalf("expected ActivationCount=3, got %d", item.ActivationCount)
	}
	if item.CreatedAt == nil || !item.CreatedAt.Equal(created.Time) {
		t.Fatal("expected CreatedAt to match")
	}
	if item.ExpiresAt == nil || !item.ExpiresAt.Equal(expires.Time) {
		t.Fatal("expected ExpiresAt to match")
	}
}

func TestToAdminLicenseItemNilTimestamps(t *testing.T) {
	id := uuid.New()
	item := toAdminLicenseItem(id, "", "", false, true, 1, 0,
		pgtype.Timestamptz{Valid: false},
		pgtype.Timestamptz{Valid: false})

	if item.CreatedAt != nil {
		t.Fatal("expected nil CreatedAt")
	}
	if item.ExpiresAt != nil {
		t.Fatal("expected nil ExpiresAt")
	}
}
