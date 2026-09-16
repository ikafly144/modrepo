//go:build windows

package assetstools

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func makeSerializedString(s string) []byte {
	b := make([]byte, 4+len(s))
	binary.LittleEndian.PutUint32(b[:4], uint32(len(s)))
	copy(b[4:], s)
	// Align to 4 bytes
	if rem := len(b) % 4; rem != 0 {
		b = append(b, make([]byte, 4-rem)...)
	}
	return b
}

func TestExtractBundleVersionFromSerializedData(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "Single simple version 0.1",
			input:    makeSerializedString("0.1"),
			expected: "0.1",
			wantErr:  false,
		},
		{
			name:     "Semantic version 1.0.0",
			input:    makeSerializedString("1.0.0"),
			expected: "1.0.0",
			wantErr:  false,
		},
		{
			name:     "Among Us style year version 2024.6.18",
			input:    makeSerializedString("2024.6.18"),
			expected: "2024.6.18",
			wantErr:  false,
		},
		{
			name: "Unity 2022.3 cluster: visionOS, tvOS, and game bundleVersion",
			input: append(
				append(
					makeSerializedString("1.0"), // visionOSBundleVersion
					makeSerializedString("1.0")..., // tvOSBundleVersion
				),
				makeSerializedString("0.1")..., // bundleVersion
			),
			expected: "0.1",
			wantErr:  false,
		},
		{
			name: "With ignored non-version strings",
			input: append(
				append(
					makeSerializedString("semiwork"),
					makeSerializedString("REPO")...,
				),
				makeSerializedString("0.1.5")...,
			),
			expected: "0.1.5",
			wantErr:  false,
		},
		{
			name:    "No valid version",
			input:   makeSerializedString("Hello World"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBundleVersionFromSerializedData(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractBundleVersionFromSerializedData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("extractBundleVersionFromSerializedData() = %v, want %v", got, tt.expected)
			}
		})
	}
}


func TestReadPlayerSettingsBundleVersion(t *testing.T) {
	gamePath, err := getRepoDirForTest()
	if err != nil {
		t.Skipf("skipping: failed to locate R.E.P.O. directory: %v", err)
	}
	path := filepath.Join(gamePath, "REPO_Data", "globalgamemanagers")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("skipping: globalgamemanagers not found: %v", err)
	}

	version, err := ReadPlayerSettingsBundleVersion(path)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}
	if version != "0.1" {
		t.Errorf("expected version 0.1, got %q", version)
	}
	t.Logf("Read R.E.P.O. version: %s", version)
}
