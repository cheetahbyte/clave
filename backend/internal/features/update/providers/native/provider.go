package native

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/google/uuid"
)

type Config struct {
	SigningRequired bool   `json:"signingRequired"`
	RolloutStrategy string `json:"rolloutStrategy"`
}

type releaseReadRepo interface {
	LatestPublishedUpdateRelease(ctx context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error)
	ListArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error)
	GetReleasePolicy(ctx context.Context, releaseID uuid.UUID) (db.UpdateReleasePolicy, error)
	GetChannelByProductAndName(ctx context.Context, params db.GetChannelByProductAndNameParams) (db.UpdateChannel, error)
}

type Provider struct {
	repo releaseReadRepo
}

func New(repo releaseReadRepo) *Provider {
	return &Provider{repo: repo}
}

func (p *Provider) Key() update.ProviderKey {
	return update.ProviderClaveNative
}

func (p *Provider) Name() string {
	return "Clave Native"
}

func (p *Provider) Capabilities(ctx context.Context) update.CapabilityMatrix {
	return update.CapabilityMatrix{
		CodeSigning:      update.CapabilityNative,
		StagedRollouts:   update.CapabilityNative,
		MandatoryUpdates: update.CapabilityNative,
		ReleaseNotes:     update.CapabilityNative,
		Channels:         update.CapabilityNative,
		ServerSideChecks: update.CapabilityNative,
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
	var c Config
	_ = json.Unmarshal(cfg.Raw, &c)

	channel := cfg.ChannelID
	if channel == uuid.Nil {
		var err error
		channel, err = p.resolveChannel(ctx, req.ProductID, req.Channel)
		if err != nil {
			return noUpdateDecision(req.CurrentVersion), nil
		}
	}

	release, err := p.repo.LatestPublishedUpdateRelease(ctx, db.LatestPublishedUpdateReleaseParams{
		ProductID: req.ProductID,
		Platform:  string(req.Platform),
		ChannelID: channel,
	})
	if err != nil {
		return noUpdateDecision(req.CurrentVersion), nil
	}

	available := update.CompareVersions(release.Version, req.CurrentVersion) > 0
	kind := update.DecisionNoUpdate
	if available {
		kind = update.DecisionUpdateAvailable
	}

	policy, _ := p.repo.GetReleasePolicy(ctx, release.ID)

	// Default to 100% rollout if no policy row exists
	if policy.RolloutPercentage == 0 && policy.ReleaseID == uuid.Nil {
		policy.RolloutPercentage = 100
	}

	if policy.Mandatory {
		kind = update.DecisionMandatoryUpdate
	}

	if req.ClientID != "" && policy.RolloutPercentage < 100 {
		if !p.isInRollout(req.ClientID, policy.RolloutPercentage) {
			return noUpdateDecision(req.CurrentVersion), nil
		}
	}

	artifacts, _ := p.repo.ListArtifactsForRelease(ctx, release.ID)

	var fullArtifact *update.ArtifactDTO

	for _, a := range artifacts {
		// Skip artifacts that don't match the requested arch/os.
		if !matchesArch(req.Arch, a.Arch) || !matchesOS(string(req.Platform), a.Os) {
			continue
		}

		md := parseArtifactMetadata(a.Metadata)

		dto := update.ArtifactDTO{
			Type:      a.ArtifactType,
			URL:       a.Url,
			Arch:      a.Arch,
			OS:        a.Os,
			Signature: "",
			Metadata:  md,
		}
		if a.SizeBytes != nil {
			dto.SizeBytes = *a.SizeBytes
		}
		if a.ChecksumSha256 != nil {
			dto.SHA256 = *a.ChecksumSha256
		}
		if a.Signature != nil {
			dto.Signature = *a.Signature
		}

		if fullArtifact == nil {
			copy1 := dto
			fullArtifact = &copy1
		}
	}

	artifactDTOs := make([]update.ArtifactDTO, 0, 1)
	var downloadURL string

	if fullArtifact != nil {
		artifactDTOs = append(artifactDTOs, *fullArtifact)
		downloadURL = fullArtifact.URL
	}

	var releaseNotes string
	if release.ReleaseNotes != nil {
		releaseNotes = *release.ReleaseNotes
	}
	var changelog string
	if release.Changelog != nil {
		changelog = *release.Changelog
	}

	decision := &update.UpdateDecision{
		Kind:            kind,
		CurrentVersion:  req.CurrentVersion,
		LatestVersion:   release.Version,
		UpdateAvailable: available,
		DownloadURL:     downloadURL,
		ReleaseNotes:    releaseNotes,
		Changelog:       changelog,
		Artifacts:       artifactDTOs,
		Metadata: map[string]any{
			"release_id": release.ID.String(),
		},
	}

	return decision, nil
}

func (p *Provider) resolveChannel(ctx context.Context, productID uuid.UUID, channelName string) (uuid.UUID, error) {
	ch, err := p.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channelName,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return ch.ID, nil
}

func noUpdateDecision(currentVersion string) *update.UpdateDecision {
	return &update.UpdateDecision{
		Kind:            update.DecisionNoUpdate,
		CurrentVersion:  currentVersion,
		LatestVersion:   currentVersion,
		UpdateAvailable: false,
	}
}

func (p *Provider) isInRollout(clientID string, percentage int32) bool {
	if clientID == "" {
		return percentage >= 100
	}
	hash := fnvHash(clientID)
	return hash%100 < uint32(percentage)
}

func fnvHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func matchesArch(requested, artifact string) bool {
	if requested == "" || artifact == "" {
		return true
	}
	ra := strings.ToLower(requested)
	aa := strings.ToLower(artifact)
	if ra == aa {
		return true
	}
	if artifact == "universal" {
		return true
	}
	return false
}

func matchesOS(requested, artifact string) bool {
	if requested == "" || artifact == "" {
		return true
	}
	return strings.EqualFold(requested, artifact)
}

func parseArtifactMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var md map[string]any
	if err := json.Unmarshal(raw, &md); err != nil {
		return nil
	}
	return md
}
