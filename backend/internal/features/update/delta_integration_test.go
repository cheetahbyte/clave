package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deltaFixture struct {
	organizationID uuid.UUID
	productID      uuid.UUID
	channelID      uuid.UUID
	sourceRelease  uuid.UUID
	targetRelease  uuid.UUID
	sourceArtifact uuid.UUID
	targetArtifact uuid.UUID
	jobID          uuid.UUID
	sourceContent  []byte
}

func TestDeltaJobGenerationAndStateTransitionsIntegration(t *testing.T) {
	tx, q := openDeltaIntegrationTx(t)
	storageRoot := t.TempDir()

	noPredecessor := seedSingleRelease(t, tx)
	svc := newDeltaIntegrationService(q, tx, storageRoot)
	jobs, err := svc.GenerateDeltaJobs(t.Context(), noPredecessor)
	if err != nil {
		t.Fatalf("generate without predecessor: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs without predecessor = %d, want 0", len(jobs))
	}

	fixture := seedDeltaFixture(t, tx, svc, storageRoot)
	first, err := svc.GenerateDeltaJobs(t.Context(), fixture.targetRelease)
	if err != nil || len(first) != 1 {
		t.Fatalf("first generation jobs=%d err=%v", len(first), err)
	}
	second, err := svc.GenerateDeltaJobs(t.Context(), fixture.targetRelease)
	if err != nil || len(second) != 1 {
		t.Fatalf("duplicate generation jobs=%d err=%v", len(second), err)
	}
	if first[0].ID != second[0].ID {
		t.Fatalf("duplicate generation created another job: %s != %s", first[0].ID, second[0].ID)
	}

	jobID := first[0].ID
	if _, err := svc.ClaimDeltaJob(t.Context(), jobID); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := svc.ClaimDeltaJob(t.Context(), jobID); !isNoRows(err) {
		t.Fatalf("second claim error = %v, want no rows", err)
	}
	if _, err := svc.SkipDeltaJob(t.Context(), jobID, "", -1, "not worthwhile"); err != nil {
		t.Fatalf("skip: %v", err)
	}
	if _, err := svc.FailDeltaJob(t.Context(), jobID, "late failure"); !isNoRows(err) {
		t.Fatalf("terminal transition error = %v, want no rows", err)
	}

	retried, err := svc.RetryDeltaJobs(t.Context(), fixture.targetRelease)
	if err != nil || len(retried) != 1 || retried[0].Status != "queued" {
		t.Fatalf("retry jobs=%v err=%v", retried, err)
	}
	if _, err := svc.ClaimDeltaJob(t.Context(), jobID); err != nil {
		t.Fatalf("claim after retry: %v", err)
	}
	if _, err := svc.RequeueDeltaJob(t.Context(), jobID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if _, err := svc.ClaimDeltaJob(t.Context(), jobID); err != nil {
		t.Fatalf("claim after requeue: %v", err)
	}
	if _, err := svc.FailDeltaJob(t.Context(), jobID, "generation failed"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if _, err := svc.RetryDeltaJobs(t.Context(), fixture.targetRelease); err != nil {
		t.Fatalf("retry failed job: %v", err)
	}
	if _, err := svc.ClaimDeltaJob(t.Context(), jobID); err != nil {
		t.Fatalf("claim before completion: %v", err)
	}
	deltaArtifactID := uuid.New()
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_artifacts (id, release_id, artifact_type, os, arch, url, size_bytes, checksum_sha256)
		VALUES ($1, $2, 'delta', 'macos', 'arm64', 'delta', 1, $3)`, deltaArtifactID, fixture.targetRelease, strings.Repeat("a", 64)); err != nil {
		t.Fatalf("insert delta artifact: %v", err)
	}
	patchSHA, patchSize := strings.Repeat("a", 64), int64(1)
	completed, err := q.CompleteDeltaJob(t.Context(), db.CompleteDeltaJobParams{
		ID: jobID, DeltaArtifactID: pgtype.UUID{Bytes: deltaArtifactID, Valid: true}, PatchSha256: &patchSHA, PatchSize: &patchSize,
	})
	if err != nil || completed.Status != "completed" {
		t.Fatalf("complete status=%s err=%v", completed.Status, err)
	}
	if _, err := svc.RequeueDeltaJob(t.Context(), jobID); !isNoRows(err) {
		t.Fatalf("completed job requeue error = %v, want no rows", err)
	}

	cleanupKey := "cleanup/patch.delta"
	_, err = storeDeltaAndPersist(t.Context(), storage.NewLocal(storageRoot), cleanupKey, bytes.NewReader([]byte("patch")), func() (db.UpdateDeltaJob, error) {
		return q.CompleteDeltaJob(t.Context(), db.CompleteDeltaJobParams{
			ID: uuid.New(), DeltaArtifactID: pgtype.UUID{Bytes: deltaArtifactID, Valid: true}, PatchSha256: &patchSHA, PatchSize: &patchSize,
		})
	})
	if !isNoRows(err) {
		t.Fatalf("persistence rejection error = %v, want no rows", err)
	}
	if _, statErr := os.Stat(filepath.Join(storageRoot, filepath.FromSlash(cleanupKey))); !os.IsNotExist(statErr) {
		t.Fatalf("patch remained after database rejection: %v", statErr)
	}
}

func TestDeltaWorkerAndAdminRoutesIntegration(t *testing.T) {
	tx, q := openDeltaIntegrationTx(t)
	storageRoot := t.TempDir()
	svc := newDeltaIntegrationService(q, tx, storageRoot)
	fixture := seedDeltaFixture(t, tx, svc, storageRoot)
	handler := NewHandler(svc, nil)

	workerRouter := chi.NewRouter()
	workerRouter.Route("/api/v1/worker", func(r chi.Router) {
		r.Use(middleware.RequireWorkerToken("worker-secret"))
		handler.RegisterWorkerRoutes(r)
	})
	workerRequest := func(method, path string, body io.Reader) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, body)
		req.Header.Set("Authorization", "Bearer worker-secret")
		response := httptest.NewRecorder()
		workerRouter.ServeHTTP(response, req)
		return response
	}

	wrongJob := workerRequest(http.MethodGet, "/api/v1/worker/delta-jobs/"+uuid.NewString()+"/artifacts/source", nil)
	if wrongJob.Code != http.StatusConflict {
		t.Fatalf("artifact for unrelated job status = %d", wrongJob.Code)
	}
	claimed := workerRequest(http.MethodPost, "/api/v1/worker/delta-jobs/"+fixture.jobID.String()+"/claim", nil)
	if claimed.Code != http.StatusOK {
		t.Fatalf("claim status = %d body=%s", claimed.Code, claimed.Body.String())
	}
	download := workerRequest(http.MethodGet, "/api/v1/worker/delta-jobs/"+fixture.jobID.String()+"/artifacts/source", nil)
	if download.Code != http.StatusOK || download.Body.String() != string(fixture.sourceContent) {
		t.Fatalf("source download status=%d body=%q", download.Code, download.Body.String())
	}
	secondClaim := workerRequest(http.MethodPost, "/api/v1/worker/delta-jobs/"+fixture.jobID.String()+"/claim", nil)
	if secondClaim.Code != http.StatusConflict {
		t.Fatalf("second claim status = %d", secondClaim.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/worker/delta-jobs/"+fixture.jobID.String()+"/complete", &zeroReader{remaining: (64 << 20) + 1})
	req.Header.Set("Authorization", "Bearer worker-secret")
	req.Header.Set("X-Patch-Size", fmt.Sprint((64<<20)+1))
	req.Header.Set("X-Patch-SHA256", strings.Repeat("0", 64))
	tooLarge := httptest.NewRecorder()
	workerRouter.ServeHTTP(tooLarge, req)
	if tooLarge.Code != http.StatusConflict {
		t.Fatalf("oversized upload status = %d body=%s", tooLarge.Code, tooLarge.Body.String())
	}

	adminServer := newDeltaAdminTestServer(t, handler, fixture.organizationID)
	defer adminServer.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	login := func(orgID uuid.UUID) {
		response, err := client.Get(adminServer.URL + "/login/" + orgID.String())
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		response.Body.Close()
	}
	listURL := adminServer.URL + "/admin/update-releases/" + fixture.targetRelease.String() + "/delta-jobs"
	login(uuid.New())
	response, err := client.Get(listURL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-organization list status = %d", response.StatusCode)
	}
	login(fixture.organizationID)
	response, err = client.Get(listURL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("owned release list status = %d", response.StatusCode)
	}
}

func openDeltaIntegrationTx(t *testing.T) (pgx.Tx, *db.Queries) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}
	pool, err := pgxpool.New(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx, db.New(tx)
}

func newDeltaIntegrationService(q *db.Queries, tx pgx.Tx, storageRoot string) *Service {
	return NewService(nil, nil, NewRepository(q, tx), nil, "http://clave.test", storageRoot)
}

func seedSingleRelease(t *testing.T, tx pgx.Tx) uuid.UUID {
	t.Helper()
	orgID, productID, channelID := seedDeltaParents(t, tx)
	releaseID := uuid.New()
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_releases (id, organization_id, product_id, channel_id, platform, version, status, published_at)
		VALUES ($1, $2, $3, $4, 'macos', '1.0.0', 'published', now())`, releaseID, orgID, productID, channelID); err != nil {
		t.Fatalf("insert single release: %v", err)
	}
	return releaseID
}

func seedDeltaFixture(t *testing.T, tx pgx.Tx, svc *Service, storageRoot string) deltaFixture {
	t.Helper()
	orgID, productID, channelID := seedDeltaParents(t, tx)
	fixture := deltaFixture{
		organizationID: orgID, productID: productID, channelID: channelID,
		sourceRelease: uuid.New(), targetRelease: uuid.New(), sourceArtifact: uuid.New(), targetArtifact: uuid.New(),
		sourceContent: []byte("deterministic source archive"),
	}
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_releases (id, organization_id, product_id, channel_id, platform, version, status, published_at)
		VALUES ($1, $2, $3, $4, 'macos', '1.0.0', 'published', now() - interval '1 hour'),
		       ($5, $2, $3, $4, 'macos', '1.1.0', 'published', now())`,
		fixture.sourceRelease, orgID, productID, channelID, fixture.targetRelease); err != nil {
		t.Fatalf("insert releases: %v", err)
	}
	targetContent := []byte("deterministic target archive")
	sourceSHA := fmt.Sprintf("%x", sha256.Sum256(fixture.sourceContent))
	targetSHA := fmt.Sprintf("%x", sha256.Sum256(targetContent))
	sourceKey := fixture.sourceArtifact.String() + "/source.zip"
	targetKey := fixture.targetArtifact.String() + "/target.zip"
	if _, err := tx.Exec(t.Context(), `
		INSERT INTO update_artifacts (id, release_id, artifact_type, os, arch, url, size_bytes, checksum_sha256, filename, mime_type, storage_backend, storage_key)
		VALUES ($1, $2, 'zip', 'macos', 'arm64', 'source', $3, $4, 'source.zip', 'application/zip', 'local', $5),
		       ($6, $7, 'zip', 'macos', 'arm64', 'target', $8, $9, 'target.zip', 'application/zip', 'local', $10)`,
		fixture.sourceArtifact, fixture.sourceRelease, len(fixture.sourceContent), sourceSHA, sourceKey,
		fixture.targetArtifact, fixture.targetRelease, len(targetContent), targetSHA, targetKey); err != nil {
		t.Fatalf("insert artifacts: %v", err)
	}
	writeDeltaFixtureFile(t, storageRoot, sourceKey, fixture.sourceContent)
	writeDeltaFixtureFile(t, storageRoot, targetKey, targetContent)
	jobs, err := svc.GenerateDeltaJobs(t.Context(), fixture.targetRelease)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("generate fixture job jobs=%d err=%v", len(jobs), err)
	}
	fixture.jobID = jobs[0].ID
	return fixture
}

func seedDeltaParents(t *testing.T, tx pgx.Tx) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	orgID, productID, channelID := uuid.New(), uuid.New(), uuid.New()
	suffix := strings.ReplaceAll(orgID.String(), "-", "")
	if _, err := tx.Exec(t.Context(), `INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)`, orgID, "Delta Test "+suffix, "delta-test-"+suffix); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if _, err := tx.Exec(t.Context(), `INSERT INTO products (id, organization_id, name) VALUES ($1, $2, $3)`, productID, orgID, "Product "+suffix); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if _, err := tx.Exec(t.Context(), `INSERT INTO update_channels (id, organization_id, product_id, name, is_default) VALUES ($1, $2, $3, 'stable', true)`, channelID, orgID, productID); err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	return orgID, productID, channelID
}

func writeDeltaFixtureFile(t *testing.T, root, key string, content []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newDeltaAdminTestServer(t *testing.T, handler *Handler, defaultOrg uuid.UUID) *httptest.Server {
	t.Helper()
	sessions := scs.New()
	router := chi.NewRouter()
	router.Use(sessions.LoadAndSave)
	router.Get("/login/{orgId}", func(w http.ResponseWriter, r *http.Request) {
		orgID := chi.URLParam(r, "orgId")
		if orgID == "" {
			orgID = defaultOrg.String()
		}
		sessions.Put(r.Context(), "admin_user_id", uuid.NewString())
		sessions.Put(r.Context(), "admin_email", "delta-test@example.test")
		sessions.Put(r.Context(), "admin_role", "admin")
		sessions.Put(r.Context(), "admin_organization_id", orgID)
		w.WriteHeader(http.StatusNoContent)
	})
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAdmin(sessions))
		r.Get("/admin/update-releases/{id}/delta-jobs", handler.AdminListDeltaJobs)
	})
	return httptest.NewServer(router)
}

func isNoRows(err error) bool { return err == pgx.ErrNoRows }

type zeroReader struct{ remaining int64 }

func (r *zeroReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = 0
	}
	r.remaining -= int64(len(p))
	return len(p), nil
}
