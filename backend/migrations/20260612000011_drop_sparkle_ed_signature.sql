-- +goose Up
-- +goose StatementBegin

ALTER TABLE update_artifacts DROP COLUMN IF EXISTS sparkle_ed_signature;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_artifacts ADD COLUMN sparkle_ed_signature TEXT;

-- +goose StatementEnd
