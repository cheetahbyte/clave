-- +goose Up
-- +goose StatementBegin

ALTER TABLE licenses ADD COLUMN IF NOT EXISTS is_trial BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_licenses_trial_lookup
    ON licenses (organization_id, product_id, customer_email)
    WHERE is_trial = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_licenses_trial_lookup;
ALTER TABLE licenses DROP COLUMN IF EXISTS is_trial;

-- +goose StatementEnd
