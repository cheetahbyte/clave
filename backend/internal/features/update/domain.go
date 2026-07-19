package update

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type ProviderKey string

const (
	ProviderClaveNative ProviderKey = "clave_native"
)

type DeliveryProtocol string

const (
	DeliveryClaveNative DeliveryProtocol = "clave_native"
)

func ValidDeliveryProtocol(s string) bool {
	switch DeliveryProtocol(s) {
	case DeliveryClaveNative:
		return true
	default:
		return false
	}
}

type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
)

type DecisionKind string

const (
	DecisionNoUpdate        DecisionKind = "no_update"
	DecisionUpdateAvailable DecisionKind = "update_available"
	DecisionMandatoryUpdate DecisionKind = "mandatory_update"
	DecisionUnsupported     DecisionKind = "unsupported"
	DecisionError           DecisionKind = "error"
)

type CapabilitySupport string

const (
	CapabilityNative      CapabilitySupport = "native"
	CapabilityEmulated    CapabilitySupport = "emulated"
	CapabilityUnsupported CapabilitySupport = "unsupported"
)

type CapabilityMatrix struct {
	CodeSigning      CapabilitySupport `json:"codeSigning"`
	StagedRollouts   CapabilitySupport `json:"stagedRollouts"`
	MandatoryUpdates CapabilitySupport `json:"mandatoryUpdates"`
	ReleaseNotes     CapabilitySupport `json:"releaseNotes"`
	Channels         CapabilitySupport `json:"channels"`
	ServerSideChecks CapabilitySupport `json:"serverSideChecks"`
}

type ProviderInfo struct {
	Key          string           `json:"key"`
	Name         string           `json:"name"`
	Capabilities CapabilityMatrix `json:"capabilities"`
}

type ProviderConfig struct {
	OrganizationID uuid.UUID
	ProductID      uuid.UUID
	Platform       Platform
	Channel        string
	ChannelID      uuid.UUID
	Raw            json.RawMessage
}

type UpdateRequest struct {
	LicenseID      string
	ProductID      uuid.UUID
	Platform       Platform
	Channel        string
	CurrentVersion string
	CurrentBuild   string
	Arch           string
	OSVersion      string
	ClientID       string
}

type ArtifactDTO struct {
	Type      string         `json:"type,omitempty"`
	URL       string         `json:"url,omitempty"`
	Arch      string         `json:"arch,omitempty"`
	OS        string         `json:"os,omitempty"`
	SizeBytes int64          `json:"sizeBytes,omitempty"`
	SHA256    string         `json:"sha256,omitempty"`
	Signature string         `json:"signature,omitempty"`
	Filename  string         `json:"filename,omitempty"`
	MimeType  string         `json:"mimeType,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type UpdateDecision struct {
	Kind            DecisionKind   `json:"kind"`
	CurrentVersion  string         `json:"currentVersion"`
	LatestVersion   string         `json:"latestVersion"`
	UpdateAvailable bool           `json:"updateAvailable"`
	DownloadURL     string         `json:"downloadUrl,omitempty"`
	ReleaseNotes    string         `json:"releaseNotes,omitempty"`
	Changelog       string         `json:"changelog,omitempty"`
	ChangelogURL    string         `json:"changelogUrl,omitempty"`
	Artifacts       []ArtifactDTO  `json:"artifacts,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type UpdateProvider interface {
	Key() ProviderKey
	Name() string
	Capabilities(ctx context.Context) CapabilityMatrix
	ValidateConfig(ctx context.Context, config ProviderConfig) error
	CheckForUpdate(ctx context.Context, req UpdateRequest, config ProviderConfig) (*UpdateDecision, error)
}
