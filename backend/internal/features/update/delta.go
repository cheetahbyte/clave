package update

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/cheetahbyte/clave/pkg/delta"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	deltaReasonLimit = 500
	deltaStaleLease  = 30 * time.Minute
)

var ErrDeltaNotWorthwhile = errors.New("delta patch is not smaller than 70 percent of target")

type DeltaPublisher interface {
	PublishDeltaGenerate(context.Context, string) error
}

type DeltaJobDTO struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	Schema          string  `json:"schema"`
	Algorithm       string  `json:"algorithm"`
	SourceVersion   string  `json:"sourceVersion"`
	TargetVersion   string  `json:"targetVersion"`
	SourceSHA256    string  `json:"sourceSha256"`
	TargetSHA256    string  `json:"targetSha256"`
	SourceSize      int64   `json:"sourceSize"`
	TargetSize      int64   `json:"targetSize"`
	OS              string  `json:"os"`
	Arch            string  `json:"arch"`
	ArtifactType    string  `json:"artifactType"`
	PatchSHA256     *string `json:"patchSha256,omitempty"`
	PatchSize       *int64  `json:"patchSize,omitempty"`
	ErrorMessage    *string `json:"errorMessage,omitempty"`
	DeltaArtifactID string  `json:"deltaArtifactId,omitempty"`
}

func (svc *Service) SetDeltaPublisher(publisher DeltaPublisher) { svc.deltaPublisher = publisher }

func (svc *Service) GenerateDeltaJobs(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateDeltaJob, error) {
	release, err := svc.repo.GetUpdateRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	previous, err := svc.repo.FindPreviousPublishedRelease(ctx, release)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if CompareVersions(previous.Version, release.Version) >= 0 {
		return nil, nil
	}
	sourceArtifacts, err := svc.repo.ListArtifactsForRelease(ctx, previous.ID)
	if err != nil {
		return nil, err
	}
	targetArtifacts, err := svc.repo.ListArtifactsForRelease(ctx, release.ID)
	if err != nil {
		return nil, err
	}
	sources := make(map[string]db.UpdateArtifact)
	for _, artifact := range sourceArtifacts {
		if deltaEligibleArtifact(artifact) {
			sources[artifactCompatibilityKey(artifact)] = artifact
		}
	}
	jobs := make([]db.UpdateDeltaJob, 0)
	for _, target := range targetArtifacts {
		if !deltaEligibleArtifact(target) {
			continue
		}
		source, ok := sources[artifactCompatibilityKey(target)]
		if !ok || source.ID == target.ID {
			continue
		}
		job, err := svc.repo.UpsertDeltaJob(ctx, db.UpsertDeltaJobParams{
			OrganizationID: release.OrganizationID, ReleaseID: release.ID, SourceReleaseID: previous.ID,
			SourceArtifactID: source.ID, TargetArtifactID: target.ID,
			SourceSha256: *source.ChecksumSha256, TargetSha256: *target.ChecksumSha256,
			SourceSize: *source.SizeBytes, TargetSize: *target.SizeBytes,
		})
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, job)
		observability.CountDeltaJob(ctx, job.Status)
	}
	return jobs, nil
}

func deltaEligibleArtifact(artifact db.UpdateArtifact) bool {
	return artifact.ArtifactType != "delta" && artifact.ChecksumSha256 != nil && *artifact.ChecksumSha256 != "" && artifact.SizeBytes != nil && *artifact.SizeBytes > 0
}

func artifactCompatibilityKey(artifact db.UpdateArtifact) string {
	return strings.ToLower(artifact.ArtifactType + "\x00" + artifact.Os + "\x00" + artifact.Arch)
}

func (svc *Service) enqueueDeltaJobs(ctx context.Context, releaseID uuid.UUID) {
	jobs, err := svc.GenerateDeltaJobs(ctx, releaseID)
	if err != nil {
		slog.Warn("failed to create delta jobs", "releaseId", releaseID, "err", err)
		return
	}
	for _, job := range jobs {
		if job.Status != "queued" || svc.deltaPublisher == nil {
			continue
		}
		if err := svc.deltaPublisher.PublishDeltaGenerate(ctx, job.ID.String()); err != nil {
			slog.Warn("failed to publish delta job", "jobId", job.ID, "err", err)
		}
	}
}

func (svc *Service) ClaimDeltaJob(ctx context.Context, jobID uuid.UUID) (*DeltaJobDTO, error) {
	job, err := svc.repo.ClaimDeltaJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	observability.CountDeltaJob(ctx, "running")
	return svc.deltaJobDTO(ctx, job)
}

func (svc *Service) deltaJobDTO(ctx context.Context, job db.UpdateDeltaJob) (*DeltaJobDTO, error) {
	sourceRelease, err := svc.repo.GetUpdateRelease(ctx, job.SourceReleaseID)
	if err != nil {
		return nil, err
	}
	targetRelease, err := svc.repo.GetUpdateRelease(ctx, job.ReleaseID)
	if err != nil {
		return nil, err
	}
	targetArtifact, err := svc.repo.GetArtifact(ctx, job.TargetArtifactID)
	if err != nil {
		return nil, err
	}
	dto := &DeltaJobDTO{
		ID: job.ID.String(), Status: job.Status, Schema: job.SchemaVersion, Algorithm: job.Algorithm,
		SourceVersion: sourceRelease.Version, TargetVersion: targetRelease.Version,
		SourceSHA256: job.SourceSha256, TargetSHA256: job.TargetSha256,
		SourceSize: job.SourceSize, TargetSize: job.TargetSize,
		OS: targetArtifact.Os, Arch: targetArtifact.Arch, ArtifactType: targetArtifact.ArtifactType,
		PatchSHA256: job.PatchSha256, PatchSize: job.PatchSize, ErrorMessage: job.ErrorMessage,
	}
	if job.DeltaArtifactID.Valid {
		dto.DeltaArtifactID = uuid.UUID(job.DeltaArtifactID.Bytes).String()
	}
	return dto, nil
}

func (svc *Service) ListDeltaJobs(ctx context.Context, releaseID uuid.UUID) ([]DeltaJobDTO, error) {
	jobs, err := svc.repo.ListDeltaJobsForRelease(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	out := make([]DeltaJobDTO, 0, len(jobs))
	for _, job := range jobs {
		dto, err := svc.deltaJobDTO(ctx, job)
		if err != nil {
			return nil, err
		}
		out = append(out, *dto)
	}
	return out, nil
}

func (svc *Service) RetryDeltaJobs(ctx context.Context, releaseID uuid.UUID) ([]db.UpdateDeltaJob, error) {
	jobs, err := svc.repo.RetryDeltaJobsForRelease(ctx, releaseID, int32(deltaStaleLease.Seconds()))
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if svc.deltaPublisher != nil {
			if err := svc.deltaPublisher.PublishDeltaGenerate(ctx, job.ID.String()); err != nil {
				slog.Warn("failed to republish delta job", "jobId", job.ID, "err", err)
			}
		}
	}
	return jobs, nil
}

func (svc *Service) ResolveDeltaJobArtifact(ctx context.Context, jobID uuid.UUID, source bool) (*ArtifactDownload, error) {
	job, err := svc.repo.GetDeltaJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "running" {
		return nil, fmt.Errorf("delta job is %s", job.Status)
	}
	artifactID := job.TargetArtifactID
	if source {
		artifactID = job.SourceArtifactID
	}
	return svc.ResolveArtifactDownload(ctx, artifactID)
}

func (svc *Service) SkipDeltaJob(ctx context.Context, jobID uuid.UUID, patchSHA string, patchSize int64, reason string) (db.UpdateDeltaJob, error) {
	reason = boundedDeltaReason(reason)
	job, err := svc.repo.SkipDeltaJob(ctx, db.SkipDeltaJobParams{ID: jobID, PatchSha256: optionalString(patchSHA), PatchSize: optionalInt64(patchSize), ErrorMessage: &reason})
	if err == nil {
		observability.CountDeltaJob(ctx, "skipped")
		observability.RecordDeltaPatchRatio(ctx, patchSize, job.TargetSize)
	}
	return job, err
}

func (svc *Service) FailDeltaJob(ctx context.Context, jobID uuid.UUID, reason string) (db.UpdateDeltaJob, error) {
	reason = boundedDeltaReason(reason)
	job, err := svc.repo.FailDeltaJob(ctx, db.FailDeltaJobParams{ID: jobID, ErrorMessage: &reason})
	if err == nil {
		observability.CountDeltaJob(ctx, "failed")
	}
	return job, err
}

func (svc *Service) RequeueDeltaJob(ctx context.Context, jobID uuid.UUID) (db.UpdateDeltaJob, error) {
	return svc.repo.RequeueDeltaJob(ctx, jobID)
}

func (svc *Service) CompleteDeltaJob(ctx context.Context, jobID uuid.UUID, reader io.Reader, declaredSHA string, declaredSize int64) (db.UpdateDeltaJob, error) {
	job, err := svc.repo.GetDeltaJob(ctx, jobID)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	if job.Status != "running" {
		return db.UpdateDeltaJob{}, fmt.Errorf("delta job is %s", job.Status)
	}
	tmp, err := os.CreateTemp("", "clave-delta-upload-*")
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	tmpName := tmp.Name()
	defer func() { tmp.Close(); os.Remove(tmpName) }()
	hasher := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hasher), reader)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	patchSHA := fmt.Sprintf("%x", hasher.Sum(nil))
	if declaredSize != size || declaredSHA != patchSHA {
		return db.UpdateDeltaJob{}, errors.New("declared patch size or SHA-256 does not match upload")
	}
	if size*100 >= job.TargetSize*70 {
		_, skipErr := svc.SkipDeltaJob(ctx, jobID, patchSHA, size, "patch_not_worthwhile")
		if skipErr != nil {
			return db.UpdateDeltaJob{}, skipErr
		}
		return db.UpdateDeltaJob{}, ErrDeltaNotWorthwhile
	}
	sourceRelease, err := svc.repo.GetUpdateRelease(ctx, job.SourceReleaseID)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	targetRelease, err := svc.repo.GetUpdateRelease(ctx, job.ReleaseID)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	targetArtifact, err := svc.repo.GetArtifact(ctx, job.TargetArtifactID)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	contract := delta.Contract{Schema: delta.Schema, Algorithm: delta.Algorithm, FromVersion: sourceRelease.Version, ToVersion: targetRelease.Version, BaseSHA256: job.SourceSha256, TargetSHA256: job.TargetSha256, PatchSHA256: patchSHA, TargetSize: job.TargetSize}
	metadata, err := delta.CanonicalContract(contract)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	signature, err := svc.signer.SignDomainPayload(delta.Schema, metadata)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	backend, kind, err := svc.storageForProduct(ctx, targetRelease.ProductID)
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	artifactID := uuid.New()
	filename := artifactID.String() + ".delta"
	storageKey := artifactID.String() + "/" + filename
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return db.UpdateDeltaJob{}, err
	}
	backendName := string(kind)
	mimeType := "application/octet-stream"
	artifactType := "delta"
	completed, err := storeDeltaAndPersist(ctx, backend, storageKey, tmp, func() (db.UpdateDeltaJob, error) {
		return svc.repo.CompleteDeltaJobWithArtifact(ctx, db.InsertUpdateArtifactParams{
			ID: artifactID, ReleaseID: targetRelease.ID, ArtifactType: artifactType,
			Os: targetArtifact.Os, Arch: targetArtifact.Arch, Url: svc.artifactDownloadURL(artifactID),
			SizeBytes: &size, ChecksumSha256: &patchSHA, Signature: &signature, Metadata: metadata,
			Filename: &filename, MimeType: &mimeType, StorageBackend: &backendName, StorageKey: &storageKey,
		}, db.CompleteDeltaJobParams{ID: jobID, DeltaArtifactID: pgtype.UUID{Bytes: artifactID, Valid: true}, PatchSha256: &patchSHA, PatchSize: &size})
	})
	if err != nil {
		return db.UpdateDeltaJob{}, err
	}
	svc.feedCache.invalidateProduct(targetRelease.ProductID)
	observability.CountDeltaJob(ctx, "completed")
	observability.RecordDeltaPatchRatio(ctx, size, job.TargetSize)
	return completed, nil
}

func storeDeltaAndPersist(
	ctx context.Context,
	backend storage.Backend,
	key string,
	reader io.Reader,
	persist func() (db.UpdateDeltaJob, error),
) (db.UpdateDeltaJob, error) {
	if _, err := backend.Put(ctx, key, reader); err != nil {
		_ = backend.Delete(ctx, key)
		return db.UpdateDeltaJob{}, err
	}
	job, err := persist()
	if err != nil {
		_ = backend.Delete(ctx, key)
		return db.UpdateDeltaJob{}, err
	}
	return job, nil
}

func boundedDeltaReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unspecified"
	}
	if len(reason) > deltaReasonLimit {
		reason = reason[:deltaReasonLimit]
	}
	return reason
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value int64) *int64 {
	if value < 0 {
		return nil
	}
	return &value
}
