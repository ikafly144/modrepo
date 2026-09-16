//go:build windows

package assetstools

import (
	"encoding/binary"
	"testing"
)

func TestExtractBundleVersionFromSerializedData(t *testing.T) {
	// Synthesize serialized data with a length-prefixed version string matching \b\d{4}\.\d+\.\d+\b
	version := "2026.8.18"
	data := make([]byte, 100)
	binary.LittleEndian.PutUint32(data[10:14], uint32(len(version)))
	copy(data[14:], version)

	got, err := extractBundleVersionFromSerializedData(data)
	if err != nil {
		t.Fatalf("failed to extract bundle version: %v", err)
	}
	if got != version {
		t.Fatalf("expected %q, got %q", version, got)
	}
}

func TestReadReleaseVersion(t *testing.T) {
	repoDir, err := getRepoDir()
	if err != nil {
		t.Skipf("skipping: %v", err)
	}
	path := repoDir + "\\REPO_Data\\sharedassets0.assets"

	version, err := ReadReleaseVersion(path)
	if err != nil {
		t.Fatalf("failed to read release version: %v", err)
	}
	if version != "v0.4.4.3" {
		t.Fatalf("expected version %q, got %q", "v0.4.4.3", version)
	}
	t.Logf("release version: %s", version)
}
