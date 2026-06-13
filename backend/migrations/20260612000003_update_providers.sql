-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS update_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id, name)
);

CREATE INDEX IF NOT EXISTS idx_update_channels_product_id ON update_channels (product_id);
CREATE INDEX IF NOT EXISTS idx_update_channels_org_product ON update_channels (organization_id, product_id);

CREATE TABLE IF NOT EXISTS product_update_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    channel_id UUID NOT NULL REFERENCES update_channels(id) ON DELETE CASCADE,
    provider_key TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id, platform, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_product_update_configs_product_id ON product_update_configs (product_id);
CREATE INDEX IF NOT EXISTS idx_product_update_configs_lookup ON product_update_configs (product_id, platform, channel_id);

CREATE TABLE IF NOT EXISTS update_releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES update_channels(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    version TEXT NOT NULL,
    build_number TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    release_notes TEXT,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id, platform, channel_id, version)
);

CREATE INDEX IF NOT EXISTS idx_update_releases_product_id ON update_releases (product_id);
CREATE INDEX IF NOT EXISTS idx_update_releases_lookup ON update_releases (product_id, platform, channel_id, status);

CREATE TABLE IF NOT EXISTS update_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    artifact_type TEXT NOT NULL,
    os TEXT NOT NULL,
    arch TEXT NOT NULL,
    url TEXT NOT NULL,
    size_bytes BIGINT,
    checksum_sha256 TEXT,
    signature TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_update_artifacts_release_id ON update_artifacts (release_id);

CREATE TABLE IF NOT EXISTS update_release_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES update_releases(id) ON DELETE CASCADE,
    mandatory BOOLEAN NOT NULL DEFAULT false,
    min_supported_version TEXT,
    rollout_percentage INTEGER NOT NULL DEFAULT 100,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS update_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    license_id UUID REFERENCES licenses(id) ON DELETE SET NULL,
    platform TEXT NOT NULL,
    channel TEXT NOT NULL,
    provider_key TEXT NOT NULL,
    current_version TEXT,
    current_build TEXT,
    arch TEXT,
    os_version TEXT,
    decision TEXT NOT NULL,
    selected_release_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_update_checks_product_id ON update_checks (product_id);
CREATE INDEX IF NOT EXISTS idx_update_checks_created_at ON update_checks (created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_update_checks_created_at;
DROP INDEX IF EXISTS idx_update_checks_product_id;
DROP TABLE IF EXISTS update_checks;

DROP TABLE IF EXISTS update_release_policies;
DROP INDEX IF EXISTS idx_update_artifacts_release_id;
DROP TABLE IF EXISTS update_artifacts;
DROP INDEX IF EXISTS idx_update_releases_lookup;
DROP INDEX IF EXISTS idx_update_releases_product_id;
DROP TABLE IF EXISTS update_releases;

DROP INDEX IF EXISTS idx_product_update_configs_lookup;
DROP INDEX IF EXISTS idx_product_update_configs_product_id;
DROP TABLE IF EXISTS product_update_configs;

DROP INDEX IF EXISTS idx_update_channels_org_product;
DROP INDEX IF EXISTS idx_update_channels_product_id;
DROP TABLE IF EXISTS update_channels;

-- +goose StatementEnd
