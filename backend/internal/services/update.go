package services

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

	"github.com/cheetahbyte/clave/internal/handlers/dto"
	problem "github.com/cheetahbyte/problems"
)

type UpdateService struct {
	licenseService *LicenseService
	signingService *SigningService
	httpClient     *http.Client
}

func NewUpdateService(licenseService *LicenseService, signingService *SigningService) *UpdateService {
	return &UpdateService{
		licenseService: licenseService,
		signingService: signingService,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

var errNoLatestRelease = errors.New("no published latest release found")

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	HTMLURL string `json:"html_url"`
}

func (svc *UpdateService) CheckUpdate(ctx context.Context, data dto.UpdateCheckRequest) (dto.UpdateCheckResponse, error) {
	instance := "/updates/check"

	claims, err := svc.signingService.ParseJWT(data.Token)
	if err != nil {
		return dto.UpdateCheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseID, err := licenseIDFromSubject(claims.Subject)
	if err != nil {
		return dto.UpdateCheckResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	license, err := svc.licenseService.GetLicenseById(ctx, licenseID)
	if err != nil || license == nil {
		return dto.UpdateCheckResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !license.IsActive {
		return dto.UpdateCheckResponse{}, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Instance(instance))
	}

	repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	if repo == "" {
		slog.Error("github repo is not configured")
		return dto.UpdateCheckResponse{}, problem.Of(500).
			Append(problem.Title("Update repository not configured")).
			Append(problem.Detail("GITHUB_REPO must be set to owner/repo")).
			Append(problem.Instance(instance))
	}

	release, err := svc.fetchLatestRelease(ctx, repo)
	if err != nil {
		slog.Error("failed to fetch github release info", "repo", repo, "err", err)
		if errors.Is(err, errNoLatestRelease) {
			return dto.UpdateCheckResponse{}, problem.Of(404).
				Append(problem.Title("No release found")).
				Append(problem.Detail("No published GitHub release exists for the configured repository")).
				Append(problem.Instance(instance))
		}
		return dto.UpdateCheckResponse{}, problem.Of(502).
			Append(problem.Title("Failed to fetch release info")).
			Append(problem.Detail("GitHub release information could not be fetched")).
			Append(problem.Instance(instance))
	}

	downloadURL := release.HTMLURL
	if len(release.Assets) > 0 {
		downloadURL = release.Assets[0].BrowserDownloadURL
	}

	return dto.UpdateCheckResponse{
		CurrentVersion:  data.Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: release.TagName != data.Version,
		DownloadURL:     downloadURL,
	}, nil
}

func (svc *UpdateService) fetchLatestRelease(ctx context.Context, repo string) (*githubRelease, error) {
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

	resp, err := svc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: github api returned %d: %s", errNoLatestRelease, resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
