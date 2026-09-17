package versioning

import (
	"strings"

	"golang.org/x/mod/semver"

	restcommon "github.com/ikafly144/modrepo/common/rest"
)

// LatestVersionsFromTags returns the latest tag per branch based on semver ordering.
// Tags that are not valid semver are ignored.
func LatestVersionsFromTags(tags []string) map[Branch]string {
	latest := make(map[Branch]string)
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || !semver.IsValid(tag) {
			continue
		}
		prerelease := strings.TrimPrefix(semver.Prerelease(tag), "-")
		before, _, _ := strings.Cut(prerelease, ".")
		for b := BranchStable; b <= BranchDev; b++ {
			if !b.match(before) {
				continue
			}
			if current, ok := latest[b]; !ok || semver.Compare(tag, current) > 0 {
				latest[b] = tag
			}
		}
	}
	return latest
}

// FindBranchVersion searches VersionInfo for the specified branch name.
func FindBranchVersion(info *restcommon.VersionInfo, branch string) string {
	b := FindBranchInfo(info, branch)
	if b != nil {
		return b.Version
	}
	return ""
}

// FindBranchInfo searches VersionInfo for the specified branch name and returns the BranchInfo.
func FindBranchInfo(info *restcommon.VersionInfo, branch string) *restcommon.BranchInfo {
	if info == nil {
		return nil
	}
	for i := range info.Branches {
		if strings.EqualFold(info.Branches[i].Name, branch) {
			return &info.Branches[i]
		}
	}
	return nil
}
