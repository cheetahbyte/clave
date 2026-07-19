package update

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestServeArtifactDownloadSupportsRanges(t *testing.T) {
	t.Parallel()

	file := artifactTestFile(t, "0123456789")
	dl := &ArtifactDownload{
		Body:     file,
		Seeker:   file,
		Name:     "artifact.bin",
		ModTime:  time.Unix(1_700_000_000, 0),
		Size:     10,
		MimeType: "application/octet-stream",
	}
	req := httptest.NewRequest(http.MethodGet, "/artifact", nil)
	req.Header.Set("Range", "bytes=2-5")
	res := httptest.NewRecorder()

	ServeResolvedArtifactDownload(res, req, dl)

	if res.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusPartialContent)
	}
	if got := res.Body.String(); got != "2345" {
		t.Fatalf("body = %q, want %q", got, "2345")
	}
	if got := res.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q, want %q", got, "bytes 2-5/10")
	}
}

func TestServeArtifactDownloadSupportsHead(t *testing.T) {
	t.Parallel()

	file := artifactTestFile(t, "0123456789")
	dl := &ArtifactDownload{
		Body:     file,
		Seeker:   file,
		Name:     "artifact.bin",
		ModTime:  time.Unix(1_700_000_000, 0),
		Size:     10,
		MimeType: "application/octet-stream",
	}
	res := httptest.NewRecorder()

	ServeResolvedArtifactDownload(res, httptest.NewRequest(http.MethodHead, "/artifact", nil), dl)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if res.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want 0", res.Body.Len())
	}
	if got := res.Header().Get("Content-Length"); got != "10" {
		t.Fatalf("Content-Length = %q, want %q", got, "10")
	}
}

func TestServeResolvedArtifactDownloadRedirectsPrivately(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	res := httptest.NewRecorder()

	ServeResolvedArtifactDownload(res, req, &ArtifactDownload{RedirectURL: "https://storage.example/file"})

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "https://storage.example/file" {
		t.Fatalf("Location = %q", got)
	}
	if got := res.Header().Get("Cache-Control"); !strings.Contains(got, "private") {
		t.Fatalf("Cache-Control = %q, want private", got)
	}
}

func artifactTestFile(t *testing.T, contents string) *os.File {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "artifact-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	if _, err := io.WriteString(file, contents); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	return file
}
