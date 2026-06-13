package sparkle

import (
	"context"
	"encoding/json"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/google/uuid"
)

type Config struct {
	Ed25519PublicKey string `json:"ed25519PublicKey,omitempty"`
}

type Provider struct {
	repo interface {
		LatestPublishedUpdateRelease(ctx context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error)
		ListArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error)
	}
}

func New(repo interface {
	LatestPublishedUpdateRelease(ctx context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error)
	ListArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error)
}) *Provider {
	return &Provider{repo: repo}
}

func (p *Provider) Key() update.ProviderKey {
	return update.ProviderSparkle
}

func (p *Provider) Name() string {
	return "Sparkle"
}

func (p *Provider) Capabilities(ctx context.Context) update.CapabilityMatrix {
	return update.CapabilityMatrix{
		DeltaUpdates:     update.CapabilityNative,
		CodeSigning:      update.CapabilityNative,
		StagedRollouts:   update.CapabilityUnsupported,
		MandatoryUpdates: update.CapabilityUnsupported,
		ReleaseNotes:     update.CapabilityNative,
		Channels:         update.CapabilityNative,
		ServerSideChecks: update.CapabilityUnsupported,
	}
}

func (p *Provider) ValidateConfig(ctx context.Context, cfg update.ProviderConfig) error {
	var c Config
	if err := json.Unmarshal(cfg.Raw, &c); err != nil {
		return err
	}
	return nil
}

func (p *Provider) CheckForUpdate(ctx context.Context, req update.UpdateRequest, cfg update.ProviderConfig) (*update.UpdateDecision, error) {
	channelID := uuid.Nil

	release, err := p.repo.LatestPublishedUpdateRelease(ctx, db.LatestPublishedUpdateReleaseParams{
		ProductID: req.ProductID,
		Platform:  string(req.Platform),
		ChannelID: channelID,
	})
	if err != nil {
		return &update.UpdateDecision{Kind: update.DecisionNoUpdate}, nil
	}

	artifacts, err := p.repo.ListArtifactsForRelease(ctx, release.ID)
	if err != nil {
		return &update.UpdateDecision{Kind: update.DecisionNoUpdate}, nil
	}

	var selectedArtifact *db.UpdateArtifact
	for _, a := range artifacts {
		if a.ArtifactType == "dmg" || a.ArtifactType == "zip" || a.ArtifactType == "pkg" {
			selectedArtifact = &a
			break
		}
	}

	if selectedArtifact == nil {
		return &update.UpdateDecision{Kind: update.DecisionNoUpdate}, nil
	}

	updateAvailable := update.CompareVersions(release.Version, req.CurrentVersion) > 0
	if !updateAvailable {
		return &update.UpdateDecision{Kind: update.DecisionNoUpdate}, nil
	}

	var artifactsDTO []update.ArtifactDTO
	if selectedArtifact != nil {
		sa := *selectedArtifact
		artifactsDTO = append(artifactsDTO, update.ArtifactDTO{
			Type:      sa.ArtifactType,
			URL:       sa.Url,
			SizeBytes: *sa.SizeBytes,
			SHA256:    *sa.ChecksumSha256,
			Signature: *sa.Signature,
		})
	}

	var releaseNotes string
	if release.ReleaseNotes != nil {
		releaseNotes = *release.ReleaseNotes
	}
	var changelog string
	if release.Changelog != nil {
		changelog = *release.Changelog
	}

	metadata := map[string]any{
		"release_id": release.ID.String(),
		"version":    release.Version,
		"build":      release.BuildNumber,
	}

	return &update.UpdateDecision{
		Kind:            update.DecisionUpdateAvailable,
		CurrentVersion:  req.CurrentVersion,
		LatestVersion:   release.Version,
		UpdateAvailable: true,
		DownloadURL:     selectedArtifact.Url,
		ReleaseNotes:    releaseNotes,
		Changelog:       changelog,
		Artifacts:       artifactsDTO,
		Metadata:        metadata,
	}, nil
}
