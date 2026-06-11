package update

type CheckRequest struct {
	Token   string `json:"token" validate:"required"`
	Version string `json:"version" validate:"required"`
}

type CheckResponse struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl"`
}
