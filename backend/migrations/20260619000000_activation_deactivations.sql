-- +goose Up
-- +goose StatementBegin

ALTER TABLE activations
    ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deactivation_reason TEXT;

DROP INDEX IF EXISTS activations_license_device_uq;

CREATE UNIQUE INDEX IF NOT EXISTS activations_license_device_active_uq
    ON activations (license_id, device_id)
    WHERE deactivated_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_activations_active_license_id
    ON activations (license_id)
    WHERE deactivated_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_activations_active_license_id;
DROP INDEX IF EXISTS activations_license_device_active_uq;

-- Keep rollback safe only when no deactivation history would be lost.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM activations WHERE deactivated_at IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot drop activation deactivation columns while deactivated rows exist';
    END IF;
END $$;

ALTER TABLE activations
    DROP COLUMN IF EXISTS deactivation_reason,
    DROP COLUMN IF EXISTS deactivated_at;

CREATE UNIQUE INDEX IF NOT EXISTS activations_license_device_uq
    ON activations (license_id, device_id);

-- +goose StatementEnd
