package license

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// TestFetchAndMapNoRowsReturnsNilPointerNilError is a regression test for the
// nil-pointer dereference panic on the public /licenses/activate endpoint.
//
// fetchAndMap converts sql.ErrNoRows into the zero value of R plus a nil
// error. When R is *License, the zero value is nil. Callers that only check
// err (and not the returned pointer) will dereference nil and panic.
//
// The activation service previously did exactly this. The fix is twofold:
//   - activation.Service.Activate now guards `err != nil || lic == nil`
//   - license.Service.GetByDigest/GetByID now return ErrNotFound instead of
//     (nil, nil), so future callers cannot fall into the same trap.
//
// This test pins the fetchAndMap contract so the root cause cannot silently
// regress.
func TestFetchAndMapNoRowsReturnsNilPointerNilError(t *testing.T) {
	t.Parallel()

	got, err := fetchAndMap(
		context.Background(),
		func(_ context.Context) (struct{}, error) {
			return struct{}{}, sql.ErrNoRows
		},
		func(_ struct{}) *License {
			t.Fatal("mapFunc must not be called on ErrNoRows")
			return nil
		},
	)

	if err != nil {
		t.Fatalf("expected nil error for ErrNoRows, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil *License for ErrNoRows, got %v", got)
	}
}

// TestFetchAndMapNoRowsReturnsNilForPointerResultType confirms the generic
// zero-value behavior for any pointer result type R, which is the property
// that makes the no-row case dangerous for pointer-returning callers.
func TestFetchAndMapNoRowsReturnsNilForPointerResultType(t *testing.T) {
	t.Parallel()

	type custom struct{ X int }
	got, err := fetchAndMap(
		context.Background(),
		func(_ context.Context) (struct{}, error) {
			return struct{}{}, sql.ErrNoRows
		},
		func(_ struct{}) *custom {
			t.Fatal("mapFunc must not be called on ErrNoRows")
			return nil
		},
	)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil pointer, got %+v", got)
	}
}

// TestFetchAndMapPropagatesNonNoRowsErrors ensures real DB errors are not
// swallowed alongside the ErrNoRows special case.
func TestFetchAndMapPropagatesNonNoRowsErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("connection refused")
	_, err := fetchAndMap(
		context.Background(),
		func(_ context.Context) (struct{}, error) {
			return struct{}{}, sentinel
		},
		func(_ struct{}) *License { return nil },
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error propagated, got %v", err)
	}
}
