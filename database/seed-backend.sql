BEGIN;

-- Products
INSERT INTO products (id, name, version)
VALUES
  (1, 'Clave Suite', '1.0.0'),
  (2, 'Clave Suite Pro', '1.0.0')
ON CONFLICT DO NOTHING;

COMMIT;
