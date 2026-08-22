package update

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type Repository struct {
	q  *db.Queries
	db transactionBeginner
}

func NewRepository(q *db.Queries, database transactionBeginner) *Repository {
	return &Repository{q: q, db: database}
}

func (r *Repository) GetProductByIDAndOrganization(ctx context.Context, orgID, id uuid.UUID) (db.Product, error) {
	return r.q.GetOneByIdForOrganization(ctx, db.GetOneByIdForOrganizationParams{
		ID:             id,
		OrganizationID: orgID,
	})
}

func (r *Repository) GetProductUpdateConfig(ctx context.Context, productID uuid.UUID, platform string, channel string) (db.GetProductUpdateConfigRow, error) {
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

func (r *Repository) GetActiveActivationByID(ctx context.Context, params db.GetActiveActivationByIDParams) (db.Activation, error) {
	return r.q.GetActiveActivationByID(ctx, params)
}

func (r *Repository) GetActiveActivationByLicenseAndHwidHash(ctx context.Context, params db.GetActiveActivationByLicenseAndHwidHashParams) (db.Activation, error) {
	return r.q.GetActiveActivationByLicenseAndHwidHash(ctx, params)
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

func (r *Repository) ListPublishedReleasesForFeed(ctx context.Context, productID uuid.UUID, platform string, channelID uuid.UUID) ([]db.UpdateRelease, error) {
	return r.q.ListPublishedReleasesForFeed(ctx, db.ListPublishedReleasesForFeedParams{
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

func (r *Repository) FindPreviousPublishedRelease(ctx context.Context, release db.UpdateRelease) (db.UpdateRelease, error) {
	return r.q.FindPreviousPublishedRelease(ctx, db.FindPreviousPublishedReleaseParams{
		ProductID: release.ProductID, Platform: release.Platform, ChannelID: release.ChannelID, ID: release.ID,
	})
}

var (
	ErrReleaseAlreadyHasArtifacts = errors.New("target release already has artifacts")
	ErrArtifactsOnlyAddedToDraft  = errors.New("artifacts can only be added to a draft release")
)

func (r *Repository) InsertUpdateArtifact(ctx context.Context, params db.InsertUpdateArtifactParams) (db.UpdateArtifact, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return db.UpdateArtifact{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, "SELECT status FROM update_releases WHERE id = $1 FOR UPDATE", params.ReleaseID).Scan(&status); err != nil {
		return db.UpdateArtifact{}, err
	}
	if status != "draft" {
		return db.UpdateArtifact{}, ErrArtifactsOnlyAddedToDraft
	}
	artifact, err := r.q.WithTx(tx).InsertUpdateArtifact(ctx, params)
	if err != nil {
		return db.UpdateArtifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.UpdateArtifact{}, err
	}
	return artifact, nil
}

// InsertUpdateArtifactsIfReleaseEmpty atomically verifies that releaseID has no
// artifacts and inserts the supplied artifacts. Locking the parent release row
// serializes competing reuse requests before the empty-release check.
func (r *Repository) InsertUpdateArtifactsIfReleaseEmpty(ctx context.Context, releaseID uuid.UUID, artifacts []db.InsertUpdateArtifactParams) ([]db.UpdateArtifact, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Acquiring a FOR UPDATE lock on the parent release serializes this guard
	// with other reuse attempts for the same release.
	if err := tx.QueryRow(ctx, "SELECT id FROM update_releases WHERE id = $1 FOR UPDATE", releaseID).Scan(new(uuid.UUID)); err != nil {
		return nil, err
	}

	qtx := r.q.WithTx(tx)
	release, err := qtx.GetUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if release.Status != "draft" {
		return nil, ErrArtifactsOnlyAddedToDraft
	}

	existing, err := qtx.ListArtifactsForRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if len(existing) != 0 {
		return nil, ErrReleaseAlreadyHasArtifacts
	}

	inserted := make([]db.UpdateArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		insertedArtifact, err := qtx.InsertUpdateArtifact(ctx, artifact)
		if err != nil {
			return nil, err
		}
		inserted = append(inserted, insertedArtifact)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return inserted, nil
}

func (r *Repository) GetArtifact(ctx context.Context, id uuid.UUID) (db.UpdateArtifact, error) {
	return r.q.GetUpdateArtifact(ctx, id)
}

func (r *Repository) UpsertDeltaJob(ctx context.Context, params db.UpsertDeltaJobParams) (db.UpdateDeltaJob, error) {
	return r.q.UpsertDeltaJob(ctx, params)
}

func (r *Repository) GetDeltaJob(ctx context.Context, id uuid.UUID) (db.UpdateDeltaJob, error) {
	return r.q.GetDeltaJob(ctx, id)
}

func (r *Repository) ClaimDeltaJob(ctx context.Context, id uuid.UUID) (db.UpdateDeltaJob, error) {
	return r.q.ClaimDeltaJob(ctx, id)
}

func (r *Repository) SkipDeltaJob(ctx context.Context, params db.SkipDeltaJobParams) (db.UpdateDeltaJob, error) {
	return r.q.SkipDeltaJob(ctx, params)
}

func (r *Repository) FailDeltaJob(ctx context.Context, params db.FailDeltaJobParams) (db.UpdateDeltaJob, error) {
	return r.q.FailDeltaJob(ctx, params)
}

func (r *Repository) RequeueDeltaJob(ctx context.Context, id uuid.UUID) (db.UpdateDeltaJob, error) {
	return r.q.RequeueDeltaJob(ctx, id)
}

func (r *Repository) RetryDeltaJobsForRelease(ctx context.Context, releaseID uuid.UUID, staleSeconds int32) ([]db.UpdateDeltaJob, error) {
	return r.q.RetryDeltaJobsForRelease(ctx, db.RetryDeltaJobsForReleaseParams{ReleaseID: releaseID, StaleSeconds: staleSeconds})
}

func (r *Repository) ListDeltaJobsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateDeltaJob, error) {
	return r.q.ListDeltaJobsForRelease(ctx, releaseID)
}

func (r *Repository) ListCompletedDeltaArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error) {
	return r.q.ListCompletedDeltaArtifactsForRelease(ctx, releaseID)
}

func (r *Repository) CompleteDeltaJobWithArtifact(ctx context.Context, artifact db.InsertUpdateArtifactParams, complete db.CompleteDeltaJobParams) (db.UpdateDeltaJob, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)
	if _, err := qtx.InsertUpdateArtifact(ctx, artifact); err != nil {
		return db.UpdateDeltaJob{}, err
	}
	job, err := qtx.CompleteDeltaJob(ctx, complete)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.UpdateDeltaJob{}, err
	}
	return job, nil
}

func (r *Repository) ListArtifactsForRelease(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateArtifact, error) {
	return r.q.ListArtifactsForRelease(ctx, releaseID)
}

func (r *Repository) ListArtifactsForReleases(ctx context.Context, releaseIDs []uuid.UUID) ([]db.UpdateArtifact, error) {
	if len(releaseIDs) == 0 {
		return nil, nil
	}
	return r.q.ListArtifactsForReleases(ctx, releaseIDs)
}

func (r *Repository) GetChangelogsByReleaseIDs(ctx context.Context, releaseIDs []uuid.UUID) ([]db.GetChangelogsByReleaseIDsRow, error) {
	if len(releaseIDs) == 0 {
		return nil, nil
	}
	return r.q.GetChangelogsByReleaseIDs(ctx, releaseIDs)
}

func (r *Repository) GetReleasePolicy(ctx context.Context, releaseID uuid.UUID) (db.UpdateReleasePolicy, error) {
	return r.q.GetReleasePolicy(ctx, releaseID)
}

func (r *Repository) GetReleasePoliciesForReleases(ctx context.Context, releaseIDs []uuid.UUID) ([]db.UpdateReleasePolicy, error) {
	if len(releaseIDs) == 0 {
		return nil, nil
	}
	return r.q.GetReleasePoliciesForReleases(ctx, releaseIDs)
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
