-- +goose Up
-- +goose StatementBegin

ALTER TABLE products ADD COLUMN IF NOT EXISTS logo_url TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE products DROP COLUMN IF EXISTS logo_url;

-- +goose StatementEnd
