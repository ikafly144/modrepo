//go:build windows

package i18n

import "golang.org/x/sys/windows"

var (
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
)

// DetectOSLanguage detects the Windows user UI language.
func DetectOSLanguage() string {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	langID := uint16(r)
	primaryLangID := langID & 0x03FF

	switch primaryLangID {
	case 0x11: // LANG_JAPANESE
		return "ja"
	default:
		return "en"
	}
}
