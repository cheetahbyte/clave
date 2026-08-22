package update

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestReuseArtifactsIntegration(t *testing.T) {
	tx, q := openDeltaIntegrationTx(t)
	orgID, productID, channelID := seedDeltaParents(t, tx)
	sourceID, targetID := uuid.New(), uuid.New()
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_releases (id, organization_id, product_id, channel_id, platform, version, status, published_at)
		VALUES ($1, $2, $3, $4, 'macos', '1.0.0', 'published', now()),
		       ($5, $2, $3, $4, 'macos', '1.0.0', 'draft', NULL)`,
		sourceID, orgID, productID, channelID, targetID); err != nil {
		t.Fatalf("insert releases: %v", err)
	}

	fullID, deltaID := uuid.New(), uuid.New()
	filename := "app.zip"
	checksum := strings.Repeat("a", 64)
	size := int64(42)
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_artifacts (id, release_id, artifact_type, os, arch, url, size_bytes, checksum_sha256, filename)
		VALUES ($1, $2, 'zip', 'macos', 'universal', 'full', $3, $4, $5),
		       ($6, $2, 'delta', 'macos', 'universal', 'delta', 1, $4, 'patch.delta')`,
		fullID, sourceID, size, checksum, filename, deltaID); err != nil {
		t.Fatalf("insert source artifacts: %v", err)
	}

	svc := NewService(nil, nil, NewRepository(q, tx), nil, "http://clave.test", t.TempDir())
	artifacts, err := svc.ReuseArtifacts(t.Context(), targetID, sourceID)
	if err != nil {
		t.Fatalf("reuse artifacts: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ArtifactType != "zip" {
		t.Fatalf("reused artifacts = %#v, want one full artifact", artifacts)
	}

	stored, err := q.ListArtifactsForRelease(t.Context(), targetID)
	if err != nil {
		t.Fatalf("list target artifacts: %v", err)
	}
	wantKey := fullID.String() + "/" + filename
	if len(stored) != 1 || stored[0].StorageKey == nil || *stored[0].StorageKey != wantKey {
		t.Fatalf("stored artifacts = %#v, want shared key %q", stored, wantKey)
	}

	_, err = svc.ReuseArtifacts(t.Context(), targetID, sourceID)
	if !errors.Is(err, ErrReleaseAlreadyHasArtifacts) {
		t.Fatalf("second reuse error = %v, want %v", err, ErrReleaseAlreadyHasArtifacts)
	}
}

func TestReuseArtifactsRejectsInvalidReleasesIntegration(t *testing.T) {
	tx, q := openDeltaIntegrationTx(t)
	orgID, productID, channelID := seedDeltaParents(t, tx)
	sourceID, publishedTargetID, otherPlatformID := uuid.New(), uuid.New(), uuid.New()
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_releases (id, organization_id, product_id, channel_id, platform, version, status, published_at)
		VALUES ($1, $2, $3, $4, 'macos', '1.0.0', 'published', now()),
		       ($5, $2, $3, $4, 'macos', '1.1.0', 'published', now()),
		       ($6, $2, $3, $4, 'windows', '1.1.0', 'draft', NULL)`,
		sourceID, orgID, productID, channelID, publishedTargetID, otherPlatformID); err != nil {
		t.Fatalf("insert releases: %v", err)
	}
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_artifacts (id, release_id, artifact_type, os, arch, url)
		VALUES ($1, $2, 'zip', 'macos', 'universal', 'full')`, uuid.New(), sourceID); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	svc := NewService(nil, nil, NewRepository(q, tx), nil, "http://clave.test", t.TempDir())
	if _, err := svc.ReuseArtifacts(t.Context(), publishedTargetID, sourceID); err == nil || !strings.Contains(err.Error(), "draft") {
		t.Fatalf("published target error = %v, want draft error", err)
	}
	if _, err := svc.ReuseArtifacts(t.Context(), otherPlatformID, sourceID); err == nil || !strings.Contains(err.Error(), "same product and platform") {
		t.Fatalf("platform mismatch error = %v", err)
	}
	if _, err := svc.ReuseArtifacts(t.Context(), sourceID, sourceID); err == nil {
		t.Fatal("reusing a release onto itself succeeded")
	}

}
