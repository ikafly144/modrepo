//go:build windows

package assetstools

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func getRepoDirForTest() (string, error) {
	// 1. Try registry
	if key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Steam App 3241660`, registry.QUERY_VALUE); err == nil {
		defer key.Close()
		if val, _, err := key.GetStringValue("InstallLocation"); err == nil && val != "" {
			if _, err := os.Stat(filepath.Join(val, "REPO_Data", "globalgamemanagers")); err == nil {
				return val, nil
			}
		}
	}

	// 2. Common Steam paths
	candidates := []string{
		`C:\Program Files (x86)\Steam\steamapps\common\REPO`,
		`C:\Program Files\Steam\steamapps\common\REPO`,
		`D:\SteamLibrary\steamapps\common\REPO`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "REPO_Data", "globalgamemanagers")); err == nil {
			return c, nil
		}
	}

	return "", os.ErrNotExist
}
