package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestFetchLatestRelease(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	svc := &UpdateService{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://api.github.com/repos/cheetahbyte/kepler-releases/releases/latest" {
					t.Fatalf("unexpected url: %s", req.URL.String())
				}
				if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
					t.Fatalf("unexpected authorization header: %q", got)
				}
				if got := req.Header.Get("User-Agent"); got != "clave-update-service" {
					t.Fatalf("unexpected user agent: %q", got)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"tag_name": "v1.2.3",
						"html_url": "https://github.com/cheetahbyte/kepler-releases/releases/tag/v1.2.3",
						"assets": [{"browser_download_url": "https://github.com/cheetahbyte/kepler-releases/releases/download/v1.2.3/app.zip"}]
					}`)),
				}, nil
			}),
		},
	}

	release, err := svc.fetchLatestRelease(context.Background(), "cheetahbyte/kepler-releases")
	if err != nil {
		t.Fatalf("fetchLatestRelease returned error: %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("unexpected tag name: %q", release.TagName)
	}
	if got := release.Assets[0].BrowserDownloadURL; got != "https://github.com/cheetahbyte/kepler-releases/releases/download/v1.2.3/app.zip" {
		t.Fatalf("unexpected asset url: %q", got)
	}
}

func TestFetchLatestReleaseNotFound(t *testing.T) {
	svc := &UpdateService{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
				}, nil
			}),
		},
	}

	_, err := svc.fetchLatestRelease(context.Background(), "cheetahbyte/kepler-releases")
	if !errors.Is(err, errNoLatestRelease) {
		t.Fatalf("expected errNoLatestRelease, got %v", err)
	}
	var githubErr *githubReleaseError
	if !errors.As(err, &githubErr) {
		t.Fatalf("expected githubReleaseError, got %v", err)
	}
	if githubErr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected github status: %d", githubErr.StatusCode)
	}
	if githubErr.Body != `{"message":"Not Found"}` {
		t.Fatalf("unexpected github body: %q", githubErr.Body)
	}
}

func TestFetchLatestReleaseContextDeadline(t *testing.T) {
	svc := &UpdateService{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := svc.fetchLatestRelease(ctx, "cheetahbyte/kepler-releases")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestFetchLatestReleaseGitHubError(t *testing.T) {
	svc := &UpdateService{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"message":"API rate limit exceeded"}`)),
				}, nil
			}),
		},
	}

	_, err := svc.fetchLatestRelease(context.Background(), "cheetahbyte/kepler-releases")
	var githubErr *githubReleaseError
	if !errors.As(err, &githubErr) {
		t.Fatalf("expected githubReleaseError, got %v", err)
	}
	if githubErr.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected github status: %d", githubErr.StatusCode)
	}
	if githubErr.Body != `{"message":"API rate limit exceeded"}` {
		t.Fatalf("unexpected github body: %q", githubErr.Body)
	}
}
