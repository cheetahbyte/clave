-- +goose Up
-- +goose StatementBegin

CREATE TABLE update_delta_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    source_release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    source_artifact_id UUID NOT NULL REFERENCES update_artifacts(id) ON DELETE CASCADE,
    target_artifact_id UUID NOT NULL REFERENCES update_artifacts(id) ON DELETE CASCADE,
    delta_artifact_id UUID REFERENCES update_artifacts(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'completed', 'skipped', 'failed')),
    schema_version TEXT NOT NULL DEFAULT 'clave.delta/v1',
    algorithm TEXT NOT NULL DEFAULT 'bsdiff',
    source_sha256 TEXT NOT NULL,
    target_sha256 TEXT NOT NULL,
    patch_sha256 TEXT,
    source_size BIGINT NOT NULL CHECK (source_size > 0),
    target_size BIGINT NOT NULL CHECK (target_size > 0),
    patch_size BIGINT CHECK (patch_size IS NULL OR patch_size >= 0),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE (source_artifact_id, target_artifact_id),
    CHECK (source_artifact_id <> target_artifact_id)
);

CREATE INDEX idx_update_delta_jobs_release_id ON update_delta_jobs (release_id, created_at DESC);
CREATE INDEX idx_update_delta_jobs_status ON update_delta_jobs (status, started_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_update_delta_jobs_status;
DROP INDEX IF EXISTS idx_update_delta_jobs_release_id;
DROP TABLE IF EXISTS update_delta_jobs;

-- +goose StatementEnd
