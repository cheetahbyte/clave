package update

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/google/uuid"
)

func TestDeltaArtifactCompatibility(t *testing.T) {
	size := int64(100)
	sha := strings.Repeat("a", 64)
	source := db.UpdateArtifact{ID: uuid.New(), ArtifactType: "zip", Os: "macos", Arch: "arm64", SizeBytes: &size, ChecksumSha256: &sha}
	target := source
	target.ID = uuid.New()
	if !deltaEligibleArtifact(source) || artifactCompatibilityKey(source) != artifactCompatibilityKey(target) {
		t.Fatal("matching full artifacts should be eligible")
	}
	target.Arch = "x86_64"
	if artifactCompatibilityKey(source) == artifactCompatibilityKey(target) {
		t.Fatal("different architectures must not pair")
	}
	source.ArtifactType = "delta"
	if deltaEligibleArtifact(source) {
		t.Fatal("delta artifacts must not become delta sources")
	}
}

func TestBoundedDeltaReason(t *testing.T) {
	if got := boundedDeltaReason("   "); got != "unspecified" {
		t.Fatalf("empty reason = %q", got)
	}
	if got := boundedDeltaReason(strings.Repeat("x", deltaReasonLimit+20)); len(got) != deltaReasonLimit {
		t.Fatalf("bounded reason length = %d", len(got))
	}
}

func TestStoreDeltaAndPersistCleansUpAfterPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	backend := storage.NewLocal(root)
	key := "artifact/patch.delta"
	persistErr := errors.New("database transaction failed")

	_, err := storeDeltaAndPersist(t.Context(), backend, key, bytes.NewReader([]byte("patch")), func() (db.UpdateDeltaJob, error) {
		return db.UpdateDeltaJob{}, persistErr
	})
	if !errors.Is(err, persistErr) {
		t.Fatalf("error = %v, want %v", err, persistErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(key))); !os.IsNotExist(statErr) {
		t.Fatalf("stored patch was not cleaned up: %v", statErr)
	}
}
