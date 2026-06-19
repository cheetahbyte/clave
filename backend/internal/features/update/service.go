package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	problem "github.com/cheetahbyte/problems"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	sparkleSigner  sparkleSigner
}

type sparkleSigner struct {
	hasKeys    bool
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func (s sparkleSigner) sign(data []byte) string {
	if !s.hasKeys {
		return ""
	}
	sig := ed25519.Sign(s.privateKey, data)
	return base64.StdEncoding.EncodeToString(sig)
}

func (s sparkleSigner) publicEDKey() string {
	if !s.hasKeys {
		return ""
	}
	return base64.StdEncoding.EncodeToString(s.publicKey)
}

func newSparkleSigner(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) sparkleSigner {
	hasKeys := len(publicKey) == ed25519.PublicKeySize && len(privateKey) == ed25519.PrivateKeySize
	return sparkleSigner{
		hasKeys:    hasKeys,
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func NewService(
	licenses *license.Service,
	signer *signing.Service,
	repo *Repository,
	registry *ProviderRegistry,
	publicAppURL string,
	storagePath string,
	sparklePublicKey ed25519.PublicKey,
	sparklePrivateKey ed25519.PrivateKey,
) *Service {
	return &Service{
		licenses:       licenses,
		signer:         signer,
		repo:           repo,
		registry:       registry,
		publicAppURL:   publicAppURL,
		storagePath:    storagePath,
		defaultStorage: storage.NewLocal(storagePath),
		sparkleSigner:  newSparkleSigner(sparklePublicKey, sparklePrivateKey),
	}
}

// storageForProduct resolves the storage backend for a product. Products
// without an explicit storage config fall back to the server's default local
// backend. The returned Kind identifies which backend was selected so it can
// be recorded on uploaded artifacts.

// SparkleSign returns the base64 Ed25519 signature of data using the configured
// Sparkle keypair. Returns an empty string if no keypair is configured.
func (svc *Service) SparkleSign(data []byte) string {
	return svc.sparkleSigner.sign(data)
}

// SparklePublicEDKey returns the base64 Sparkle Ed25519 public key for
// SUPublicEDKey, or an empty string if no keypair is configured.
func (svc *Service) SparklePublicEDKey() string {
	return svc.sparkleSigner.publicEDKey()
}

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

func (svc *Service) activeActivationID(ctx context.Context, licenseID uuid.UUID, claims *signing.LicenseClaims) (uuid.UUID, error) {
	hwidHash := svc.signer.HMACSign(claims.HWID, signing.DontNormalizeKey)
	if claims.ActivationID != uuid.Nil {
		act, err := svc.repo.GetActiveActivationByID(ctx, db.GetActiveActivationByIDParams{
			ActivationID: claims.ActivationID,
			LicenseID:    licenseID,
			HwidHash:     hwidHash,
		})
		if err != nil {
			return uuid.Nil, err
		}
		return act.ID, nil
	}

	act, err := svc.repo.GetActiveActivationByLicenseAndHwidHash(ctx, db.GetActiveActivationByLicenseAndHwidHashParams{
		LicenseID: licenseID,
		HwidHash:  hwidHash,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return act.ID, nil
}

func (svc *Service) parseActiveLicenseToken(ctx context.Context, token string, instance string) (uuid.UUID, *license.License, *signing.LicenseClaims, error) {
	claims, err := svc.signer.ParseJWT(token)
	if err != nil {
		return uuid.Nil, nil, nil, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseID, err := license.LicenseIDFromSubject(claims.Subject)
	if err != nil {
		return uuid.Nil, nil, nil, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	lic, err := svc.licenses.GetByID(ctx, licenseID)
	if err != nil || lic == nil {
		return uuid.Nil, nil, nil, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !lic.Active {
		return uuid.Nil, nil, nil, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Instance(instance))
	}

	if !lic.ExpiresAt.IsZero() && time.Now().UTC().After(lic.ExpiresAt.UTC()) {
		return uuid.Nil, nil, nil, problem.Of(403).
			Append(problem.Title("License expired")).
			Append(problem.Instance(instance))
	}

	if _, err := svc.activeActivationID(ctx, licenseID, claims); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, nil, nil, problem.Of(403).
				Append(problem.Title("Activation deactivated")).
				Append(problem.Detail("This device has been deactivated for the license")).
				Append(problem.Instance(instance))
		}
		return uuid.Nil, nil, nil, problem.Of(500).
			Append(problem.Title("Activation check failed")).
			Append(problem.Instance(instance))
	}

	return licenseID, lic, claims, nil
}

func (svc *Service) Check(ctx context.Context, data CheckRequest) (CheckResponse, error) {
	instance := "/updates/check"

	licenseID, lic, claims, err := svc.parseActiveLicenseToken(ctx, data.Token, instance)
	if err != nil {
		return CheckResponse{}, err
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

	// Carry the caller's license token onto download URLs so the client can
	// fetch feature-gated artifacts from the channel-gated /download endpoint.
	decision.DownloadURL = appendDownloadToken(decision.DownloadURL, data.Token)
	for i := range decision.Artifacts {
		decision.Artifacts[i].URL = appendDownloadToken(decision.Artifacts[i].URL, data.Token)
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
		config := MustParseProviderConfig(r.Config)
		result[i] = ProductUpdateConfigDTO{
			ID:          r.ID.String(),
			ProductID:   r.ProductID.String(),
			Platform:    r.Platform,
			Channel:     r.ChannelName,
			ChannelID:   r.ChannelID.String(),
			ProviderKey: r.ProviderKey,
			Enabled:     r.Enabled,
			Config:      config,
			FeedURL:     svc.FeedURL(r.ProviderKey, r.ProductID, r.Platform, r.ChannelName, config),
		}
	}
	return result, nil
}

func (svc *Service) SaveProductUpdateConfig(ctx context.Context, orgID, productID uuid.UUID, req SaveProductUpdateConfigRequest) (*ProductUpdateConfigDTO, error) {
	if req.ProviderKey != string(ProviderClaveNative) {
		return nil, fmt.Errorf("unsupported provider key: %s", req.ProviderKey)
	}

	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))

	delivery := deliveryFromConfig(req.Config)
	if !ValidDeliveryProtocol(delivery) {
		return nil, fmt.Errorf("unsupported delivery protocol: %s", delivery)
	}
	if DeliveryProtocol(delivery) == DeliverySparkle && req.Platform != "macos" {
		return nil, fmt.Errorf("sparkle delivery is only supported for macos")
	}

	ch, err := svc.repo.UpsertUpdateChannel(ctx, db.UpsertUpdateChannelParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Name:           req.Channel,
		IsDefault:      false,
	})
	if err != nil {
		return nil, err
	}

	normalizedConfig := map[string]any{"delivery": delivery}
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

	config := MustParseProviderConfig(cfg.Config)
	return &ProductUpdateConfigDTO{
		ID:          cfg.ID.String(),
		ProductID:   cfg.ProductID.String(),
		Platform:    cfg.Platform,
		Channel:     ch.Name,
		ChannelID:   ch.ID.String(),
		ProviderKey: cfg.ProviderKey,
		Enabled:     cfg.Enabled,
		Config:      config,
		FeedURL:     svc.FeedURL(cfg.ProviderKey, cfg.ProductID, cfg.Platform, ch.Name, config),
	}, nil
}

func (svc *Service) DeleteProductUpdateConfig(ctx context.Context, orgID, configID uuid.UUID) error {
	_, err := svc.repo.DeleteProductUpdateConfig(ctx, configID, orgID)
	return err
}

// FeedURL returns the public feed URL based on the configured delivery protocol.
func (svc *Service) FeedURL(providerKey string, productID uuid.UUID, platform, channel string, config map[string]any) string {
	switch DeliveryProtocol(deliveryFromConfig(config)) {
	case DeliverySparkle:
		if platform == "macos" {
			return svc.sparkleFeedURL(productID, channel)
		}
		fallthrough
	default:
		return svc.nativeFeedURL(productID, platform, channel)
	}
}

func (svc *Service) nativeFeedURL(productID uuid.UUID, platform, channel string) string {
	path := fmt.Sprintf("/api/v1/updates/products/%s/%s/%s/feed.json", productID, platform, channel)
	if svc.publicAppURL == "" {
		return path
	}
	return strings.TrimRight(svc.publicAppURL, "/") + path
}

func (svc *Service) sparkleFeedURL(productID uuid.UUID, channel string) string {
	path := fmt.Sprintf("/api/v1/updates/products/%s/macos/%s/appcast.xml", productID, channel)
	if svc.publicAppURL == "" {
		return path
	}
	return strings.TrimRight(svc.publicAppURL, "/") + path
}

func deliveryFromConfig(config map[string]any) string {
	if config == nil {
		return string(DeliveryClaveNative)
	}
	delivery, _ := config["delivery"].(string)
	if delivery == "" {
		return string(DeliveryClaveNative)
	}
	return delivery
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
	return svc.authorizeChannel(ctx, ch, productID, token)
}

// AuthorizeArtifactAccess gates a direct artifact download by the feature
// requirements of the release's channel, mirroring feed access. Channels
// without required features stay open; feature-gated channels need a valid
// license token (passed via ?token= or Authorization: Bearer).
func (svc *Service) AuthorizeArtifactAccess(ctx context.Context, artifactID uuid.UUID, token string) error {
	artifact, err := svc.repo.GetArtifact(ctx, artifactID)
	if err != nil {
		return err
	}
	release, err := svc.repo.GetUpdateRelease(ctx, artifact.ReleaseID)
	if err != nil {
		return err
	}
	channels, err := svc.repo.GetChannelsForProduct(ctx, release.ProductID)
	if err != nil {
		// Can't resolve the channel: don't block (matches unknown-channel feed behavior).
		return nil
	}
	for _, ch := range channels {
		if ch.ID == release.ChannelID {
			return svc.authorizeChannel(ctx, ch, release.ProductID, token)
		}
	}
	return nil
}

// authorizeChannel enforces a channel's required-features against a license
// token. An empty required-features set is always allowed.
func (svc *Service) authorizeChannel(ctx context.Context, ch db.UpdateChannel, productID uuid.UUID, token string) error {
	if len(ch.RequiredFeatures) == 0 {
		return nil
	}
	if token == "" {
		return fmt.Errorf("license token required for this channel")
	}
	_, _, claims, err := svc.parseActiveLicenseToken(ctx, token, "/updates/feed")
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

func (svc *Service) GenerateNativeFeed(ctx context.Context, productID uuid.UUID, platform, channel, token string) ([]byte, error) {
	ch, err := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channel,
	})
	if err != nil {
		return nil, err
	}

	releases, err := svc.repo.ListPublishedReleasesForFeed(ctx, productID, platform, ch.ID)
	if err != nil {
		return nil, err
	}

	inputs := make([]NativeFeedReleaseInput, len(releases))
	releaseIDs := make([]uuid.UUID, len(releases))
	for i, rel := range releases {
		releaseIDs[i] = rel.ID
	}

	allArtifacts, _ := svc.repo.ListArtifactsForReleases(ctx, releaseIDs)
	artifactMap := make(map[uuid.UUID][]db.UpdateArtifact)
	for _, a := range allArtifacts {
		artifactMap[a.ReleaseID] = append(artifactMap[a.ReleaseID], a)
	}

	changelogRows, _ := svc.repo.GetChangelogsByReleaseIDs(ctx, releaseIDs)
	changelogBodyByRelease := make(map[uuid.UUID]string)
	for _, cl := range changelogRows {
		changelogBodyByRelease[cl.ReleaseID] = cl.ChangelogBody
	}

	for i, rel := range releases {
		artifacts := artifactMap[rel.ID]
		for j := range artifacts {
			artifacts[j].Url = appendDownloadToken(artifacts[j].Url, token)
		}
		policy, _ := svc.repo.GetReleasePolicy(ctx, rel.ID)
		changelogBody := changelogBodyByRelease[rel.ID]
		changelogURL := ""
		if rel.ChangelogID.Valid {
			changelogURL = svc.ChangelogURL(rel.ID)
		}
		inputs[i] = NativeFeedReleaseInput{Release: rel, Artifacts: artifacts, Policy: policy, Changelog: changelogBody, ChangelogURL: changelogURL}
	}

	product, err := svc.licenses.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return GenerateNativeFeed(product, platform, channel, inputs)
}

func (svc *Service) GenerateSparkleAppcast(ctx context.Context, productID uuid.UUID, channel, arch, token string) ([]byte, error) {
	ch, err := svc.repo.GetChannelByProductAndName(ctx, db.GetChannelByProductAndNameParams{
		ProductID: productID,
		Name:      channel,
	})
	if err != nil {
		return nil, err
	}

	if err := svc.authorizeChannel(ctx, ch, productID, token); err != nil {
		return nil, err
	}

	releases, err := svc.repo.ListPublishedReleasesForFeed(ctx, productID, "macos", ch.ID)
	if err != nil {
		return nil, err
	}

	inputs := make([]SparkleFeedInput, len(releases))
	releaseIDs := make([]uuid.UUID, len(releases))
	for i, rel := range releases {
		releaseIDs[i] = rel.ID
	}

	allArtifacts, _ := svc.repo.ListArtifactsForReleases(ctx, releaseIDs)
	artifactMap := make(map[uuid.UUID][]db.UpdateArtifact)
	for _, a := range allArtifacts {
		artifactMap[a.ReleaseID] = append(artifactMap[a.ReleaseID], a)
	}

	changelogRows, _ := svc.repo.GetChangelogsByReleaseIDs(ctx, releaseIDs)
	changelogBodyByRelease := make(map[uuid.UUID]string)
	for _, cl := range changelogRows {
		changelogBodyByRelease[cl.ReleaseID] = cl.ChangelogBody
	}

	for i, rel := range releases {
		artifacts := artifactMap[rel.ID]
		for j := range artifacts {
			artifacts[j].Url = appendDownloadToken(artifacts[j].Url, token)
		}
		policy, _ := svc.repo.GetReleasePolicy(ctx, rel.ID)
		changelogBody := changelogBodyByRelease[rel.ID]
		changelogURL := ""
		if rel.ChangelogID.Valid {
			changelogURL = svc.ChangelogURL(rel.ID)
		}
		inputs[i] = SparkleFeedInput{Release: rel, Artifacts: artifacts, Policy: policy, ChangelogBody: changelogBody, ChangelogURL: changelogURL}
	}

	product, err := svc.licenses.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return GenerateSparkleAppcast(product, channel, inputs, arch)
}

func (svc *Service) ListReleases(ctx context.Context, orgID uuid.UUID, productID *uuid.UUID, limit, offset int32) ([]ReleaseDTO, error) {
	rows, err := svc.repo.ListReleasesForOrganization(ctx, orgID, productID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]ReleaseDTO, len(rows))
	releaseIDs := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		releaseIDs[i] = r.ID
	}

	allArtifacts, _ := svc.repo.ListArtifactsForReleases(ctx, releaseIDs)
	artifactMap := make(map[uuid.UUID][]db.UpdateArtifact)
	for _, a := range allArtifacts {
		artifactMap[a.ReleaseID] = append(artifactMap[a.ReleaseID], a)
	}

	for i, r := range rows {
		artifacts := artifactMap[r.ID]
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
				OS:        a.Os,
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
			if len(a.Metadata) > 0 && string(a.Metadata) != "{}" {
				var md map[string]any
				if err := json.Unmarshal(a.Metadata, &md); err == nil {
					artifactDTOs[j].Metadata = md
				}
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

func (svc *Service) UploadArtifact(ctx context.Context, releaseID uuid.UUID, reader io.Reader, artifactType, osName, arch string, originalFilename string, metadata []byte) (*ArtifactDTOFull, error) {
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

	// Read the entire artifact into memory so we can both hash and sign it.
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}

	hash := sha256.Sum256(data)
	checksum := fmt.Sprintf("%x", hash)
	sizeBytes := int64(len(data))

	// Sign the artifact for Sparkle if keys are configured.
	var signature *string
	if sig := svc.SparkleSign(data); sig != "" {
		signature = &sig
	}

	downloadURL := svc.artifactDownloadURL(artifactID)
	mimeType := mimeTypeForArtifact(artifactType)
	backendName := string(kind)
	md := metadata
	if len(md) == 0 {
		md = []byte(`{}`)
	}
	artifact, err := svc.repo.InsertUpdateArtifact(ctx, db.InsertUpdateArtifactParams{
		ID:                   artifactID,
		ReleaseID:            releaseID,
		ArtifactType:         artifactType,
		Os:                   osName,
		Arch:                 arch,
		Url:                  downloadURL,
		SizeBytes:            &sizeBytes,
		ChecksumSha256:       &checksum,
		Signature:            signature,
		Metadata:             md,
		Filename:             &storedFilename,
		MimeType:             &mimeType,
		MinimumSystemVersion: nil,
		StorageBackend:       &backendName,
		StorageKey:           &storageKey,
	})
	if err != nil {
		return nil, fmt.Errorf("insert artifact: %w", err)
	}

	// Upload to storage after DB insert so we only store what was committed.
	if _, putErr := backend.Put(ctx, storageKey, bytes.NewReader(data)); putErr != nil {
		_ = backend.Delete(ctx, storageKey)
		return nil, fmt.Errorf("store artifact: %w", putErr)
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

// presignTTL bounds how long a presigned artifact download URL stays valid.
const presignTTL = 15 * time.Minute

// ArtifactDownload describes how to serve an artifact: either a redirect to a
// presigned URL (RedirectURL set) or a stream the caller must close (Body set).
type ArtifactDownload struct {
	RedirectURL string
	Body        io.ReadCloser
	Size        int64
	MimeType    string
}

// ResolveArtifactDownload prefers a presigned redirect (offloading transfer to
// the storage backend, e.g. S3) and falls back to streaming through the app
// when the backend cannot presign (e.g. local disk).
func (svc *Service) ResolveArtifactDownload(ctx context.Context, artifactID uuid.UUID) (*ArtifactDownload, error) {
	artifact, err := svc.repo.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	release, err := svc.repo.GetUpdateRelease(ctx, artifact.ReleaseID)
	if err != nil {
		return nil, err
	}

	backend, _, err := svc.storageForProduct(ctx, release.ProductID)
	if err != nil {
		return nil, err
	}

	var size int64
	if artifact.SizeBytes != nil {
		size = *artifact.SizeBytes
	}
	mimeType := "application/octet-stream"
	if artifact.MimeType != nil && *artifact.MimeType != "" {
		mimeType = *artifact.MimeType
	}

	key := svc.artifactStorageKey(artifact)

	if url, ok, perr := storage.PresignGet(ctx, backend, key, presignTTL); perr != nil {
		return nil, perr
	} else if ok {
		return &ArtifactDownload{RedirectURL: url, Size: size, MimeType: mimeType}, nil
	}

	rc, err := backend.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	return &ArtifactDownload{Body: rc, Size: size, MimeType: mimeType}, nil
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

// appendDownloadToken adds a license token to an artifact download URL so the
// updater carries it to the (channel-gated) /download endpoint. No-op when the
// token is empty (open channels need none).
func appendDownloadToken(rawURL, token string) string {
	if token == "" {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "token=" + url.QueryEscape(token)
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mimeTypeForArtifact(artifactType string) string {
	switch artifactType {
	case "dmg":
		return "application/x-apple-diskimage"
	case "zip":
		return "application/zip"
	case "pkg":
		return "application/x-apple-installer"
	case "exe":
		return "application/x-msdownload"
	case "msi":
		return "application/x-msi"
	case "deb":
		return "application/vnd.debian.binary-package"
	case "appimage":
		return "application/x-executable"
	default:
		return "application/octet-stream"
	}
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
