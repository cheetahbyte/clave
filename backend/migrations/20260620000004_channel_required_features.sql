-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS update_channel_required_features (
    channel_id  UUID NOT NULL REFERENCES update_channels(id) ON DELETE CASCADE,
    feature_id  UUID NOT NULL REFERENCES product_features(id) ON DELETE CASCADE,

    PRIMARY KEY (channel_id, feature_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_req_features_channel_id
    ON update_channel_required_features (channel_id);

CREATE INDEX IF NOT EXISTS idx_channel_req_features_feature_id
    ON update_channel_required_features (feature_id);

-- Backfill from update_channels.required_features TEXT[].
INSERT INTO update_channel_required_features (channel_id, feature_id)
SELECT c.id, pf.id
FROM update_channels c
CROSS JOIN unnest(c.required_features) AS f
JOIN product_features pf
  ON pf.organization_id = c.organization_id
 AND pf.product_id = c.product_id
 AND pf.key = f
WHERE NOT EXISTS (
    SELECT 1 FROM update_channel_required_features crf
    WHERE crf.channel_id = c.id AND crf.feature_id = pf.id
)
ON CONFLICT (channel_id, feature_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS update_channel_required_features;

-- +goose StatementEnd
