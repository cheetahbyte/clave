package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

	license, err := svc.licenseService.GetLicenseById(ctx, licenseID.Int32)
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

	release, err := svc.fetchLatestRelease(ctx, os.Getenv("GITHUB_REPO"))
	if err != nil {
		return dto.UpdateCheckResponse{}, problem.Of(502).
			Append(problem.Title("Failed to fetch release info")).
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
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := svc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
