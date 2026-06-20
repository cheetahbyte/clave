package e2e

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These end-to-end tests run against a live server + Postgres. They are skipped
// unless E2E_BASE_URL (the running server) and DATABASE_URL (for seeding) are set,
// so `go test ./...` stays green for unit-only runs.

var (
	baseURL string
	dbURL   string
)

func TestMain(m *testing.M) {
	baseURL = os.Getenv("E2E_BASE_URL")
	dbURL = os.Getenv("DATABASE_URL")
	os.Exit(m.Run())
}

func requireE2E(t *testing.T) {
	t.Helper()
	if baseURL == "" || dbURL == "" {
		t.Skip("E2E_BASE_URL and DATABASE_URL must be set to run e2e tests")
	}
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func uniqueEmail() string {
	return fmt.Sprintf("e2e-%d@example.com", time.Now().UnixNano())
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// --- seed helpers ---

func orgIDBySlug(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`select id from organizations where slug = $1`, slug).Scan(&id)
	if err != nil {
		t.Fatalf("org by slug %q: %v", slug, err)
	}
	return id
}

func createOrg(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`insert into organizations (name, slug) values ($1, $1) returning id`, slug).Scan(&id)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	return id
}

func createProduct(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`insert into products (name, organization_id) values ($1, $2) returning id`,
		name, orgID).Scan(&id)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	return id
}

func createLicense(t *testing.T, pool *pgxpool.Pool, orgID, productID uuid.UUID, email string, maxActivations int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`insert into licenses (organization_id, product_id, lookup_digest, key_phc, customer_email, max_activations, expires_at)
		 values ($1, $2, $3, $4, $5, $6, now() + interval '365 days') returning id`,
		orgID, productID, randomBytes(32), "phc$test", email, maxActivations).Scan(&id)
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	return id
}

func createDevice(t *testing.T, pool *pgxpool.Pool, licenseID uuid.UUID, hostname string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`insert into devices (license_id, hwid_hash, hostname) values ($1, $2, $3) returning id`,
		licenseID, randomBytes(32), hostname).Scan(&id)
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	_, err = pool.Exec(context.Background(),
		`insert into activations (device_id, license_id) values ($1, $2)`, id, licenseID)
	if err != nil {
		t.Fatalf("create activation: %v", err)
	}
	return id
}

func licenseActive(t *testing.T, pool *pgxpool.Pool, licenseID uuid.UUID) bool {
	t.Helper()
	var active bool
	err := pool.QueryRow(context.Background(),
		`select is_active from licenses where id = $1`, licenseID).Scan(&active)
	if err != nil {
		t.Fatalf("read license active: %v", err)
	}
	return active
}

func deviceExists(t *testing.T, pool *pgxpool.Pool, deviceID uuid.UUID) bool {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`select count(*) from devices where id = $1`, deviceID).Scan(&n)
	if err != nil {
		t.Fatalf("count device: %v", err)
	}
	return n > 0
}

func deviceHasActiveActivation(t *testing.T, pool *pgxpool.Pool, deviceID uuid.UUID) bool {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`select count(*) from activations where device_id = $1 and deactivated_at is null`, deviceID).Scan(&n)
	if err != nil {
		t.Fatalf("count active activation: %v", err)
	}
	return n > 0
}

func licenseFeatures(t *testing.T, pool *pgxpool.Pool, licenseID uuid.UUID) []string {
	t.Helper()
	keys := make([]string, 0)
	rows, err := pool.Query(context.Background(), `
		select pf.key
		from license_features lf
		join product_features pf on pf.id = lf.feature_id
		where lf.license_id = $1
		order by pf.key
	`, licenseID)
	if err != nil {
		t.Fatalf("query license features: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatalf("scan license feature: %v", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return keys
}

func seedProductFeatures(t *testing.T, pool *pgxpool.Pool, orgID, productID uuid.UUID, keys []string) {
	t.Helper()
	for _, k := range keys {
		var id uuid.UUID
		err := pool.QueryRow(context.Background(), `
			insert into product_features (organization_id, product_id, key)
			values ($1, $2, $3)
			on conflict (organization_id, product_id, key) do update set key = excluded.key
			returning id
		`, orgID, productID, k).Scan(&id)
		if err != nil {
			t.Fatalf("seed product feature %q: %v", k, err)
		}
	}
}

func requireSameStringSet(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d (%v)", len(want), len(got), got)
	}
	remaining := map[string]int{}
	for _, v := range want {
		remaining[v]++
	}
	for _, v := range got {
		remaining[v]--
	}
	for v, n := range remaining {
		if n > 0 {
			t.Errorf("missing item %q in %v", v, got)
		}
		if n < 0 {
			t.Errorf("unexpected item %q in %v", v, got)
		}
	}
}

func assignLicenseFeatures(t *testing.T, pool *pgxpool.Pool, licenseID uuid.UUID, keys []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into license_features (license_id, feature_id, source)
		select $1, pf.id, 'manual'
		from product_features pf
		where pf.key = any($2::text[])
		on conflict (license_id, feature_id) do nothing
	`, licenseID, keys)
	if err != nil {
		t.Fatalf("assign license features: %v", err)
	}
	_, err = pool.Exec(context.Background(), `
		update licenses set features = $2::text[] where id = $1
	`, licenseID, keys)
	if err != nil {
		t.Fatalf("backfill licenses.features: %v", err)
	}
}

// --- http helpers ---

func httpClient(t *testing.T) *http.Client {
	t.Helper()
	jar := newCookieJar(t)
	return &http.Client{Jar: jar, Timeout: 10 * time.Second}
}
