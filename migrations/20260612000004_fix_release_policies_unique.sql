-- +goose Up
-- +goose StatementBegin

ALTER TABLE update_release_policies
  ADD CONSTRAINT update_release_policies_release_id_key UNIQUE (release_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_release_policies
  DROP CONSTRAINT IF EXISTS update_release_policies_release_id_key;

-- +goose StatementEnd
