package selfservice

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeProductDownloadResolver struct {
	productID uuid.UUID
	platform  string
	features  []string
	download  *update.ArtifactDownload
	err       error
}

type fakeDownloadLicenseRepository struct {
	license db.License
	err     error
}

func (f *fakeDownloadLicenseRepository) GetDownloadLicense(context.Context, uuid.UUID, string, uuid.UUID) (db.License, error) {
	return f.license, f.err
}

func (f *fakeProductDownloadResolver) ResolveLatestProductDownload(_ context.Context, productID uuid.UUID, platform string, features []string) (*update.ArtifactDownload, error) {
	f.productID, f.platform, f.features = productID, platform, features
	return f.download, f.err
}

func TestMapLicenseItemIncludesDownloadURL(t *testing.T) {
	licenseID := uuid.New()
	expiresAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	logoURL := "https://example.com/logo.png"
	got := mapLicenseItem(db.ListByCustomerEmailAndOrganizationRow{
		IsActive:       true,
		ID:             licenseID,
		MaxActivations: 3,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Name:           "Example",
		LogoUrl:        &logoURL,
	})
	if got.ID != licenseID.String() || got.DownloadURL != "/api/v1/self-service/licenses/"+licenseID.String()+"/download" {
		t.Fatalf("mapped item = %+v", got)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) || got.LogoURL == nil || *got.LogoURL != logoURL {
		t.Fatalf("nullable fields = %+v", got)
	}
}

func TestLatestDownloadUsesAuthorizedProduct(t *testing.T) {
	licenseID, productID, orgID := uuid.New(), uuid.New(), uuid.New()
	resolver := &fakeProductDownloadResolver{download: &update.ArtifactDownload{Name: "app.dmg"}}
	svc := &Service{
		downloadLicenses: &fakeDownloadLicenseRepository{license: db.License{
			ID:        licenseID,
			ProductID: pgtype.UUID{Bytes: productID, Valid: true},
			Features:  []string{"pro"},
		}},
		downloads: resolver,
	}

	download, err := svc.LatestDownload(context.Background(), "customer@example.com", orgID, licenseID, "macos")
	if err != nil {
		t.Fatal(err)
	}
	if download.Name != "app.dmg" || resolver.productID != productID || resolver.platform != "macos" || !slices.Equal(resolver.features, []string{"pro"}) {
		t.Fatalf("download=%+v product=%s platform=%q features=%v", download, resolver.productID, resolver.platform, resolver.features)
	}
}

func TestLatestDownloadReturnsOwnershipError(t *testing.T) {
	wantErr := errors.New("not owned")
	svc := &Service{
		downloadLicenses: &fakeDownloadLicenseRepository{err: wantErr},
		downloads:        &fakeProductDownloadResolver{},
	}

	_, err := svc.LatestDownload(context.Background(), "customer@example.com", uuid.New(), uuid.New(), "macos")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func downloadTestRouter(t *testing.T, svc *Service) (http.Handler, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"aud":   "selfservice",
		"email": "customer@example.com",
		"org":   uuid.New().String(),
	})
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(svc, nil, "", false, true)
	router := chi.NewRouter()
	router.With(middleware.RequireSelfServiceAuth(publicKey)).Get("/licenses/{licenseId}/download", h.DownloadLatest)
	router.With(middleware.RequireSelfServiceAuth(publicKey)).Head("/licenses/{licenseId}/download", h.DownloadLatest)
	return router, signed
}

func performDownloadRequest(t *testing.T, router http.Handler, token, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestDownloadLatestRejectsMissingPlatform(t *testing.T) {
	licenseID, productID := uuid.New(), uuid.New()
	svc := &Service{
		downloadLicenses: &fakeDownloadLicenseRepository{license: db.License{
			ID: licenseID, ProductID: pgtype.UUID{Bytes: productID, Valid: true},
		}},
		downloads: &fakeProductDownloadResolver{err: update.ErrUnsupportedDownloadPlatform},
	}
	router, token := downloadTestRouter(t, svc)

	res := performDownloadRequest(t, router, token, http.MethodGet, "/licenses/"+licenseID.String()+"/download")
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestDownloadLatestHidesUnavailableDownloads(t *testing.T) {
	tests := []struct {
		name        string
		licenseErr  error
		downloadErr error
	}{
		{name: "unknown license", licenseErr: pgx.ErrNoRows},
		{name: "feature gated", downloadErr: update.ErrDownloadUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			licenseID, productID := uuid.New(), uuid.New()
			svc := &Service{
				downloadLicenses: &fakeDownloadLicenseRepository{
					license: db.License{ID: licenseID, ProductID: pgtype.UUID{Bytes: productID, Valid: true}},
					err:     tt.licenseErr,
				},
				downloads: &fakeProductDownloadResolver{err: tt.downloadErr},
			}
			router, token := downloadTestRouter(t, svc)

			res := performDownloadRequest(t, router, token, http.MethodGet, "/licenses/"+licenseID.String()+"/download?platform=macos")
			if res.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
			}
		})
	}
}

func TestDownloadLatestServesResolvedArtifact(t *testing.T) {
	licenseID, productID := uuid.New(), uuid.New()
	body := strings.NewReader("artifact")
	svc := &Service{
		downloadLicenses: &fakeDownloadLicenseRepository{license: db.License{
			ID: licenseID, ProductID: pgtype.UUID{Bytes: productID, Valid: true},
		}},
		downloads: &fakeProductDownloadResolver{download: &update.ArtifactDownload{
			Body: io.NopCloser(body), Seeker: body, Size: 8, Name: "app.dmg", MimeType: "application/x-apple-diskimage",
		}},
	}
	router, token := downloadTestRouter(t, svc)

	res := performDownloadRequest(t, router, token, http.MethodGet, "/licenses/"+licenseID.String()+"/download?platform=macos")
	if res.Code != http.StatusOK || res.Body.String() != "artifact" {
		t.Fatalf("status = %d body = %q", res.Code, res.Body.String())
	}
}

func TestDownloadLatestSupportsHead(t *testing.T) {
	licenseID, productID := uuid.New(), uuid.New()
	body := strings.NewReader("artifact")
	svc := &Service{
		downloadLicenses: &fakeDownloadLicenseRepository{license: db.License{
			ID: licenseID, ProductID: pgtype.UUID{Bytes: productID, Valid: true},
		}},
		downloads: &fakeProductDownloadResolver{download: &update.ArtifactDownload{
			Body: io.NopCloser(body), Seeker: body, Size: 8, Name: "app.dmg", MimeType: "application/x-apple-diskimage",
		}},
	}
	router, token := downloadTestRouter(t, svc)

	res := performDownloadRequest(t, router, token, http.MethodHead, "/licenses/"+licenseID.String()+"/download?platform=macos")
	if res.Code != http.StatusOK || res.Body.Len() != 0 || res.Header().Get("Content-Length") != "8" {
		t.Fatalf("status = %d body = %q length = %q", res.Code, res.Body.String(), res.Header().Get("Content-Length"))
	}
}
