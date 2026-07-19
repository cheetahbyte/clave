package native

import (
	"context"
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/google/uuid"
)

type testRepo struct {
	channelLookups int
	latestParams   db.LatestPublishedUpdateReleaseParams
	release        db.UpdateRelease
}

func (r *testRepo) LatestPublishedUpdateRelease(_ context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error) {
	r.latestParams = params
	return r.release, nil
}
func (*testRepo) ListArtifactsForRelease(context.Context, uuid.UUID) ([]db.UpdateArtifact, error) {
	return nil, nil
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
