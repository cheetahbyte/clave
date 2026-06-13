package update

type CheckRequest struct {
	Token   string `json:"token" validate:"required"`
	Version string `json:"version" validate:"required"`

	Build                   string `json:"build,omitempty"`
	Platform                string `json:"platform,omitempty"`
	Channel                 string `json:"channel,omitempty"`
	Arch                    string `json:"arch,omitempty"`
	OSVersion               string `json:"osVersion,omitempty"`
	ClientID                string `json:"clientId,omitempty"`
}

type CheckResponse struct {
	CurrentVersion  string        `json:"currentVersion"`
	LatestVersion   string        `json:"latestVersion"`
	UpdateAvailable bool          `json:"updateAvailable"`
	DownloadURL     string        `json:"downloadUrl,omitempty"`
	Kind            string        `json:"kind,omitempty"`
	ReleaseNotes    string        `json:"releaseNotes,omitempty"`
	Changelog       string        `json:"changelog,omitempty"`
	ChangelogURL    string        `json:"changelogUrl,omitempty"`
	Artifacts       []ArtifactDTO `json:"artifacts,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type ChannelDTO struct {
	ID               string   `json:"id"`
	ProductID        string   `json:"productId"`
	Name             string   `json:"name"`
	IsDefault        bool     `json:"isDefault"`
	RequiredFeatures []string `json:"requiredFeatures"`
	Description      string   `json:"description,omitempty"`
}

type SaveChannelRequest struct {
	Name             string   `json:"name" validate:"required"`
	IsDefault        bool     `json:"isDefault"`
	RequiredFeatures []string `json:"requiredFeatures"`
	Description      string   `json:"description,omitempty"`
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
	FeedURL     string                 `json:"feedUrl,omitempty"`
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
	ChangelogID  string `json:"changelogId,omitempty"`
}

type ChangelogDTO struct {
	ID        string  `json:"id"`
	ProductID string  `json:"productId"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	CreatedAt *string `json:"createdAt,omitempty"`
	UpdatedAt *string `json:"updatedAt,omitempty"`
}

type SaveChangelogRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body"`
}

type AttachChangelogRequest struct {
	ChangelogID *string `json:"changelogId"`
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
	ChangelogID  string         `json:"changelogId,omitempty"`
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
}
