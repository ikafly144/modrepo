//go:build windows

package assetstools

import "testing"

func TestReadPlayerSettingsBundleVersion(t *testing.T) {
	gamePath, err := getAmongUsDir()
	if err != nil {
		t.Skipf("skipping: failed to locate Among Us directory: %v", err)
	}
	path := gamePath + "Among Us_Data\\globalgamemanagers"

	version, err := ReadPlayerSettingsBundleVersion(path)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}
	if version == "" {
		t.Fatal("version must not be empty")
	}
	t.Logf("version: %s", version)
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

