-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_org_created
    ON admin_audit_log (organization_id, created_at DESC);

DROP INDEX IF EXISTS idx_licenses_trial_lookup;

CREATE INDEX IF NOT EXISTS idx_licenses_trial_lookup
    ON licenses (organization_id, product_id, lower(customer_email))
    WHERE is_trial = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_admin_audit_log_org_created;

DROP INDEX IF EXISTS idx_licenses_trial_lookup;

CREATE INDEX IF NOT EXISTS idx_licenses_trial_lookup
    ON licenses (organization_id, product_id, customer_email)
    WHERE is_trial = true;

-- +goose StatementEnd
