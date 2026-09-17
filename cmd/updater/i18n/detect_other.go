//go:build !windows

package i18n

import (
	"os"
	"strings"
)

// DetectOSLanguage detects the OS language from environment variables on non-Windows.
func DetectOSLanguage() string {
	lang := os.Getenv("LC_ALL")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	if strings.HasPrefix(strings.ToLower(lang), "ja") {
		return "ja"
	}
	return "en"
}
