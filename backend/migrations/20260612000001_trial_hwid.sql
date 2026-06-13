-- +goose Up
-- +goose StatementBegin

ALTER TABLE licenses ADD COLUMN IF NOT EXISTS trial_hwid_hash BYTEA;

CREATE UNIQUE INDEX IF NOT EXISTS licenses_one_trial_per_hwid_product_uq
    ON licenses (organization_id, product_id, trial_hwid_hash)
    WHERE is_trial = true AND trial_hwid_hash IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS licenses_one_trial_per_hwid_product_uq;
ALTER TABLE licenses DROP COLUMN IF EXISTS trial_hwid_hash;

-- +goose StatementEnd
