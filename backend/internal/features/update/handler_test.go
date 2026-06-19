package update

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

func TestFeedTokenReadsAuthorizationHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json", nil)
	r.Header.Set("Authorization", "Bearer license.jwt.token")

	got := feedToken(r)
	if got != "license.jwt.token" {
		t.Fatalf("expected 'license.jwt.token', got %q", got)
	}
}

func TestFeedTokenIgnoresQueryToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json?token=query.token", nil)

	got := feedToken(r)
	if got != "" {
		t.Fatalf("query-string token must not be accepted, got %q", got)
	}
}

func TestFeedTokenHeaderWinsOverQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json?token=query.token", nil)
	r.Header.Set("Authorization", "Bearer header.token")

	got := feedToken(r)
	if got != "header.token" {
		t.Fatalf("expected header token to win, got %q", got)
	}
}

func TestFeedTokenEmptyWhenMissing(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json", nil)

	got := feedToken(r)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFeedTokenTrimsHeaderWhitespace(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json", nil)
	r.Header.Set("Authorization", "Bearer   spaced.token   ")

	got := feedToken(r)
	if got != "spaced.token" {
		t.Fatalf("expected trimmed token, got %q", got)
	}
}

func TestFeedTokenRejectsNonBearerScheme(t *testing.T) {
	r := httptest.NewRequest("GET", "/updates/products/abc/macos/stable/feed.json", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	got := feedToken(r)
	if got != "" {
		t.Fatalf("non-Bearer scheme must not yield a token, got %q", got)
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

func TestGenerateSparkleAppcastDoesNotAppendToken(t *testing.T) {
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

	inputs := []SparkleFeedInput{{
		Release:   release,
		Artifacts: []db.UpdateArtifact{artifact},
		Policy:    policy,
	}}

	body, err := GenerateSparkleAppcast(product, "stable", inputs, "arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(body)
	if strings.Contains(out, "token=") {
		t.Fatalf("appcast must not contain 'token=', got: %s", out)
	}
	if !strings.Contains(out, downloadURL) {
		t.Fatalf("expected appcast to contain the original download URL %q, got: %s", downloadURL, out)
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
