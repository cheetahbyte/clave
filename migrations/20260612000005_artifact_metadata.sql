-- +goose Up
-- +goose StatementBegin

ALTER TABLE update_artifacts
  ADD COLUMN filename TEXT,
  ADD COLUMN mime_type TEXT,
  ADD COLUMN minimum_system_version TEXT,
  ADD COLUMN sparkle_ed_signature TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE update_artifacts
  DROP COLUMN IF EXISTS filename,
  DROP COLUMN IF EXISTS mime_type,
  DROP COLUMN IF EXISTS minimum_system_version,
  DROP COLUMN IF EXISTS sparkle_ed_signature;

-- +goose StatementEnd
