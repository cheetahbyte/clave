-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS product_storage_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    backend TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id)
);

CREATE INDEX IF NOT EXISTS idx_product_storage_configs_product_id ON product_storage_configs (product_id);

ALTER TABLE update_artifacts ADD COLUMN IF NOT EXISTS storage_backend TEXT;
ALTER TABLE update_artifacts ADD COLUMN IF NOT EXISTS storage_key TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_artifacts DROP COLUMN IF EXISTS storage_key;
ALTER TABLE update_artifacts DROP COLUMN IF EXISTS storage_backend;

DROP INDEX IF EXISTS idx_product_storage_configs_product_id;
DROP TABLE IF EXISTS product_storage_configs;

-- +goose StatementEnd
