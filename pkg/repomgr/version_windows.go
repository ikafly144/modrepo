//go:build windows

package repomgr

import (
	"fmt"
	"path/filepath"

	"github.com/ikafly144/modrepo/pkg/assetstools"
)

func GetVersion(gamePath string) (string, error) {
	sharedAssetsPath := filepath.Join(gamePath, DataDirName, "sharedassets0.assets")
	version, err := assetstools.ReadReleaseVersion(sharedAssetsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read game version from %s: %w", sharedAssetsPath, err)
	}
	return version, nil
}
