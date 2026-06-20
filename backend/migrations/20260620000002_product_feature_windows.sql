-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS product_feature_windows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    feature_id      UUID NOT NULL REFERENCES product_features(id) ON DELETE CASCADE,
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    applies_to      TEXT NOT NULL DEFAULT 'standard',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT product_feature_windows_starts_ends_chk CHECK (starts_at < ends_at),
    CONSTRAINT product_feature_windows_applies_to_chk CHECK (applies_to IN ('standard', 'trial', 'all'))
);

-- Partial index for the hot path: license creation looks up active windows.
CREATE INDEX IF NOT EXISTS idx_product_feature_windows_active_lookup
    ON product_feature_windows (product_id, starts_at, ends_at)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_product_feature_windows_product_id
    ON product_feature_windows (product_id);

CREATE INDEX IF NOT EXISTS idx_product_feature_windows_organization_id
    ON product_feature_windows (organization_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS product_feature_windows;

-- +goose StatementEnd
