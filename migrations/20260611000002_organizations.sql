-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    admin_user_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'admin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, admin_user_id)
);

-- Add nullable columns first
ALTER TABLE products ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE admin_audit_log ADD COLUMN IF NOT EXISTS organization_id UUID;

-- Insert default organization
INSERT INTO organizations (id, name, slug, created_at, updated_at)
VALUES (gen_random_uuid(), 'Default Organization', 'default', now(), now())
ON CONFLICT DO NOTHING;

-- Backfill existing products and licenses
UPDATE products
SET organization_id = o.id
FROM organizations o
WHERE o.slug = 'default'
  AND products.organization_id IS NULL;

UPDATE licenses
SET organization_id = o.id
FROM organizations o
WHERE o.slug = 'default'
  AND licenses.organization_id IS NULL;

-- Backfill existing audit logs
UPDATE admin_audit_log
SET organization_id = o.id
FROM organizations o
WHERE o.slug = 'default'
  AND admin_audit_log.organization_id IS NULL;

-- Add default organization membership for all admins
INSERT INTO organization_members (organization_id, admin_user_id, role)
SELECT o.id, au.id, au.role
FROM organizations o
CROSS JOIN admin_users au
WHERE o.slug = 'default'
ON CONFLICT DO NOTHING;

-- Add NOT NULL constraints
ALTER TABLE products ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE licenses ALTER COLUMN organization_id SET NOT NULL;

-- Add foreign keys
ALTER TABLE products
    ADD CONSTRAINT products_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE licenses
    ADD CONSTRAINT licenses_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE admin_audit_log
    ADD CONSTRAINT admin_audit_log_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_products_organization_id ON products (organization_id);
CREATE INDEX IF NOT EXISTS idx_licenses_organization_id ON licenses (organization_id);
CREATE INDEX IF NOT EXISTS idx_licenses_org_product ON licenses (organization_id, product_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_admin_user_id ON organization_members (admin_user_id);

-- Make product names unique per org
ALTER TABLE products
    ADD CONSTRAINT products_org_name_uq UNIQUE (organization_id, name);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_organization_members_admin_user_id;
DROP INDEX IF EXISTS idx_licenses_org_product;
DROP INDEX IF EXISTS idx_licenses_organization_id;
DROP INDEX IF EXISTS idx_products_organization_id;

ALTER TABLE admin_audit_log DROP CONSTRAINT IF EXISTS admin_audit_log_organization_id_fkey;
ALTER TABLE licenses DROP CONSTRAINT IF EXISTS licenses_organization_id_fkey;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_organization_id_fkey;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_org_name_uq;

ALTER TABLE products DROP COLUMN IF EXISTS organization_id;
ALTER TABLE licenses DROP COLUMN IF EXISTS organization_id;
ALTER TABLE admin_audit_log DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;

-- +goose StatementEnd
