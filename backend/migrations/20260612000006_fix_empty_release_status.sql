-- +goose Up
-- +goose StatementBegin

UPDATE update_releases SET status = 'draft' WHERE status = '' OR status IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No down migration needed for a data fix.
SELECT 1;
-- +goose StatementEnd
