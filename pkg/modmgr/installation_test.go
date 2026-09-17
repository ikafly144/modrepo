package modmgr

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractThunderstoreZip_BepInExPackSkipsMetadataFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_bepinex_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	profileRoot, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer profileRoot.Close()

	// Create mock BepInExPack zip containing metadata files and runtime files
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"icon.png":                 "fake-icon-data",
		"manifest.json":            `{"name":"BepInExPack"}`,
		"README.md":                "# BepInEx",
		"CHANGELOG.md":             "## 5.4.2100",
		"winhttp.dll":              "fake-winhttp",
		"doorstop_config.ini":      "fake-doorstop",
		"BepInEx/core/BepInEx.dll": "fake-bepinex-dll",
	}

	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	reader := bytes.NewReader(buf.Bytes())
	extracted, err := extractThunderstoreZip(reader, reader.Size(), "BepInEx-BepInExPack", profileRoot)
	require.NoError(t, err)

	// Verify that icon.png, manifest.json, README.md, CHANGELOG.md are NOT extracted
	for _, ext := range extracted {
		assert.NotEqual(t, "icon.png", ext)
		assert.NotEqual(t, "manifest.json", ext)
		assert.NotEqual(t, "README.md", ext)
		assert.NotEqual(t, "CHANGELOG.md", ext)
	}

	_, err = os.Stat(filepath.Join(tempDir, "icon.png"))
	assert.True(t, os.IsNotExist(err), "icon.png should not exist in profile directory")

	_, err = os.Stat(filepath.Join(tempDir, "manifest.json"))
	assert.True(t, os.IsNotExist(err), "manifest.json should not exist in profile directory")

	// Verify runtime files ARE extracted
	_, err = os.Stat(filepath.Join(tempDir, "winhttp.dll"))
	assert.NoError(t, err, "winhttp.dll should exist in profile directory")

	_, err = os.Stat(filepath.Join(tempDir, "BepInEx", "core", "BepInEx.dll"))
	assert.NoError(t, err, "BepInEx.dll should exist in profile directory")
}
