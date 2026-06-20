-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS license_features (
    license_id       UUID NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    feature_id       UUID NOT NULL REFERENCES product_features(id) ON DELETE CASCADE,
    granted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    source           TEXT NOT NULL DEFAULT 'manual',
    source_window_id UUID REFERENCES product_feature_windows(id) ON DELETE SET NULL,

    PRIMARY KEY (license_id, feature_id)
);

CREATE INDEX IF NOT EXISTS idx_license_features_license_id
    ON license_features (license_id);

CREATE INDEX IF NOT EXISTS idx_license_features_feature_id
    ON license_features (feature_id);

-- Backfill: create product_features entries for any feature keys that don't
-- exist yet, derived from existing licenses.features and
-- update_channels.required_features TEXT[] columns.
INSERT INTO product_features (organization_id, product_id, key)
SELECT DISTINCT l.organization_id, l.product_id, f
FROM licenses l
CROSS JOIN unnest(l.features) AS f
WHERE NOT EXISTS (
    SELECT 1 FROM product_features pf
    WHERE pf.organization_id = l.organization_id
      AND pf.product_id = l.product_id
      AND pf.key = f
)
ON CONFLICT (organization_id, product_id, key) DO NOTHING;

INSERT INTO product_features (organization_id, product_id, key)
SELECT DISTINCT c.organization_id, c.product_id, f
FROM update_channels c
CROSS JOIN unnest(c.required_features) AS f
WHERE NOT EXISTS (
    SELECT 1 FROM product_features pf
    WHERE pf.organization_id = c.organization_id
      AND pf.product_id = c.product_id
      AND pf.key = f
)
ON CONFLICT (organization_id, product_id, key) DO NOTHING;

-- Backfill license_features from licenses.features TEXT[].
INSERT INTO license_features (license_id, feature_id, source)
SELECT l.id, pf.id, 'backfill'
FROM licenses l
CROSS JOIN unnest(l.features) AS f
JOIN product_features pf
  ON pf.organization_id = l.organization_id
 AND pf.product_id = l.product_id
 AND pf.key = f
WHERE NOT EXISTS (
    SELECT 1 FROM license_features lf
    WHERE lf.license_id = l.id AND lf.feature_id = pf.id
)
ON CONFLICT (license_id, feature_id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS license_features;

-- +goose StatementEnd
