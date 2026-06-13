package update

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
)

// SparkleItemInput carries the data for a single Sparkle appcast <item>.
type SparkleItemInput struct {
	Version              string
	Build                string
	ReleaseNotesLink     string
	PubDate              time.Time
	DownloadURL          string
	SparkleVersion       string
	ShortVersionString   string
	MinimumSystemVersion string
	Length               int64
	MIMEType             string
}

// SparkleFeedInput bundles a release with its artifacts and policy.
type SparkleFeedInput struct {
	Release   db.UpdateRelease
	Artifacts []db.UpdateArtifact
	Policy    db.UpdateReleasePolicy
	ChangelogBody    string
	ChangelogURL     string
}

const sparkleXMLNS = "http://www.andymatuschak.org/xml-namespaces/sparkle"

// GenerateSparkleAppcast renders a Sparkle-compatible RSS 2.0 appcast XML
// document. arch is the requested architecture filter; the best-matching
// artifact (exact arch > universal) is picked per release. Only downloadable
// full-file artifact types (dmg, zip, pkg, exe, msi, deb, appimage) are in-
// cluded. Delta artifacts are silently skipped.
func GenerateSparkleAppcast(product db.Product, channel string, inputs []SparkleFeedInput, arch string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString(`<rss version="2.0" xmlns:sparkle="` + sparkleXMLNS + `">` + "\n")
	buf.WriteString("<channel>\n")
	buf.WriteString(fmt.Sprintf("  <title>%s</title>\n", xmlEscaped(product.Name)))
	buf.WriteString(fmt.Sprintf("  <description>%s %s updates</description>\n", xmlEscaped(product.Name), xmlEscaped(channel)))
	buf.WriteString("  <language>en</language>\n")

	for _, in := range inputs {
		art := pickArtifact(in.Artifacts, arch)
		if art == nil {
			continue
		}

		sparkleVersion := in.Release.Version
		if in.Release.BuildNumber != nil && *in.Release.BuildNumber != "" {
			sparkleVersion = *in.Release.BuildNumber
		}

		mime := mimeTypeForArtifact(art.ArtifactType)

		minSysVersion := ""
		if in.Policy.MinSupportedVersion != nil && *in.Policy.MinSupportedVersion != "" {
			minSysVersion = *in.Policy.MinSupportedVersion
		}

		changelogLink := in.ChangelogURL
		if changelogLink == "" && in.ChangelogBody != "" {
			changelogLink = in.ChangelogBody
		}

		var size int64
		if art.SizeBytes != nil {
			size = *art.SizeBytes
		}

		buf.WriteString("  <item>\n")
		buf.WriteString(fmt.Sprintf("    <title>%s</title>\n", xmlEscaped(in.Release.Version)))
		if in.Release.PublishedAt.Valid {
			buf.WriteString(fmt.Sprintf("    <pubDate>%s</pubDate>\n", in.Release.PublishedAt.Time.UTC().Format(time.RFC1123Z)))
		}
		if changelogLink != "" {
			if looksLikeURL(changelogLink) {
				buf.WriteString(fmt.Sprintf("    <sparkle:releaseNotesLink>%s</sparkle:releaseNotesLink>\n", xmlEscaped(changelogLink)))
			} else {
				buf.WriteString(fmt.Sprintf("    <description><![CDATA[%s]]></description>\n", changelogLink))
			}
		}

		attrs := fmt.Sprintf(` url="%s" sparkle:version="%s" sparkle:shortVersionString="%s"`,
			attrEscape(art.Url),
			attrEscape(sparkleVersion),
			attrEscape(in.Release.Version))
		if minSysVersion != "" {
			attrs += fmt.Sprintf(` sparkle:minimumSystemVersion="%s"`, attrEscape(minSysVersion))
		}
		if art.Signature != nil && *art.Signature != "" {
			attrs += fmt.Sprintf(` sparkle:edSignature="%s"`, attrEscape(*art.Signature))
		}
		attrs += fmt.Sprintf(` length="%d"`, size)
		attrs += fmt.Sprintf(` type="%s"`, attrEscape(mime))
		buf.WriteString(fmt.Sprintf("    <enclosure%s/>\n", attrs))
		buf.WriteString("  </item>\n")
	}

	buf.WriteString("</channel>\n")
	buf.WriteString("</rss>\n")
	return buf.Bytes(), nil
}

// pickArtifact returns the best-matching full-file artifact for the requested
// architecture. It prefers an exact arch match, falls back to "universal", and
// skips non-downloadable types as well as delta artifacts.
func pickArtifact(artifacts []db.UpdateArtifact, arch string) *db.UpdateArtifact {
	if arch == "" {
		arch = "universal"
	}
	var universal *db.UpdateArtifact
	for i := range artifacts {
		a := &artifacts[i]
		if !isFullArtifact(a.ArtifactType) {
			continue
		}
		if strings.EqualFold(a.Arch, arch) {
			return a
		}
		if universal == nil && strings.EqualFold(a.Arch, "universal") {
			universal = a
		}
	}
	if universal != nil {
		return universal
	}
	// Last resort: return the first compatible artifact regardless of arch.
	for i := range artifacts {
		a := &artifacts[i]
		if isFullArtifact(a.ArtifactType) {
			return a
		}
	}
	return nil
}

// isFullArtifact returns true for artifact types that represent downloadable
// full installers (as opposed to deltas or metadata-only entries).
func isFullArtifact(artifactType string) bool {
	switch artifactType {
	case "dmg", "zip", "pkg", "exe", "msi", "deb", "appimage":
		return true
	default:
		return false
	}
}

func xmlEscaped(s string) string {
	return html.EscapeString(s)
}

func attrEscape(s string) string {
	return html.EscapeString(s)
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
