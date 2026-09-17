package versioning

import (
	"testing"

	restcommon "github.com/ikafly144/modrepo/common/rest"
)

func TestFindBranchInfo(t *testing.T) {
	info := &restcommon.VersionInfo{
		Branches: []restcommon.BranchInfo{
			{Name: "stable", Version: "v1.0.0", Title: "Release 1.0.0", ReleaseNotes: "Initial release"},
			{Name: "preview", Version: "v1.1.0-rc.1", Title: "RC 1", ReleaseNotes: "Testing RC"},
		},
	}

	stable := FindBranchInfo(info, "stable")
	if stable == nil || stable.Version != "v1.0.0" {
		t.Fatalf("expected stable v1.0.0, got %+v", stable)
	}
	if stable.Title != "Release 1.0.0" || stable.ReleaseNotes != "Initial release" {
		t.Fatalf("unexpected branch fields: %+v", stable)
	}

	preview := FindBranchInfo(info, "preview")
	if preview == nil || preview.Version != "v1.1.0-rc.1" {
		t.Fatalf("expected preview v1.1.0-rc.1, got %+v", preview)
	}

	notFound := FindBranchInfo(info, "dev")
	if notFound != nil {
		t.Fatalf("expected nil for dev branch, got %+v", notFound)
	}
}

func TestLatestVersionsFromTags(t *testing.T) {
	tags := []string{
		"v1.0.0",
		"v1.0.1",
		"v1.1.0-rc.1",
		"v1.1.0-rc.2",
		"v1.2.0-pre.1",
		"v1.3.0-beta.1",
		"v1.4.0-alpha.1",
		"invalid-tag",
	}

	latest := LatestVersionsFromTags(tags)

	if latest[BranchStable] != "v1.0.1" {
		t.Errorf("expected stable v1.0.1, got %s", latest[BranchStable])
	}
	if latest[BranchPreview] != "v1.1.0-rc.2" {
		t.Errorf("expected preview v1.1.0-rc.2, got %s", latest[BranchPreview])
	}
	if latest[BranchBeta] != "v1.2.0-pre.1" {
		t.Errorf("expected beta v1.2.0-pre.1, got %s", latest[BranchBeta])
	}
	if latest[BranchCanary] != "v1.3.0-beta.1" {
		t.Errorf("expected canary v1.3.0-beta.1, got %s", latest[BranchCanary])
	}
	if latest[BranchDev] != "v1.4.0-alpha.1" {
		t.Errorf("expected dev v1.4.0-alpha.1, got %s", latest[BranchDev])
	}
}

func TestBranchFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Branch
	}{
		{"stable", BranchStable},
		{"preview", BranchPreview},
		{"beta", BranchBeta},
		{"canary", BranchCanary},
		{"dev", BranchDev},
		{"unknown", BranchStable},
	}

	for _, tc := range tests {
		got := BranchFromString(tc.input)
		if got != tc.want {
			t.Errorf("BranchFromString(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
