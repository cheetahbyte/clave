-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_update_releases_latest_published
    ON update_releases (product_id, platform, channel_id, published_at DESC, created_at DESC)
    WHERE status = 'published';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_update_releases_latest_published;

-- +goose StatementEnd
