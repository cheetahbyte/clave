package update

import (
	"encoding/json"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
)

// NativeFeed is the Clave Native update feed. Unlike the Sparkle appcast (RSS
// XML), this is a custom JSON document describing all published releases for a
// product/platform/channel, including full artifact and rollout metadata.
type NativeFeed struct {
	Schema      string              `json:"schema"`
	Product     string              `json:"product"`
	Platform    string              `json:"platform"`
	Channel     string              `json:"channel"`
	GeneratedAt string              `json:"generatedAt"`
	Releases    []NativeFeedRelease `json:"releases"`
}

type NativeFeedRelease struct {
	Version              string               `json:"version"`
	Build                string               `json:"build,omitempty"`
	ReleaseNotes         string               `json:"releaseNotes,omitempty"`
	Changelog            string               `json:"changelog,omitempty"`
	ChangelogURL         string               `json:"changelogUrl,omitempty"`
	PublishedAt          string               `json:"publishedAt,omitempty"`
	Mandatory            bool                 `json:"mandatory"`
	RolloutPercentage    int32                `json:"rolloutPercentage"`
	MinimumSystemVersion string               `json:"minimumSystemVersion,omitempty"`
	Artifacts            []NativeFeedArtifact `json:"artifacts"`
}

type NativeFeedArtifact struct {
	Type      string `json:"type"`
	Arch      string `json:"arch,omitempty"`
	URL       string `json:"url"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// NativeFeedReleaseInput bundles a release with its artifacts and rollout policy.
type NativeFeedReleaseInput struct {
	Release      db.UpdateRelease
	Artifacts    []db.UpdateArtifact
	Policy       db.UpdateReleasePolicy
	Changelog    string
	ChangelogURL string
}

const nativeFeedSchema = "clave.native.feed/v1"

// GenerateNativeFeed renders the custom Clave Native JSON feed.
func GenerateNativeFeed(product db.Product, platform, channel string, releases []NativeFeedReleaseInput) ([]byte, error) {
	feed := NativeFeed{
		Schema:      nativeFeedSchema,
		Product:     product.Name,
		Platform:    platform,
		Channel:     channel,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Releases:    make([]NativeFeedRelease, 0, len(releases)),
	}

	for _, r := range releases {
		fr := NativeFeedRelease{
			Version:              r.Release.Version,
			Build:                safeString(r.Release.BuildNumber),
			ReleaseNotes:         safeString(r.Release.ReleaseNotes),
			Changelog:            r.Changelog,
			ChangelogURL:         r.ChangelogURL,
			Mandatory:            r.Policy.Mandatory,
			RolloutPercentage:    r.Policy.RolloutPercentage,
			MinimumSystemVersion: safeString(r.Policy.MinSupportedVersion),
			Artifacts:            make([]NativeFeedArtifact, 0, len(r.Artifacts)),
		}
		// Default to full rollout when no policy row exists.
		if r.Policy.ReleaseID == [16]byte{} && fr.RolloutPercentage == 0 {
			fr.RolloutPercentage = 100
		}
		if r.Release.PublishedAt.Valid {
			fr.PublishedAt = r.Release.PublishedAt.Time.UTC().Format(time.RFC3339)
		}

		for _, a := range r.Artifacts {
			art := NativeFeedArtifact{
				Type:      a.ArtifactType,
				Arch:      a.Arch,
				URL:       a.Url,
				SHA256:    safeString(a.ChecksumSha256),
				Signature: safeString(a.Signature),
			}
			if a.SizeBytes != nil {
				art.SizeBytes = *a.SizeBytes
			}
			fr.Artifacts = append(fr.Artifacts, art)
		}

		feed.Releases = append(feed.Releases, fr)
	}

	return json.MarshalIndent(feed, "", "  ")
}
