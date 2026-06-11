package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	problem "github.com/cheetahbyte/problems"

	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/shared/signing"
)

const githubReleaseFetchTimeout = 2 * time.Second

var errNoLatestRelease = errors.New("no published latest release found")

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	HTMLURL string `json:"html_url"`
}

type githubReleaseError struct {
	StatusCode int
	Body       string
	URL        string
}

func (err *githubReleaseError) Error() string {
	return fmt.Sprintf("github api returned %d for %s: %s", err.StatusCode, err.URL, err.Body)
}

type Service struct {
	licenses *license.Service
	signer   *signing.Service
	client   *http.Client
}

func NewService(licenses *license.Service, signer *signing.Service) *Service {
	return &Service{
		licenses: licenses,
		signer:   signer,
		client:   &http.Client{Timeout: githubReleaseFetchTimeout},
	}
}

func (svc *Service) Check(ctx context.Context, data CheckRequest) (CheckResponse, error) {
	instance := "/updates/check"

	claims, err := svc.signer.ParseJWT(data.Token)
	if err != nil {
		return CheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseID, err := license.LicenseIDFromSubject(claims.Subject)
	if err != nil {
		return CheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	lic, err := svc.licenses.GetByID(ctx, licenseID)
	if err != nil || lic == nil {
		return CheckResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !lic.Active {
		return CheckResponse{}, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Instance(instance))
	}

	repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	if repo == "" {
		slog.Error("github repo is not configured")
		return CheckResponse{}, problem.Of(500).
			Append(problem.Title("Update repository not configured")).
			Append(problem.Detail("GITHUB_REPO must be set to owner/repo")).
			Append(problem.Instance(instance))
	}

	githubCtx, cancel := context.WithTimeout(ctx, githubReleaseFetchTimeout)
	defer cancel()

	release, err := svc.fetchLatestRelease(githubCtx, repo)
	if err != nil {
		slog.Error("failed to fetch github release info", "repo", repo, "err", err)
		if errors.Is(err, errNoLatestRelease) {
			return CheckResponse{}, problem.Of(404).
				Append(problem.Title("No release found")).
				Append(problem.Detail("No published latest release is available for the configured repository.")).
				Append(problem.Instance(instance))
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || os.IsTimeout(err) {
			return CheckResponse{}, problem.Of(504).
				Append(problem.Title("Release lookup timed out")).
				Append(problem.Detail("GitHub release information could not be fetched before the request deadline")).
				Append(problem.Instance(instance))
		}
		return CheckResponse{}, problem.Of(502).
			Append(problem.Title("Failed to fetch release info")).
			Append(problem.Detail("GitHub release information could not be fetched")).
			Append(problem.Instance(instance))
	}

	downloadURL := release.HTMLURL
	if len(release.Assets) > 0 {
		downloadURL = release.Assets[0].BrowserDownloadURL
	}

	return CheckResponse{
		CurrentVersion:  data.Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: release.TagName != data.Version,
		DownloadURL:     downloadURL,
	}, nil
}

func (svc *Service) fetchLatestRelease(ctx context.Context, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "clave-update-service")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := svc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		githubErr := &githubReleaseError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
			URL:        url,
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %w", errNoLatestRelease, githubErr)
		}
		return nil, githubErr
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
