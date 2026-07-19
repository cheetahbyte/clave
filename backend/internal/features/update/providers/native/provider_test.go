package native

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/pkg/delta"
	"github.com/google/uuid"
)

type testRepo struct {
	channelLookups int
	latestParams   db.LatestPublishedUpdateReleaseParams
	release        db.UpdateRelease
	artifacts      []db.UpdateArtifact
}

func (r *testRepo) LatestPublishedUpdateRelease(_ context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error) {
	r.latestParams = params
	return r.release, nil
}
func (r *testRepo) ListArtifactsForRelease(context.Context, uuid.UUID) ([]db.UpdateArtifact, error) {
	return r.artifacts, nil
}

func TestCheckForUpdateReturnsApplicableDeltaAfterFullArtifact(t *testing.T) {
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte("artifact")))
	fullType, deltaType := "zip", "delta"
	size := int64(1000)
	signature := "signed"
	contract := delta.Contract{Schema: delta.Schema, Algorithm: delta.Algorithm, FromVersion: "1.0.0", ToVersion: "2.0.0", BaseSHA256: digest, TargetSHA256: digest, PatchSHA256: digest, TargetSize: size}
	metadata, _ := json.Marshal(contract)
	repo := &testRepo{
		release: db.UpdateRelease{ID: uuid.New(), Version: "2.0.0"},
		artifacts: []db.UpdateArtifact{
			{ID: uuid.New(), ArtifactType: deltaType, Os: "macos", Arch: "arm64", Url: "/delta", SizeBytes: &size, ChecksumSha256: &digest, Signature: &signature, Metadata: metadata},
			{ID: uuid.New(), ArtifactType: fullType, Os: "macos", Arch: "arm64", Url: "/full", SizeBytes: &size, ChecksumSha256: &digest},
		},
	}
	decision, err := New(repo).CheckForUpdate(context.Background(), update.UpdateRequest{ProductID: uuid.New(), Platform: "macos", CurrentVersion: "1.0.0", Arch: "arm64"}, update.ProviderConfig{ChannelID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Artifacts) != 2 || decision.Artifacts[0].URL != "/full" || decision.Artifacts[1].URL != "/delta" {
		t.Fatalf("expected full then delta, got %#v", decision.Artifacts)
	}
	decision, err = New(repo).CheckForUpdate(context.Background(), update.UpdateRequest{ProductID: uuid.New(), Platform: "macos", CurrentVersion: "1.5.0", Arch: "arm64"}, update.ProviderConfig{ChannelID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Artifacts) != 1 || decision.Artifacts[0].URL != "/full" {
		t.Fatalf("expected full-only fallback, got %#v", decision.Artifacts)
	}
}
func (*testRepo) GetReleasePolicy(context.Context, uuid.UUID) (db.UpdateReleasePolicy, error) {
	return db.UpdateReleasePolicy{}, nil
}
func (r *testRepo) GetChannelByProductAndName(context.Context, db.GetChannelByProductAndNameParams) (db.UpdateChannel, error) {
	r.channelLookups++
	return db.UpdateChannel{}, nil
}

func TestCheckForUpdateUsesConfiguredChannelID(t *testing.T) {
	channelID := uuid.New()
	repo := &testRepo{release: db.UpdateRelease{ID: uuid.New(), Version: "2.0.0"}}
	decision, err := New(repo).CheckForUpdate(context.Background(), update.UpdateRequest{
		ProductID:      uuid.New(),
		Platform:       "macos",
		Channel:        "stable",
		CurrentVersion: "1.0.0",
	}, update.ProviderConfig{ChannelID: channelID})
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if repo.channelLookups != 0 {
		t.Fatalf("expected no channel lookup, got %d", repo.channelLookups)
	}
	if repo.latestParams.ChannelID != channelID {
		t.Fatalf("expected configured channel ID %s, got %s", channelID, repo.latestParams.ChannelID)
	}
	if !decision.UpdateAvailable {
		t.Fatal("expected update decision")
	}
}
