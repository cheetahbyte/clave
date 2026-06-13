package update

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewRepository(q *db.Queries, pool *pgxpool.Pool) *Repository {
	return &Repository{q: q, pool: pool}
}

func (r *Repository) GetProductByIDAndOrganization(ctx context.Context, orgID, id uuid.UUID) (db.Product, error) {
	return r.q.GetOneByIdForOrganization(ctx, db.GetOneByIdForOrganizationParams{
		ID:             id,
		OrganizationID: orgID,
	})
}

func (r *Repository) GetProductUpdateConfig(ctx context.Context, productID uuid.UUID, platform string, channel string) (db.ProductUpdateConfig, error) {
	return r.q.GetProductUpdateConfig(ctx, db.GetProductUpdateConfigParams{
		ProductID: productID,
		Platform:  platform,
		Name:      channel,
	})
}

func (r *Repository) GetProductUpdateConfigs(ctx context.Context, productID uuid.UUID) ([]db.GetProductUpdateConfigsRow, error) {
	return r.q.GetProductUpdateConfigs(ctx, productID)
}

func (r *Repository) UpsertProductUpdateConfig(ctx context.Context, params db.UpsertProductUpdateConfigParams) (db.ProductUpdateConfig, error) {
	return r.q.UpsertProductUpdateConfig(ctx, params)
}

func (r *Repository) DeleteProductUpdateConfig(ctx context.Context, configID, orgID uuid.UUID) (db.ProductUpdateConfig, error) {
	return r.q.DeleteProductUpdateConfig(ctx, db.DeleteProductUpdateConfigParams{
		ID:             configID,
		OrganizationID: orgID,
	})
}

func (r *Repository) GetProductStorageConfig(ctx context.Context, productID uuid.UUID) (db.ProductStorageConfig, error) {
	return r.q.GetProductStorageConfig(ctx, productID)
}

func (r *Repository) UpsertProductStorageConfig(ctx context.Context, params db.UpsertProductStorageConfigParams) (db.ProductStorageConfig, error) {
	return r.q.UpsertProductStorageConfig(ctx, params)
}

func (r *Repository) GetDefaultChannelForProduct(ctx context.Context, productID uuid.UUID) (db.UpdateChannel, error) {
	return r.q.GetDefaultChannelForProduct(ctx, productID)
}

func (r *Repository) GetChannelByProductAndName(ctx context.Context, params db.GetChannelByProductAndNameParams) (db.UpdateChannel, error) {
	return r.q.GetChannelByProductAndName(ctx, params)
}

func (r *Repository) GetChannelsForProduct(ctx context.Context, productID uuid.UUID) ([]db.UpdateChannel, error) {
	return r.q.GetChannelsForProduct(ctx, productID)
}

func (r *Repository) GetUpdateChannelByID(ctx context.Context, id uuid.UUID) (db.UpdateChannel, error) {
	return r.q.GetUpdateChannelByID(ctx, id)
}

func (r *Repository) UpsertUpdateChannel(ctx context.Context, params db.UpsertUpdateChannelParams) (db.UpdateChannel, error) {
	return r.q.UpsertUpdateChannel(ctx, params)
}

func (r *Repository) CreateUpdateChannel(ctx context.Context, params db.CreateUpdateChannelParams) (db.UpdateChannel, error) {
	return r.q.CreateUpdateChannel(ctx, params)
}

func (r *Repository) UpdateUpdateChannel(ctx context.Context, params db.UpdateUpdateChannelParams) (db.UpdateChannel, error) {
	return r.q.UpdateUpdateChannel(ctx, params)
}

func (r *Repository) DeleteUpdateChannel(ctx context.Context, channelID, orgID uuid.UUID) (db.UpdateChannel, error) {
	return r.q.DeleteUpdateChannel(ctx, db.DeleteUpdateChannelParams{ID: channelID, OrganizationID: orgID})
}

func (r *Repository) CountReleasesForChannel(ctx context.Context, channelID uuid.UUID) (int64, error) {
	return r.q.CountReleasesForChannel(ctx, channelID)
}

func (r *Repository) CountConfigsForChannel(ctx context.Context, channelID uuid.UUID) (int64, error) {
	return r.q.CountConfigsForChannel(ctx, channelID)
}

func (r *Repository) UpdateReleaseChangelog(ctx context.Context, params db.UpdateReleaseChangelogParams) (db.UpdateRelease, error) {
	return r.q.UpdateReleaseChangelog(ctx, params)
}

func (r *Repository) ListChangelogsForProduct(ctx context.Context, productID uuid.UUID) ([]db.Changelog, error) {
	return r.q.ListChangelogsForProduct(ctx, productID)
}

func (r *Repository) GetChangelog(ctx context.Context, id uuid.UUID) (db.Changelog, error) {
	return r.q.GetChangelog(ctx, id)
}

func (r *Repository) CreateChangelog(ctx context.Context, params db.CreateChangelogParams) (db.Changelog, error) {
	return r.q.CreateChangelog(ctx, params)
}

func (r *Repository) UpdateChangelog(ctx context.Context, params db.UpdateChangelogParams) (db.Changelog, error) {
	return r.q.UpdateChangelog(ctx, params)
}

func (r *Repository) DeleteChangelog(ctx context.Context, id, orgID uuid.UUID) (db.Changelog, error) {
	return r.q.DeleteChangelog(ctx, db.DeleteChangelogParams{ID: id, OrganizationID: orgID})
}

func (r *Repository) CountReleasesForChangelog(ctx context.Context, changelogID uuid.UUID) (int64, error) {
	pg := pgtype.UUID{Bytes: changelogID, Valid: true}
	return r.q.CountReleasesForChangelog(ctx, pg)
}

func (r *Repository) SetReleaseChangelog(ctx context.Context, releaseID uuid.UUID, changelogID *uuid.UUID) (db.UpdateRelease, error) {
	pg := pgtype.UUID{}
	if changelogID != nil {
		pg = pgtype.UUID{Bytes: *changelogID, Valid: true}
	}
	return r.q.SetReleaseChangelog(ctx, db.SetReleaseChangelogParams{ID: releaseID, ChangelogID: pg})
}

func (r *Repository) LatestPublishedUpdateRelease(ctx context.Context, params db.LatestPublishedUpdateReleaseParams) (db.UpdateRelease, error) {
	return r.q.LatestPublishedUpdateRelease(ctx, params)
}

func (r *Repository) ListPublishedReleasesForAppcast(ctx context.Context, productID uuid.UUID, platform string, channelID uuid.UUID) ([]db.UpdateRelease, error) {
	return r.q.ListPublishedReleasesForAppcast(ctx, db.ListPublishedReleasesForAppcastParams{
		ProductID: productID,
		Platform:  platform,
		ChannelID: channelID,
	})
}

func (r *Repository) ListReleasesForOrganization(ctx context.Context, orgID uuid.UUID, productID *uuid.UUID, limit, offset int32) ([]db.ListReleasesForOrganizationRow, error) {
	if productID == nil {
		return r.q.ListReleasesForOrganization(ctx, db.ListReleasesForOrganizationParams{
			OrganizationID: orgID,
			Limit:          limit,
			Offset:         offset,
		})
	}

	rows, err := r.q.ListReleasesForOrganizationByProduct(ctx, db.ListReleasesForOrganizationByProductParams{
		OrganizationID: orgID,
		ProductID:      *productID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]db.ListReleasesForOrganizationRow, len(rows))
	for i, r := range rows {
		out[i] = db.ListReleasesForOrganizationRow(r)
	}
	return out, nil
}

func (r *Repository) InsertUpdateRelease(ctx context.Context, params db.InsertUpdateReleaseParams) (db.UpdateRelease, error) {
	return r.q.InsertUpdateRelease(ctx, params)
}

func (r *Repository) GetUpdateRelease(ctx context.Context, id uuid.UUID) (db.UpdateRelease, error) {
	return r.q.GetUpdateRelease(ctx, id)
}

func (r *Repository) PublishUpdateRelease(ctx context.Context, id uuid.UUID) (db.UpdateRelease, error) {
	return r.q.PublishUpdateRelease(ctx, id)
}

func (r *Repository) YankUpdateRelease(ctx context.Context, id uuid.UUID) (db.UpdateRelease, error) {
	return r.q.YankUpdateRelease(ctx, id)
}

func (r *Repository) DeleteUpdateRelease(ctx context.Context, id uuid.UUID) (db.UpdateRelease, error) {
	return r.q.DeleteUpdateRelease(ctx, id)
}

func (r *Repository) ListReleasesForProductChannel(ctx context.Context, productID uuid.UUID, platform string, channelID uuid.UUID, limit, offset int32) ([]db.UpdateRelease, error) {
	return r.q.ListReleasesForProductChannel(ctx, db.ListReleasesForProductChannelParams{
		ProductID: productID,
		Platform:  platform,
		ChannelID: channelID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *Repository) InsertUpdateArtifact(ctx context.Context, params db.InsertUpdateArtifactParams) (db.UpdateArtifact, error) {
	return r.q.InsertUpdateArtifact(ctx, params)
}

func (r *Repository) GetArtifact(ctx context.Context, id uuid.UUID) (db.UpdateArtifact, error) {
	return r.q.GetUpdateArtifact(ctx, id)
}

func (r *Repository) ListArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error) {
	return r.q.ListArtifactsForRelease(ctx, releaseID)
}

func (r *Repository) GetReleasePolicy(ctx context.Context, releaseID uuid.UUID) (db.UpdateReleasePolicy, error) {
	return r.q.GetReleasePolicy(ctx, releaseID)
}

func (r *Repository) InsertUpdateCheck(ctx context.Context, orgID, productID, licenseID uuid.UUID, platform, channel, providerKey, currentVersion, currentBuild, arch, osVersion, decision string, selectedReleaseID *uuid.UUID) error {
	org := pgtype.UUID{Bytes: orgID, Valid: true}
	prod := pgtype.UUID{Bytes: productID, Valid: true}
	lic := pgtype.UUID{Bytes: licenseID, Valid: true}

	ver := &currentVersion
	build := &currentBuild
	a := &arch
	os := &osVersion

	if currentVersion == "" {
		ver = nil
	}
	if currentBuild == "" {
		build = nil
	}
	if arch == "" {
		a = nil
	}
	if osVersion == "" {
		os = nil
	}

	relID := pgtype.UUID{}
	if selectedReleaseID != nil {
		relID = pgtype.UUID{Bytes: *selectedReleaseID, Valid: true}
	}

	_, err := r.q.InsertUpdateCheck(ctx, db.InsertUpdateCheckParams{
		OrganizationID:    org,
		ProductID:         prod,
		LicenseID:         lic,
		Platform:          platform,
		Channel:           channel,
		ProviderKey:       providerKey,
		CurrentVersion:    ver,
		CurrentBuild:      build,
		Arch:              a,
		OsVersion:         os,
		Decision:          decision,
		SelectedReleaseID: relID,
	})
	return err
}

func (r *Repository) EnsureDefaultChannel(ctx context.Context, orgID, productID uuid.UUID) (db.UpdateChannel, error) {
	ch, err := r.GetDefaultChannelForProduct(ctx, productID)
	if err == nil {
		return ch, nil
	}

	return r.UpsertUpdateChannel(ctx, db.UpsertUpdateChannelParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Name:           "stable",
		IsDefault:      true,
	})
}

func MustParseProviderConfig(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func uuidToPG(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
