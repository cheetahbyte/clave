package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{name: "header", auth: "Bearer license.jwt.token", want: "license.jwt.token"},
		{name: "case insensitive scheme", auth: "bearer license.jwt.token", want: "license.jwt.token"},
		{name: "whitespace", auth: "Bearer   spaced.token   ", want: "spaced.token"},
		{name: "missing"},
		{name: "basic", auth: "Basic dXNlcjpwYXNz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json", nil)
			r.Header.Set("Authorization", tt.auth)
			if got := middleware.BearerToken(r); got != tt.want {
				t.Fatalf("BearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBearerTokenRejectsBodyAndQueryCredentials(t *testing.T) {
	r := httptest.NewRequest("POST", "/client/updates/check?token=query", strings.NewReader(`{"token":"body"}`))
	if got := middleware.BearerToken(r); got != "" {
		t.Fatalf("BearerToken() = %q, want empty", got)
	}
}

func TestClientEndpointsRejectBodyToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		body string
		h    func(http.ResponseWriter, *http.Request)
		want int
	}{
		{"check", `{"token":"body-token","version":"1.0.0"}`, NewHandler(nil, nil).Check, http.StatusBadRequest},
		{"channels", `{"token":"body-token"}`, NewHandler(&Service{signer: signing.New(publicKey, privateKey, "")}, nil).ClientChannels, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/updates/"+tt.name, strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			tt.h(w, r)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tt.want, w.Body.String())
			}
			if tt.name == "channels" && strings.Contains(w.Body.String(), "EOF") {
				t.Fatalf("body must not contain JSON EOF error: %s", w.Body.String())
			}
		})
	}
}

func TestClientChannelsAcceptsBodylessBearerRequest(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	NewHandler(&Service{signer: signing.New(publicKey, privateKey, "")}, nil).RegisterClientRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/updates/channels", nil)
	req.Header.Set("Authorization", "Bearer valid-header-token")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusUnauthorized, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "EOF") {
		t.Fatalf("body must not contain JSON EOF error: %s", res.Body.String())
	}
}

func TestSetPrivateCacheHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	setPrivateCacheHeaders(rr)

	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("expected Cache-Control 'private, no-store', got %q", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected Referrer-Policy 'no-referrer', got %q", got)
	}
}

func TestGenerateNativeFeedDoesNotAppendToken(t *testing.T) {
	product := db.Product{ID: uuid.New(), Name: "TestApp"}
	release := db.UpdateRelease{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		ProductID:      product.ID,
		ChannelID:      uuid.New(),
		Platform:       "macos",
		Version:        "1.2.3",
		BuildNumber:    strPtr("100"),
		Status:         "published",
	}
	downloadURL := "https://releases.example.com/testapp-1.2.3.dmg"
	artifact := db.UpdateArtifact{
		ID:             uuid.New(),
		ReleaseID:      release.ID,
		ArtifactType:   "dmg",
		Os:             "macos",
		Arch:           "arm64",
		Url:            downloadURL,
		ChecksumSha256: strPtr("deadbeef"),
	}
	policy := db.UpdateReleasePolicy{
		ReleaseID:         release.ID,
		Mandatory:         false,
		RolloutPercentage: 100,
	}

	inputs := []NativeFeedReleaseInput{{
		Release:   release,
		Artifacts: []db.UpdateArtifact{artifact},
		Policy:    policy,
	}}

	body, err := GenerateNativeFeed(product, "macos", "stable", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(body)
	if strings.Contains(out, "token=") {
		t.Fatalf("native feed must not contain 'token=', got: %s", out)
	}
	if !strings.Contains(out, downloadURL) {
		t.Fatalf("expected native feed to contain the original download URL %q, got: %s", downloadURL, out)
	}
}

func TestGenerateNativeFeedPreservesAllArtifactURLs(t *testing.T) {
	product := db.Product{ID: uuid.New(), Name: "TestApp"}
	release := db.UpdateRelease{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		ProductID:      product.ID,
		ChannelID:      uuid.New(),
		Platform:       "macos",
		Version:        "1.2.3",
		BuildNumber:    strPtr("100"),
		Status:         "published",
	}
	artifacts := []db.UpdateArtifact{
		{
			ID:           uuid.New(),
			ReleaseID:    release.ID,
			ArtifactType: "dmg",
			Os:           "macos",
			Arch:         "arm64",
			Url:          "https://releases.example.com/testapp-1.2.3-arm64.dmg",
		},
		{
			ID:           uuid.New(),
			ReleaseID:    release.ID,
			ArtifactType: "dmg",
			Os:           "macos",
			Arch:         "x64",
			Url:          "https://releases.example.com/testapp-1.2.3-x64.dmg?signature=abc",
		},
	}
	policy := db.UpdateReleasePolicy{
		ReleaseID:         release.ID,
		Mandatory:         false,
		RolloutPercentage: 100,
	}

	inputs := []NativeFeedReleaseInput{{
		Release:   release,
		Artifacts: artifacts,
		Policy:    policy,
	}}

	body, err := GenerateNativeFeed(product, "macos", "stable", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(body)
	for _, a := range artifacts {
		if !strings.Contains(out, a.Url) {
			t.Fatalf("expected feed to contain artifact URL %q, got: %s", a.Url, out)
		}
	}
	if strings.Contains(out, "&token=") || strings.Contains(out, "?token=") {
		t.Fatalf("feed must not introduce token query params, got: %s", out)
	}
}

// make sure the pgtype import is used in case of future field additions.
func strPtr(s string) *string { return &s }
