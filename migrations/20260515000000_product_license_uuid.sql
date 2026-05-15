-- +goose Up
-- +goose StatementBegin

-- Step 1: Add new UUID columns alongside existing integer ones
ALTER TABLE products ADD COLUMN new_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE licenses ADD COLUMN new_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE licenses ADD COLUMN new_product_id UUID;
ALTER TABLE devices  ADD COLUMN new_license_id UUID;
ALTER TABLE activations ADD COLUMN new_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE activations ADD COLUMN new_license_id UUID;

-- Step 2: Propagate UUIDs through FK relationships
UPDATE licenses l
SET new_product_id = p.new_id
FROM products p
WHERE l.product_id = p.id;

UPDATE devices d
SET new_license_id = l.new_id
FROM licenses l
WHERE d.license_id = l.id;

UPDATE activations a
SET new_license_id = l.new_id
FROM licenses l
WHERE a.license_id = l.id;

-- Step 3: Drop indexes that reference old integer columns
DROP INDEX IF EXISTS activations_license_device_uq;
DROP INDEX IF EXISTS idx_activations_license_id;
DROP INDEX IF EXISTS idx_devices_license_id;

-- Step 4: Drop FK constraints referencing columns that are changing
ALTER TABLE licenses    DROP CONSTRAINT IF EXISTS licenses_product_id_fkey;
ALTER TABLE devices     DROP CONSTRAINT IF EXISTS devices_license_id_fkey;
ALTER TABLE activations DROP CONSTRAINT IF EXISTS activations_license_id_fkey;

-- Step 5: Drop old PKs
ALTER TABLE products    DROP CONSTRAINT products_pkey;
ALTER TABLE licenses    DROP CONSTRAINT licenses_pkey;
ALTER TABLE activations DROP CONSTRAINT activations_pkey;

-- Step 6: Drop old integer columns (sequences auto-dropped, unique constraints auto-dropped)
ALTER TABLE products    DROP COLUMN id;
ALTER TABLE licenses    DROP COLUMN id;
ALTER TABLE licenses    DROP COLUMN product_id;
ALTER TABLE devices     DROP COLUMN license_id;
ALTER TABLE activations DROP COLUMN id;
ALTER TABLE activations DROP COLUMN license_id;

-- Step 7: Rename new UUID columns into place
ALTER TABLE products    RENAME COLUMN new_id TO id;
ALTER TABLE licenses    RENAME COLUMN new_id TO id;
ALTER TABLE licenses    RENAME COLUMN new_product_id TO product_id;
ALTER TABLE devices     RENAME COLUMN new_license_id TO license_id;
ALTER TABLE activations RENAME COLUMN new_id TO id;
ALTER TABLE activations RENAME COLUMN new_license_id TO license_id;

-- Step 8: Apply NOT NULL / DEFAULT constraints
ALTER TABLE products    ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE licenses    ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE activations ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE devices     ALTER COLUMN license_id SET NOT NULL;
ALTER TABLE activations ALTER COLUMN license_id SET NOT NULL;

-- Step 9: Restore PKs
ALTER TABLE products    ADD PRIMARY KEY (id);
ALTER TABLE licenses    ADD PRIMARY KEY (id);
ALTER TABLE activations ADD PRIMARY KEY (id);

-- Step 10: Restore FK constraints
ALTER TABLE licenses ADD CONSTRAINT licenses_product_id_fkey
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE;

ALTER TABLE devices ADD CONSTRAINT devices_license_id_fkey
    FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE;

ALTER TABLE activations ADD CONSTRAINT activations_license_id_fkey
    FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE;

-- Step 11: Restore unique constraint and indexes
ALTER TABLE devices ADD UNIQUE (license_id, hwid_hash);

CREATE INDEX IF NOT EXISTS idx_devices_license_id
    ON devices (license_id);

CREATE INDEX IF NOT EXISTS idx_activations_license_id
    ON activations (license_id);

CREATE UNIQUE INDEX IF NOT EXISTS activations_license_device_uq
    ON activations (license_id, device_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- UUID → integer reversal is not deterministic; manual rollback required.
-- Restore from a pre-migration backup.
DO $$ BEGIN RAISE EXCEPTION 'Down migration not supported: restore from backup'; END $$;

-- +goose StatementEnd
