-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS changelogs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_changelogs_product_id ON changelogs (product_id);

ALTER TABLE update_releases
    ADD COLUMN IF NOT EXISTS changelog_id UUID REFERENCES changelogs(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_releases DROP COLUMN IF EXISTS changelog_id;
DROP INDEX IF EXISTS idx_changelogs_product_id;
DROP TABLE IF EXISTS changelogs;

-- +goose StatementEnd
