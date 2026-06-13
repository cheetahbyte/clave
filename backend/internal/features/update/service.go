package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	problem "github.com/cheetahbyte/problems"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultChannel = "stable"
const defaultPlatform = "macos"

type Service struct {
	licenses       *license.Service
	signer         *signing.Service
	repo           *Repository
	registry       *ProviderRegistry
	publicAppURL   string
	storagePath    string
	defaultStorage storage.Backend
}

func NewService(
	licenses *license.Service,
	signer *signing.Service,
	repo *Repository,
	registry *ProviderRegistry,
	publicAppURL string,
	storagePath string,
) *Service {
	return &Service{
		licenses:       licenses,
		signer:         signer,
		repo:           repo,
		registry:       registry,
		publicAppURL:   publicAppURL,
		storagePath:    storagePath,
		defaultStorage: storage.NewLocal(storagePath),
	}
}

// storageForProduct resolves the storage backend for a product. Products
// without an explicit storage config fall back to the server's default local
// backend. The returned Kind identifies which backend was selected so it can
// be recorded on uploaded artifacts.
func (svc *Service) storageForProduct(ctx context.Context, productID uuid.UUID) (storage.Backend, storage.Kind, error) {
	cfg, err := svc.repo.GetProductStorageConfig(ctx, productID)
	if err != nil {
		// No per-product config: use the default local backend.
		return svc.defaultStorage, storage.KindLocal, nil
	}

	kind := storage.Kind(cfg.Backend)
	if kind == storage.KindLocal && len(cfg.Config) <= 2 {
		// Local backend without an explicit base_path: use the server default.
		return svc.defaultStorage, storage.KindLocal, nil
	}

	backend, err := storage.New(kind, cfg.Config)
	if err != nil {
		return nil, "", fmt.Errorf("build storage backend %q: %w", kind, err)
	}
	return backend, kind, nil
}

func (svc *Service) Check(ctx context.Context, data CheckRequest) (CheckResponse, error) {
	instance := "/updates/check"

	claims, err := svc.signer.ParseJWT(data.Token)
	if err != nil {
		return CheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseID, err := license.LicenseIDFromSubject(claims.Subject)
	if err != nil {
		return CheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	lic, err := svc.licenses.GetByID(ctx, licenseID)
	if err != nil || lic == nil {
		return CheckResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !lic.Active {
		return CheckResponse{}, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Instance(instance))
	}

	platform := strings.TrimSpace(data.Platform)
	if platform == "" {
		platform = defaultPlatform
	}
	channel := strings.TrimSpace(data.Channel)
	if channel == "" {
		channel = defaultChannel
	}

	// Feature-gated channels: a license may only pull updates from a channel
	// when it holds every feature the channel requires.
	if ch, chErr := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: claims.ProductID,
		Name:      channel,
	}); chErr == nil {
		if !hasAllFeatures(lic.Features, ch.RequiredFeatures) {
			return CheckResponse{}, problem.Of(403).
				Append(problem.Title("Channel not available")).
				Append(problem.Detail("This license does not have access to the requested update channel.")).
				Append(problem.Instance(instance))
		}
	}

	updateReq := UpdateRequest{
		LicenseID:      licenseID.String(),
		ProductID:      claims.ProductID,
		Platform:       Platform(platform),
		Channel:        channel,
		CurrentVersion: data.Version,
		CurrentBuild:   data.Build,
		Arch:           data.Arch,
		OSVersion:      data.OSVersion,
		ClientID:       data.ClientID,
	}

	dbConfig, dbErr := svc.repo.GetProductUpdateConfig(ctx, claims.ProductID, platform, channel)
	if dbErr != nil {
		return CheckResponse{}, problem.Of(500).
			Append(problem.Title("Update engine not configured")).
			Append(problem.Detail("No update configuration found for this product, platform, and channel.")).
			Append(problem.Instance(instance))
	}

	provider, ok := svc.registry.Get(ProviderKey(dbConfig.ProviderKey))
	if !ok {
		slog.Error("update provider is not registered", "provider", dbConfig.ProviderKey, "product", claims.ProductID)
		return CheckResponse{}, problem.Of(500).
			Append(problem.Title("Update engine not configured")).
			Append(problem.Detail("The configured update provider is not registered.")).
			Append(problem.Instance(instance))
	}

	providerConfig := ProviderConfig{
		OrganizationID: dbConfig.OrganizationID,
		ProductID:      dbConfig.ProductID,
		Platform:       Platform(platform),
		Channel:        channel,
		Raw:            dbConfig.Config,
	}

	decision, err := provider.CheckForUpdate(ctx, updateReq, providerConfig)
	if err != nil {
		slog.Error("update provider check failed", "provider", provider.Key(), "err", err)
		return CheckResponse{}, problem.Of(500).
			Append(problem.Title("Update check failed")).
			Append(problem.Instance(instance))
	}

	var releaseID *uuid.UUID
	if decision.Metadata != nil {
		if rid, ok := decision.Metadata["release_id"].(string); ok {
			if parsed, parseErr := uuid.Parse(rid); parseErr == nil {
				releaseID = &parsed
			}
		}
	}

	if releaseID != nil && decision.Changelog != "" {
		decision.ChangelogURL = svc.ChangelogURL(*releaseID)
	}

	_ = svc.repo.InsertUpdateCheck(ctx, dbConfig.OrganizationID, claims.ProductID, licenseID,
		platform, channel, string(provider.Key()),
		data.Version, data.Build, data.Arch, data.OSVersion,
		string(decision.Kind), releaseID,
	)

	return CheckResponse{
		CurrentVersion:  decision.CurrentVersion,
		LatestVersion:   decision.LatestVersion,
		UpdateAvailable: decision.UpdateAvailable,
		DownloadURL:     decision.DownloadURL,
		Kind:            string(decision.Kind),
		ReleaseNotes:    decision.ReleaseNotes,
		Changelog:       decision.Changelog,
		ChangelogURL:    decision.ChangelogURL,
		Artifacts:       decision.Artifacts,
		Metadata:        decision.Metadata,
	}, nil
}

func (svc *Service) VerifyProductOwnership(ctx context.Context, orgID, productID uuid.UUID) error {
	_, err := svc.repo.GetProductByIDAndOrganization(ctx, orgID, productID)
	return err
}

func (svc *Service) ListProviders(ctx context.Context) []ProviderInfo {
	return svc.registry.List(ctx)
}

func (svc *Service) GetProductUpdateConfigs(ctx context.Context, productID uuid.UUID) ([]ProductUpdateConfigDTO, error) {
	rows, err := svc.repo.GetProductUpdateConfigs(ctx, productID)
	if err != nil {
		return nil, err
	}

	result := make([]ProductUpdateConfigDTO, len(rows))
	for i, r := range rows {
		result[i] = ProductUpdateConfigDTO{
			ID:          r.ID.String(),
			ProductID:   r.ProductID.String(),
			Platform:    r.Platform,
			Channel:     r.ChannelName,
			ChannelID:   r.ChannelID.String(),
			ProviderKey: r.ProviderKey,
			Enabled:     r.Enabled,
			Config:      MustParseProviderConfig(r.Config),
			FeedURL:     svc.FeedURL(r.ProviderKey, r.ProductID, r.Platform, r.ChannelName),
		}
	}
	return result, nil
}

func (svc *Service) SaveProductUpdateConfig(ctx context.Context, orgID, productID uuid.UUID, req SaveProductUpdateConfigRequest) (*ProductUpdateConfigDTO, error) {
	if req.ProviderKey != string(ProviderSparkle) && req.ProviderKey != string(ProviderClaveNative) {
		return nil, fmt.Errorf("unsupported provider key: %s", req.ProviderKey)
	}

	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))

	ch, err := svc.repo.UpsertUpdateChannel(ctx, db.UpsertUpdateChannelParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Name:           req.Channel,
		IsDefault:      false,
	})
	if err != nil {
		return nil, err
	}

	// Normalize config for Clave-managed providers
	normalizedConfig := map[string]any{}
	switch ProviderKey(req.ProviderKey) {
	case ProviderSparkle:
		normalizedConfig = map[string]any{"delivery": "sparkle"}
	case ProviderClaveNative:
		normalizedConfig = map[string]any{"delivery": "clave_native"}
	}

	raw, _ := json.Marshal(normalizedConfig)

	cfg, err := svc.repo.UpsertProductUpdateConfig(ctx, db.UpsertProductUpdateConfigParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Platform:       req.Platform,
		ChannelID:      ch.ID,
		ProviderKey:    req.ProviderKey,
		Config:         raw,
		Enabled:        req.Enabled,
	})
	if err != nil {
		return nil, err
	}

	return &ProductUpdateConfigDTO{
		ID:          cfg.ID.String(),
		ProductID:   cfg.ProductID.String(),
		Platform:    cfg.Platform,
		Channel:     ch.Name,
		ChannelID:   ch.ID.String(),
		ProviderKey: cfg.ProviderKey,
		Enabled:     cfg.Enabled,
		Config:      MustParseProviderConfig(cfg.Config),
		FeedURL:     svc.FeedURL(cfg.ProviderKey, cfg.ProductID, cfg.Platform, ch.Name),
	}, nil
}

func (svc *Service) DeleteProductUpdateConfig(ctx context.Context, orgID, configID uuid.UUID) error {
	_, err := svc.repo.DeleteProductUpdateConfig(ctx, configID, orgID)
	return err
}

func (svc *Service) AppcastURL(productID uuid.UUID, platform, channel string) string {
	if svc.publicAppURL == "" {
		return fmt.Sprintf("/api/v1/updates/products/%s/%s/%s/appcast.xml", productID, platform, channel)
	}
	return fmt.Sprintf("%s/api/v1/updates/products/%s/%s/%s/appcast.xml", strings.TrimRight(svc.publicAppURL, "/"), productID, platform, channel)
}

// FeedURL returns the public feed URL for a configured source, dependent on
// the delivery protocol: Sparkle serves the appcast XML, Clave Native serves
// the custom JSON feed.
func (svc *Service) FeedURL(providerKey string, productID uuid.UUID, platform, channel string) string {
	if providerKey == string(ProviderClaveNative) {
		return svc.nativeFeedURL(productID, platform, channel)
	}
	return svc.AppcastURL(productID, platform, channel)
}

func (svc *Service) nativeFeedURL(productID uuid.UUID, platform, channel string) string {
	path := fmt.Sprintf("/api/v1/updates/products/%s/%s/%s/feed.json", productID, platform, channel)
	if svc.publicAppURL == "" {
		return path
	}
	return strings.TrimRight(svc.publicAppURL, "/") + path
}

func (svc *Service) NativeUpdateURL() string {
	if svc.publicAppURL == "" {
		return "/api/v1/client/updates/check"
	}
	return fmt.Sprintf("%s/api/v1/client/updates/check", strings.TrimRight(svc.publicAppURL, "/"))
}

func (svc *Service) ChangelogURL(releaseID uuid.UUID) string {
	path := fmt.Sprintf("/api/v1/updates/releases/%s/changelog.html", releaseID)
	if svc.publicAppURL == "" {
		return path
	}
	return strings.TrimRight(svc.publicAppURL, "/") + path
}

// AuthorizeChannelAccess gates the public native feed for feature-gated
// channels. Open channels are always allowed; gated channels require a valid
// license token whose features satisfy the channel's requirements.
func (svc *Service) AuthorizeChannelAccess(ctx context.Context, productID uuid.UUID, channel, token string) error {
	ch, err := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channel,
	})
	if err != nil {
		// Unknown channel: nothing to gate, let feed generation 404.
		return nil
	}
	if len(ch.RequiredFeatures) == 0 {
		return nil
	}
	if token == "" {
		return fmt.Errorf("license token required for this channel")
	}
	claims, err := svc.signer.ParseJWT(token)
	if err != nil {
		return fmt.Errorf("invalid license token")
	}
	if claims.ProductID != productID {
		return fmt.Errorf("token not valid for this product")
	}
	if !hasAllFeatures(claims.Features, ch.RequiredFeatures) {
		return fmt.Errorf("license lacks required features for this channel")
	}
	return nil
}

// RenderReleaseChangelog returns the sanitized HTML changelog for a release.
func (svc *Service) RenderReleaseChangelog(ctx context.Context, releaseID uuid.UUID) ([]byte, error) {
	release, err := svc.repo.GetUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	body, _ := svc.releaseChangelog(ctx, release)
	if body == "" {
		return nil, fmt.Errorf("no changelog for release")
	}
	title := release.Version
	if product, perr := svc.licenses.GetProductByID(ctx, release.ProductID); perr == nil {
		title = product.Name + " " + release.Version
	}
	return RenderChangelogHTML(title, body)
}

func (svc *Service) GenerateAppcast(ctx context.Context, productID uuid.UUID, platform, channel string) ([]byte, error) {
	ch, err := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channel,
	})
	if err != nil {
		return nil, err
	}

	releases, err := svc.repo.ListPublishedReleasesForAppcast(ctx, productID, platform, ch.ID)
	if err != nil {
		return nil, err
	}

	slog.Debug("generating appcast",
		"product", productID,
		"platform", platform,
		"channel", channel,
		"channelID", ch.ID,
		"numReleases", len(releases),
	)
	for _, r := range releases {
		slog.Debug("appcast release", "id", r.ID, "version", r.Version, "status", r.Status, "publishedAt", r.PublishedAt)
	}

	appcastReleases := make([]AppcastRelease, len(releases))
	for i, rel := range releases {
		artifacts, _ := svc.repo.ListArtifactsForRelease(ctx, rel.ID)
		_, changelogURL := svc.releaseChangelog(ctx, rel)
		appcastReleases[i] = AppcastRelease{Release: rel, Artifacts: artifacts, ChangelogURL: changelogURL}
	}

	product, err := svc.licenses.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return GenerateAppcastXML(product, platform, channel, appcastReleases, svc.AppcastURL(productID, platform, channel))
}

func (svc *Service) GenerateNativeFeed(ctx context.Context, productID uuid.UUID, platform, channel string) ([]byte, error) {
	ch, err := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channel,
	})
	if err != nil {
		return nil, err
	}

	releases, err := svc.repo.ListPublishedReleasesForAppcast(ctx, productID, platform, ch.ID)
	if err != nil {
		return nil, err
	}

	inputs := make([]NativeFeedReleaseInput, len(releases))
	for i, rel := range releases {
		artifacts, _ := svc.repo.ListArtifactsForRelease(ctx, rel.ID)
		policy, _ := svc.repo.GetReleasePolicy(ctx, rel.ID)
		changelogBody, changelogURL := svc.releaseChangelog(ctx, rel)
		inputs[i] = NativeFeedReleaseInput{Release: rel, Artifacts: artifacts, Policy: policy, Changelog: changelogBody, ChangelogURL: changelogURL}
	}

	product, err := svc.licenses.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return GenerateNativeFeed(product, platform, channel, inputs)
}

func (svc *Service) ListReleases(ctx context.Context, orgID uuid.UUID, productID *uuid.UUID, limit, offset int32) ([]ReleaseDTO, error) {
	rows, err := svc.repo.ListReleasesForOrganization(ctx, orgID, productID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]ReleaseDTO, len(rows))
	for i, r := range rows {
		artifacts, _ := svc.repo.ListArtifactsForRelease(ctx, r.ID)
		artifactDTOs := make([]ArtifactDTO, len(artifacts))
		for j, a := range artifacts {
			var sig string
			if a.Signature != nil {
				sig = *a.Signature
			}
			artifactDTOs[j] = ArtifactDTO{
				Type:      a.ArtifactType,
				URL:       a.Url,
				Arch:      a.Arch,
				Signature: sig,
			}
			if a.SizeBytes != nil {
				artifactDTOs[j].SizeBytes = *a.SizeBytes
			}
			if a.ChecksumSha256 != nil {
				artifactDTOs[j].SHA256 = *a.ChecksumSha256
			}
			if a.Filename != nil {
				artifactDTOs[j].Filename = *a.Filename
			}
			if a.MimeType != nil {
				artifactDTOs[j].MimeType = *a.MimeType
			}
		}

		var (
			notes        string
			pubAt        *string
			createdAtStr *string
		)
		if r.ReleaseNotes != nil {
			notes = *r.ReleaseNotes
		}
		if r.PublishedAt.Valid {
			s := r.PublishedAt.Time.Format(time.RFC3339)
			pubAt = &s
		}
		if r.CreatedAt.Valid {
			s := r.CreatedAt.Time.Format(time.RFC3339)
			createdAtStr = &s
		}

		var buildNum string
		if r.BuildNumber != nil {
			buildNum = *r.BuildNumber
		}

		var changelogID string
		if r.ChangelogID.Valid {
			changelogID = uuid.UUID(r.ChangelogID.Bytes).String()
		}

		result[i] = ReleaseDTO{
			ID:           r.ID.String(),
			ProductID:    r.ProductID.String(),
			ProductName:  r.ProductName,
			Channel:      r.ChannelName,
			ChannelID:    r.ChannelID.String(),
			Platform:     r.Platform,
			Version:      r.Version,
			BuildNumber:  buildNum,
			Status:       r.Status,
			ReleaseNotes: notes,
			ChangelogID:  changelogID,
			PublishedAt:  pubAt,
			CreatedAt:    createdAtStr,
			Artifacts:    artifactDTOs,
		}
	}
	return result, nil
}

func (svc *Service) CreateRelease(ctx context.Context, orgID uuid.UUID, req CreateReleaseRequest) (*ReleaseDTO, error) {
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("invalid product id")
	}

	if _, err := svc.repo.GetProductByIDAndOrganization(ctx, orgID, productID); err != nil {
		return nil, fmt.Errorf("product not found in organization")
	}

	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))

	ch, err := svc.repo.UpsertUpdateChannel(ctx, db.UpsertUpdateChannelParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Name:           req.Channel,
		IsDefault:      false,
	})
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}

	var buildNum *string
	if req.BuildNumber != "" {
		buildNum = &req.BuildNumber
	}
	var notes *string
	if req.ReleaseNotes != "" {
		notes = &req.ReleaseNotes
	}
	changelogID := pgtype.UUID{}
	if req.ChangelogID != "" {
		parsed, perr := uuid.Parse(req.ChangelogID)
		if perr != nil {
			return nil, fmt.Errorf("invalid changelog id")
		}
		cl, cerr := svc.repo.GetChangelog(ctx, parsed)
		if cerr != nil || cl.ProductID != productID {
			return nil, fmt.Errorf("changelog not found for this product")
		}
		changelogID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	release, err := svc.repo.InsertUpdateRelease(ctx, db.InsertUpdateReleaseParams{
		OrganizationID: orgID,
		ProductID:      productID,
		ChannelID:      ch.ID,
		Platform:       req.Platform,
		Version:        req.Version,
		BuildNumber:    buildNum,
		Status:         "draft",
		ReleaseNotes:   notes,
		ChangelogID:    changelogID,
	})
	if err != nil {
		return nil, fmt.Errorf("insert release: %w", err)
	}

	dto := &ReleaseDTO{
		ID:           release.ID.String(),
		ProductID:    release.ProductID.String(),
		Channel:      req.Channel,
		ChannelID:    ch.ID.String(),
		Platform:     release.Platform,
		Version:      release.Version,
		Status:       release.Status,
		BuildNumber:  req.BuildNumber,
		ReleaseNotes: req.ReleaseNotes,
	}
	return dto, nil
}

func (svc *Service) UploadArtifact(ctx context.Context, releaseID uuid.UUID, reader io.Reader, artifactType, osName, arch string, originalFilename string) (*ArtifactDTOFull, error) {
	release, err := svc.repo.GetUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("release not found: %w", err)
	}

	backend, kind, err := svc.storageForProduct(ctx, release.ProductID)
	if err != nil {
		return nil, fmt.Errorf("resolve storage backend: %w", err)
	}

	artifactID := uuid.New()
	ext := filepath.Ext(originalFilename)
	storedFilename := artifactID.String() + ext
	storageKey := artifactID.String() + "/" + storedFilename

	// Hash the stream as it is written to the backend.
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	sizeBytes, err := backend.Put(ctx, storageKey, tee)
	if err != nil {
		return nil, fmt.Errorf("store artifact: %w", err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	downloadURL := svc.artifactDownloadURL(artifactID)
	mimeType := mimeTypeForArtifact(artifactType)
	backendName := string(kind)
	artifact, err := svc.repo.InsertUpdateArtifact(ctx, db.InsertUpdateArtifactParams{
		ID:                   artifactID,
		ReleaseID:            releaseID,
		ArtifactType:         artifactType,
		Os:                   osName,
		Arch:                 arch,
		Url:                  downloadURL,
		SizeBytes:            &sizeBytes,
		ChecksumSha256:       &checksum,
		Metadata:             []byte(`{}`),
		Filename:             &storedFilename,
		MimeType:             &mimeType,
		MinimumSystemVersion: nil,
		SparkleEdSignature:   nil,
		StorageBackend:       &backendName,
		StorageKey:           &storageKey,
	})
	if err != nil {
		_ = backend.Delete(ctx, storageKey)
		return nil, fmt.Errorf("insert artifact: %w", err)
	}

	return &ArtifactDTOFull{
		ID:             artifact.ID.String(),
		ReleaseID:      artifact.ReleaseID.String(),
		ArtifactType:   artifact.ArtifactType,
		OS:             artifact.Os,
		Arch:           artifact.Arch,
		URL:            artifact.Url,
		SizeBytes:      artifact.SizeBytes,
		ChecksumSHA256: artifact.ChecksumSha256,
		Signature:      artifact.Signature,
	}, nil
}

func (svc *Service) PublishRelease(ctx context.Context, releaseID uuid.UUID) (*ReleaseDTO, error) {
	artifacts, err := svc.repo.ListArtifactsForRelease(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to check artifacts: %w", err)
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("cannot publish a release with no artifacts")
	}

	release, err := svc.repo.PublishUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	var notes string
	if release.ReleaseNotes != nil {
		notes = *release.ReleaseNotes
	}
	var buildNum string
	if release.BuildNumber != nil {
		buildNum = *release.BuildNumber
	}

	return &ReleaseDTO{
		ID:           release.ID.String(),
		ProductID:    release.ProductID.String(),
		Platform:     release.Platform,
		Version:      release.Version,
		BuildNumber:  buildNum,
		Status:       release.Status,
		ReleaseNotes: notes,
	}, nil
}

func (svc *Service) YankRelease(ctx context.Context, releaseID uuid.UUID) (*ReleaseDTO, error) {
	release, err := svc.repo.YankUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	var notes string
	if release.ReleaseNotes != nil {
		notes = *release.ReleaseNotes
	}
	var buildNum string
	if release.BuildNumber != nil {
		buildNum = *release.BuildNumber
	}

	return &ReleaseDTO{
		ID:           release.ID.String(),
		ProductID:    release.ProductID.String(),
		Platform:     release.Platform,
		Version:      release.Version,
		BuildNumber:  buildNum,
		Status:       release.Status,
		ReleaseNotes: notes,
	}, nil
}

func (svc *Service) GetArtifact(ctx context.Context, artifactID uuid.UUID) (db.UpdateArtifact, error) {
	return svc.repo.GetArtifact(ctx, artifactID)
}

// OpenArtifactDownload resolves the storage backend for an artifact and
// returns a reader for its bytes along with size and mime type for response
// headers. Caller must close the reader.
func (svc *Service) OpenArtifactDownload(ctx context.Context, artifactID uuid.UUID) (io.ReadCloser, int64, string, error) {
	artifact, err := svc.repo.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, 0, "", err
	}

	release, err := svc.repo.GetUpdateRelease(ctx, artifact.ReleaseID)
	if err != nil {
		return nil, 0, "", err
	}

	backend, _, err := svc.storageForProduct(ctx, release.ProductID)
	if err != nil {
		return nil, 0, "", err
	}

	key := svc.artifactStorageKey(artifact)
	rc, err := backend.Open(ctx, key)
	if err != nil {
		return nil, 0, "", err
	}

	var size int64
	if artifact.SizeBytes != nil {
		size = *artifact.SizeBytes
	}
	mimeType := "application/octet-stream"
	if artifact.MimeType != nil && *artifact.MimeType != "" {
		mimeType = *artifact.MimeType
	}
	return rc, size, mimeType, nil
}

// artifactStorageKey returns the storage key for an artifact, deriving the
// legacy layout for older artifacts uploaded before storage keys were tracked.
func (svc *Service) artifactStorageKey(artifact db.UpdateArtifact) string {
	if artifact.StorageKey != nil && *artifact.StorageKey != "" {
		return *artifact.StorageKey
	}
	filename := artifact.ID.String()
	if artifact.Filename != nil && *artifact.Filename != "" {
		filename = *artifact.Filename
	}
	return artifact.ID.String() + "/" + filename
}

func (svc *Service) DeleteRelease(ctx context.Context, releaseID uuid.UUID) error {
	_, err := svc.repo.DeleteUpdateRelease(ctx, releaseID)
	return err
}

func (svc *Service) artifactDownloadURL(artifactID uuid.UUID) string {
	base := svc.publicAppURL
	if base == "" {
		base = ""
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		return fmt.Sprintf("/api/v1/updates/artifacts/%s/download", artifactID)
	}
	return fmt.Sprintf("%s/api/v1/updates/artifacts/%s/download", base, artifactID)
}
