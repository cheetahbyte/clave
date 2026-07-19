package update

import (
	"testing"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

func TestNormalizeDownloadPlatform(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"macos", "macos", true},
		{"WINDOWS", "windows", true},
		{" linux ", "linux", true},
		{"", "", false},
		{"freebsd", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := normalizeDownloadPlatform(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("normalizeDownloadPlatform(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSelectFullDownloadArtifact(t *testing.T) {
	universalID := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	artifacts := []db.UpdateArtifact{
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), ArtifactType: "dmg", Os: "windows", Arch: "universal"},
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), ArtifactType: "delta", Os: "macos", Arch: "universal"},
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), ArtifactType: "dmg", Os: "macos", Arch: "arm64"},
		{ID: universalID, ArtifactType: "dmg", Os: "macos", Arch: "universal"},
	}

	got, ok := selectFullDownloadArtifact(artifacts, "macos")
	if !ok || got.ID != universalID {
		t.Fatalf("selected (%s, %v), want universal artifact %s", got.ID, ok, universalID)
	}

	got, ok = selectFullDownloadArtifact([]db.UpdateArtifact{
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000006"), ArtifactType: "zip", Os: "linux", Arch: "x64"},
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000005"), ArtifactType: "zip", Os: "linux", Arch: "arm64"},
	}, "linux")
	if !ok || got.Arch != "arm64" {
		t.Fatalf("selected (%q, %v), want deterministic arm64 fallback", got.Arch, ok)
	}

	if _, ok := selectFullDownloadArtifact(artifacts, "linux"); ok {
		t.Fatal("expected no matching Linux artifact")
	}
}

func TestDownloadFeatureAccess(t *testing.T) {
	if !hasAllFeatures([]string{"pro", "beta"}, []string{"pro"}) {
		t.Fatal("expected matching feature set to authorize the download")
	}
	if hasAllFeatures([]string{"basic"}, []string{"pro"}) {
		t.Fatal("expected missing required feature to deny the download")
	}
}
