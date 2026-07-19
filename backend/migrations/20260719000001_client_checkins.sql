-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS client_checkins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    license_id UUID NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    activation_id UUID NOT NULL REFERENCES activations(id) ON DELETE CASCADE,
    version TEXT NOT NULL CHECK (length(btrim(version)) > 0),
    build TEXT,
    platform TEXT,
    arch TEXT,
    os_version TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_client_checkins_org_created
    ON client_checkins (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_client_checkins_product_created
    ON client_checkins (product_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_client_checkins_activation_created
    ON client_checkins (activation_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS client_checkins;

-- +goose StatementEnd
