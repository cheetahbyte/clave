-- +goose Up
-- +goose StatementBegin

ALTER TABLE activations
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_version TEXT,
    ADD COLUMN IF NOT EXISTS current_build TEXT,
    ADD COLUMN IF NOT EXISTS platform TEXT,
    ADD COLUMN IF NOT EXISTS arch TEXT,
    ADD COLUMN IF NOT EXISTS os_version TEXT;

CREATE INDEX IF NOT EXISTS idx_activations_current_state
    ON activations (last_seen_at DESC)
    WHERE deactivated_at IS NULL AND last_seen_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS daily_version_adoption (
    date DATE NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    device_count BIGINT NOT NULL CHECK (device_count >= 0),
    PRIMARY KEY (date, organization_id, product_id, version)
);

CREATE INDEX IF NOT EXISTS idx_daily_version_adoption_org_date
    ON daily_version_adoption (organization_id, date DESC);

WITH latest AS (
    SELECT DISTINCT ON (activation_id)
        activation_id, version, build, platform, arch, os_version, created_at
    FROM client_checkins
    ORDER BY activation_id, created_at DESC, id DESC
)
UPDATE activations a
SET last_seen_at = latest.created_at,
    current_version = latest.version,
    current_build = latest.build,
    platform = latest.platform,
    arch = latest.arch,
    os_version = latest.os_version
FROM latest
WHERE a.id = latest.activation_id;

INSERT INTO daily_version_adoption (date, organization_id, product_id, version, device_count)
SELECT observed_date, organization_id, product_id, version, count(*)
FROM (
    SELECT DISTINCT ON ((created_at AT TIME ZONE 'UTC')::date, activation_id)
        (created_at AT TIME ZONE 'UTC')::date AS observed_date,
        organization_id,
        product_id,
        activation_id,
        version
    FROM client_checkins
    WHERE (created_at AT TIME ZONE 'UTC')::date < (now() AT TIME ZONE 'UTC')::date
    ORDER BY (created_at AT TIME ZONE 'UTC')::date, activation_id, created_at DESC, id DESC
) latest_daily
GROUP BY observed_date, organization_id, product_id, version
ON CONFLICT (date, organization_id, product_id, version)
DO UPDATE SET device_count = EXCLUDED.device_count;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_daily_version_adoption_org_date;
DROP TABLE IF EXISTS daily_version_adoption;
DROP INDEX IF EXISTS idx_activations_current_state;

ALTER TABLE activations
    DROP COLUMN IF EXISTS os_version,
    DROP COLUMN IF EXISTS arch,
    DROP COLUMN IF EXISTS platform,
    DROP COLUMN IF EXISTS current_build,
    DROP COLUMN IF EXISTS current_version,
    DROP COLUMN IF EXISTS last_seen_at;

-- +goose StatementEnd
