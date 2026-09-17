package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLanguage(t *testing.T) {
	assert.Equal(t, "ja", NormalizeLanguage("ja"))
	assert.Equal(t, "ja", NormalizeLanguage("ja-JP"))
	assert.Equal(t, "ja", NormalizeLanguage("ja_JP"))
	assert.Equal(t, "ja", NormalizeLanguage("JA-JP"))
	assert.Equal(t, "en", NormalizeLanguage("en-US"))
	assert.Equal(t, "en", NormalizeLanguage("en"))
	assert.Equal(t, "", NormalizeLanguage(""))
	assert.Equal(t, "", NormalizeLanguage("   "))
}

func TestResolveLanguage(t *testing.T) {
	assert.Equal(t, "ja", ResolveLanguage("ja", "en"))
	assert.Equal(t, "ja", ResolveLanguage("ja-JP", ""))
	assert.Equal(t, "en", ResolveLanguage("", "en"))
	assert.Equal(t, "ja", ResolveLanguage("", "ja"))
}

func TestTranslation(t *testing.T) {
	SetLanguage("en")
	assert.Equal(t, "MODREPO Updater", T("updater.title"))
	assert.Equal(t, "Updating MODREPO to the latest version...", T("updater.status.updating"))
	assert.Equal(t, "Downloading version v1.2.0...", T("updater.status.downloading_start", map[string]any{
		"Version": "v1.2.0",
	}))
	assert.Equal(t, "Downloading version v1.2.0... (10 MB / 50 MB)", T("updater.status.downloading_progress", map[string]any{
		"Version":    "v1.2.0",
		"Downloaded": "10 MB",
		"Total":      "50 MB",
	}))

	SetLanguage("ja")
	assert.Equal(t, "MODREPO アップデーター", T("updater.title"))
	assert.Equal(t, "MODREPO を最新バージョンに更新しています...", T("updater.status.updating"))
	assert.Equal(t, "バージョン v1.2.0 をダウンロードしています...", T("updater.status.downloading_start", map[string]any{
		"Version": "v1.2.0",
	}))
	assert.Equal(t, "バージョン v1.2.0 をダウンロード中... (10 MB / 50 MB)", T("updater.status.downloading_progress", map[string]any{
		"Version":    "v1.2.0",
		"Downloaded": "10 MB",
		"Total":      "50 MB",
	}))

	// Fallback to en for unknown key in unsupported lang
	SetLanguage("fr")
	assert.Equal(t, "MODREPO Updater", T("updater.title"))

	// Non-existent key returns key name
	assert.Equal(t, "non.existent.key", T("non.existent.key"))
}
