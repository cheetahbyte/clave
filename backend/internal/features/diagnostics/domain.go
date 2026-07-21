package diagnostics

import (
	"time"

	"github.com/google/uuid"
)

type Checkin struct {
	OrganizationID uuid.UUID
	ProductID      uuid.UUID
	LicenseID      uuid.UUID
	ActivationID   uuid.UUID
	Version        string
	Build          string
	Platform       string
	Arch           string
	OSVersion      string
}

type LatestCheckin struct {
	ActivationID uuid.UUID
	Hostname     *string
	Version      string
	Build        *string
	Platform     *string
	Arch         *string
	OSVersion    *string
	CreatedAt    time.Time
}

type DailyVersion struct {
	Date        time.Time
	Version     string
	DeviceCount int64
}

type VersionDistribution struct {
	Version     string  `json:"version"`
	DeviceCount int64   `json:"deviceCount"`
	Percentage  float64 `json:"percentage"`
}

type VersionTrendValue struct {
	Version     string `json:"version"`
	DeviceCount int64  `json:"deviceCount"`
}

type VersionTrendPoint struct {
	Date     string              `json:"date"`
	Versions []VersionTrendValue `json:"versions"`
}

type VersionDevice struct {
	ActivationID string    `json:"activationId"`
	Hostname     *string   `json:"hostname"`
	Version      string    `json:"version"`
	Build        *string   `json:"build"`
	Platform     *string   `json:"platform"`
	Arch         *string   `json:"arch"`
	OSVersion    *string   `json:"osVersion"`
	LastCheckin  time.Time `json:"lastCheckin"`
}

type VersionAdoptionResponse struct {
	ActiveDevices int64                 `json:"activeDevices"`
	VersionCount  int                   `json:"versionCount"`
	Distribution  []VersionDistribution `json:"distribution"`
	Trend         []VersionTrendPoint   `json:"trend"`
	Devices       []VersionDevice       `json:"devices"`
}

type SigningKeyResponse struct {
	Algorithm   string `json:"algorithm"`
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
}
