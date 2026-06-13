-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS update_delta_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    source_release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    source_artifact_id UUID REFERENCES update_artifacts(id) ON DELETE SET NULL,
    target_artifact_id UUID REFERENCES update_artifacts(id) ON DELETE SET NULL,
    delta_artifact_id UUID REFERENCES update_artifacts(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_update_delta_jobs_release_id ON update_delta_jobs (release_id);
CREATE INDEX IF NOT EXISTS idx_update_delta_jobs_status ON update_delta_jobs (status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_update_delta_jobs_status;
DROP INDEX IF EXISTS idx_update_delta_jobs_release_id;
DROP TABLE IF EXISTS update_delta_jobs;

-- +goose StatementEnd
