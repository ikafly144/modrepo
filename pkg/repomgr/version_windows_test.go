//go:build windows

package repomgr

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	repoDir, err := GetRepoDir()
	if err != nil {
		t.Skipf("skipping: failed to locate REPO directory: %v", err)
	}

	version, err := GetVersion(repoDir)
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if version != "v0.4.4.3" {
		t.Fatalf("expected version %q, got %q", "v0.4.4.3", version)
	}
	t.Logf("Successfully retrieved game version: %s", version)
}
