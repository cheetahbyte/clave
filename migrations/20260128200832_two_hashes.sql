-- +goose Up
-- +goose StatementBegin
ALTER TABLE licenses
    ADD COLUMN lookup_digest BYTEA,
    ADD COLUMN key_phc TEXT;

DROP INDEX IF EXISTS idx_licenses_key_hash;
ALTER TABLE licenses
    DROP CONSTRAINT IF EXISTS licenses_key_hash_key;

ALTER TABLE licenses
    DROP COLUMN key_hash;

ALTER TABLE licenses
    ALTER COLUMN lookup_digest SET NOT NULL,
    ALTER COLUMN key_phc SET NOT NULL;

CREATE UNIQUE INDEX licenses_lookup_digest_uq
    ON licenses (lookup_digest);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS licenses_lookup_digest_uq;

ALTER TABLE licenses
    DROP COLUMN lookup_digest,
    DROP COLUMN key_phc;

ALTER TABLE licenses
    ADD COLUMN key_hash TEXT NOT NULL;

CREATE UNIQUE INDEX idx_licenses_key_hash
    ON licenses (key_hash);

-- +goose StatementEnd
