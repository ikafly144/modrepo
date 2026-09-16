//go:build windows

package assetstools

import (
	"fmt"
	"os"
)

func getRepoDir() (string, error) {
	commonPaths := []string{
		`C:\Program Files (x86)\Steam\steamapps\common\REPO`,
		`C:\Program Files\Steam\steamapps\common\REPO`,
		`D:\SteamLibrary\steamapps\common\REPO`,
	}
	for _, p := range commonPaths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("REPO directory not found")
}
