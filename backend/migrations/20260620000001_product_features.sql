-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS product_features (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    name            TEXT,
    description     TEXT,
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT product_features_org_product_key_uq UNIQUE (organization_id, product_id, key)
);

CREATE INDEX IF NOT EXISTS idx_product_features_product_id
    ON product_features (product_id);

CREATE INDEX IF NOT EXISTS idx_product_features_organization_id
    ON product_features (organization_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS product_features;

-- +goose StatementEnd
