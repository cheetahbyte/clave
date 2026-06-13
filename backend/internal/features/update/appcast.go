package update

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
)

type AppcastRelease struct {
	Release      db.UpdateRelease
	Artifacts    []db.UpdateArtifact
	ChangelogURL string
}

type AppcastEnclosure struct {
	XMLName   xml.Name `xml:"enclosure"`
	URL       string   `xml:"url,attr"`
	Type      string   `xml:"type,attr"`
	Length    int64    `xml:"length,attr"`
	Signature *string  `xml:"sparkle:edSignature,attr,omitempty"`
}

type AppcastItem struct {
	XMLName            xml.Name          `xml:"item"`
	Title              string            `xml:"title"`
	Description        string            `xml:"description"`
	PubDate            string            `xml:"pubDate"`
	BuildNumber        string            `xml:"sparkle:version"`
	ShortVersionString string            `xml:"sparkle:shortVersionString"`
	ReleaseNotesLink   string            `xml:"sparkle:releaseNotesLink,omitempty"`
	Enclosure          *AppcastEnclosure `xml:"enclosure,omitempty"`
}

type AppcastChannel struct {
	XMLName xml.Name      `xml:"channel"`
	Title   string        `xml:"title"`
	Items   []AppcastItem `xml:"item"`
}

type AppcastFeed struct {
	XMLName xml.Name       `xml:"rss"`
	XMLNS   string         `xml:"xmlns:sparkle,attr"`
	Version string         `xml:"version,attr"`
	Channel AppcastChannel `xml:"channel"`
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mimeTypeForArtifact(artifactType string) string {
	switch artifactType {
	case "dmg":
		return "application/x-apple-diskimage"
	case "zip":
		return "application/zip"
	case "pkg":
		return "application/x-apple-installer"
	case "exe":
		return "application/x-msdownload"
	case "msi":
		return "application/x-msi"
	case "deb":
		return "application/vnd.debian.binary-package"
	case "appimage":
		return "application/x-executable"
	default:
		return "application/octet-stream"
	}
}

func GenerateAppcastXML(product db.Product, platform, channel string, releases []AppcastRelease, selfURL string) ([]byte, error) {
	feed := AppcastFeed{
		XMLNS:   "http://www.andymatuschak.org/xml-namespaces/sparkle",
		Version: "2.0",
		Channel: AppcastChannel{
			Title: fmt.Sprintf("%s — %s (%s)", product.Name, platform, channel),
			Items: []AppcastItem{},
		},
	}

	for _, rel := range releases {
		buildNumber := safeString(rel.Release.BuildNumber)
		if buildNumber == "" {
			buildNumber = rel.Release.Version
		}

		item := AppcastItem{
			Title:              fmt.Sprintf("%s %s", product.Name, rel.Release.Version),
			Description:        "",
			PubDate:            rel.Release.PublishedAt.Time.Format(time.RFC1123),
			BuildNumber:        buildNumber,
			ShortVersionString: rel.Release.Version,
		}

		if rel.Release.ReleaseNotes != nil && *rel.Release.ReleaseNotes != "" {
			item.Description = *rel.Release.ReleaseNotes
		}

		// Sparkle fetches and renders the changelog from this link instead of
		// inlining it; richer than the plain-text <description> summary.
		if rel.ChangelogURL != "" {
			item.ReleaseNotesLink = rel.ChangelogURL
		}

		for _, a := range rel.Artifacts {
			if a.ArtifactType == "dmg" || a.ArtifactType == "zip" || a.ArtifactType == "pkg" {
				enclosure := &AppcastEnclosure{
					URL:    a.Url,
					Type:   mimeTypeForArtifact(a.ArtifactType),
					Length: 0,
				}
				if a.SizeBytes != nil {
					enclosure.Length = *a.SizeBytes
				}
				if a.SparkleEdSignature != nil && *a.SparkleEdSignature != "" {
					enclosure.Signature = a.SparkleEdSignature
				}
				item.Enclosure = enclosure
				break
			}
		}

		feed.Channel.Items = append(feed.Channel.Items, item)
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}

	header := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	return append(header, output...), nil
}
