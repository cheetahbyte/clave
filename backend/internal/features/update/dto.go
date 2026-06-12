package update

type CheckRequest struct {
	Token   string `json:"token" validate:"required"`
	Version string `json:"version" validate:"required"`

	Build     string `json:"build,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Arch      string `json:"arch,omitempty"`
	OSVersion string `json:"osVersion,omitempty"`
	ClientID  string `json:"clientId,omitempty"`
}

type CheckResponse struct {
	CurrentVersion  string        `json:"currentVersion"`
	LatestVersion   string        `json:"latestVersion"`
	UpdateAvailable bool          `json:"updateAvailable"`
	DownloadURL     string        `json:"downloadUrl,omitempty"`
	Kind            string        `json:"kind,omitempty"`
	ReleaseNotes    string        `json:"releaseNotes,omitempty"`
	Artifacts       []ArtifactDTO `json:"artifacts,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type SaveProductUpdateConfigRequest struct {
	Platform    string `json:"platform" validate:"required"`
	Channel     string `json:"channel" validate:"required"`
	ProviderKey string `json:"providerKey" validate:"required"`
	Enabled     bool   `json:"enabled"`
	Config      map[string]any `json:"config"`
}

type ProductUpdateConfigDTO struct {
	ID          string                 `json:"id"`
	ProductID   string                 `json:"productId"`
	Platform    string                 `json:"platform"`
	Channel     string                 `json:"channel"`
	ChannelID   string                 `json:"channelId"`
	ProviderKey string                 `json:"providerKey"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	AppcastURL  string                 `json:"appcastUrl,omitempty"`
}

type StorageConfigDTO struct {
	ProductID string         `json:"productId"`
	Backend   string         `json:"backend"`
	Config    map[string]any `json:"config"`
}

type SaveStorageConfigRequest struct {
	Backend string         `json:"backend" validate:"required"`
	Config  map[string]any `json:"config"`
}

type CreateReleaseRequest struct {
	ProductID    string `json:"productId" validate:"required"`
	Platform     string `json:"platform" validate:"required"`
	Channel      string `json:"channel" validate:"required"`
	Version      string `json:"version" validate:"required"`
	BuildNumber  string `json:"buildNumber,omitempty"`
	ReleaseNotes string `json:"releaseNotes,omitempty"`
}

type ReleaseDTO struct {
	ID           string         `json:"id"`
	ProductID    string         `json:"productId"`
	ProductName  string         `json:"productName"`
	Channel      string         `json:"channel"`
	ChannelID    string         `json:"channelId"`
	Platform     string         `json:"platform"`
	Version      string         `json:"version"`
	BuildNumber  string         `json:"buildNumber,omitempty"`
	Status       string         `json:"status"`
	ReleaseNotes string         `json:"releaseNotes,omitempty"`
	PublishedAt  *string        `json:"publishedAt,omitempty"`
	CreatedAt    *string        `json:"createdAt,omitempty"`
	Artifacts    []ArtifactDTO  `json:"artifacts,omitempty"`
}

type ArtifactDTOFull struct {
	ID                   string  `json:"id"`
	ReleaseID            string  `json:"releaseId"`
	ArtifactType         string  `json:"artifactType"`
	OS                   string  `json:"os"`
	Arch                 string  `json:"arch"`
	URL                  string  `json:"url"`
	SizeBytes            *int64  `json:"sizeBytes,omitempty"`
	ChecksumSHA256       *string `json:"checksumSha256,omitempty"`
	Signature            *string `json:"signature,omitempty"`
	Filename             *string `json:"filename,omitempty"`
	MimeType             *string `json:"mimeType,omitempty"`
	MinimumSystemVersion *string `json:"minimumSystemVersion,omitempty"`
	SparkleEdSignature   *string `json:"sparkleEdSignature,omitempty"`
}
