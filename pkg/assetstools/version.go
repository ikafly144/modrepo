package assetstools

import (
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func ReadPlayerSettingsBundleVersion(globalGameManagersPath string) (string, error) {
	f, err := os.Open(globalGameManagersPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := NewAssetsFileReader(f)
	assetsFile, err := ReadAssetsFile(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read assets file: %w", err)
	}

	playerSettings, err := findSingleAssetByClassID(assetsFile, int32(ClassIDPlayerSettings))
	if err != nil {
		return "", err
	}

	if assetsFile.Metadata.TypeTreeEnabled {
		version, err := extractBundleVersionWithTypeTree(reader, assetsFile, playerSettings)
		if err == nil {
			return version, nil
		}
	}

	data, err := readAssetData(reader, assetsFile, playerSettings)
	if err != nil {
		return "", err
	}
	version, err := extractBundleVersionFromSerializedData(data)
	if err != nil {
		return "", fmt.Errorf("failed to extract bundleVersion from PlayerSettings: %w", err)
	}
	return version, nil
}

func findSingleAssetByClassID(file *AssetsFile, classID int32) (*AssetFileInfo, error) {
	var found *AssetFileInfo
	for i := range file.Metadata.AssetInfos {
		info := &file.Metadata.AssetInfos[i]
		if info.TypeID != classID {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple assets found for class id %d", classID)
		}
		found = info
	}
	if found == nil {
		return nil, fmt.Errorf("asset with class id %d not found", classID)
	}
	return found, nil
}

func extractBundleVersionWithTypeTree(reader *AssetsFileReader, file *AssetsFile, info *AssetFileInfo) (string, error) {
	typeTreeType := file.Metadata.FindTypeTreeTypeByID(int32(ClassIDPlayerSettings), -1)
	if typeTreeType == nil {
		return "", fmt.Errorf("type tree for PlayerSettings not found")
	}
	template, err := NewTemplateFieldFromTypeTree(typeTreeType)
	if err != nil {
		return "", err
	}

	value, err := template.MakeValueAt(reader, info.GetAbsoluteByteOffset(file))
	if err != nil {
		return "", fmt.Errorf("failed to decode PlayerSettings value: %w", err)
	}

	bundleVersionField := value.Find("bundleVersion")
	if bundleVersionField == nil {
		return "", fmt.Errorf("bundleVersion field not found")
	}
	bundleVersion, ok := bundleVersionField.AsString()
	if !ok {
		return "", fmt.Errorf("bundleVersion is not a string")
	}
	if bundleVersion == "" {
		return "", fmt.Errorf("bundleVersion is empty")
	}
	return bundleVersion, nil
}

func readAssetData(reader *AssetsFileReader, file *AssetsFile, info *AssetFileInfo) ([]byte, error) {
	if uint64(info.ByteSize) > uint64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("asset too large: %d", info.ByteSize)
	}
	if err := reader.SeekAt(info.GetAbsoluteByteOffset(file)); err != nil {
		return nil, err
	}
	return reader.ReadBytes(int(info.ByteSize))
}

var bundleVersionPattern = regexp.MustCompile(`^v?\d+(\.\d+)+([a-zA-Z0-9_\-\.]*)?$`)

var nonVersionStrings = map[string]bool{
	"public.app-category.games": true,
	"semiwork":                  true,
	"REPO":                      true,
	"Very Low":                  true,
	"Low":                       true,
	"Medium":                    true,
	"High":                      true,
	"Very High":                 true,
	"Ultra":                     true,
}

type versionCandidate struct {
	offset int
	value  string
}

func extractBundleVersionFromSerializedData(data []byte) (string, error) {
	var candidates []versionCandidate

	// 1. Length-prefixed string extraction (standard Unity string serialization)
	for i := 0; i+4 <= len(data); i += 4 {
		n := int(int32(binary.LittleEndian.Uint32(data[i : i+4])))
		if n < 1 || n > 64 || i+4+n > len(data) {
			continue
		}
		value := string(data[i+4 : i+4+n])
		if !isPrintableASCII(value) || nonVersionStrings[value] || !bundleVersionPattern.MatchString(value) {
			continue
		}
		candidates = append(candidates, versionCandidate{offset: i, value: value})
	}

	// 2. Fallback: scan for contiguous ASCII strings matching versionPattern
	if len(candidates) == 0 {
		seen := map[string]struct{}{}
		for i := 0; i < len(data); {
			if data[i] < 32 || data[i] > 126 {
				i++
				continue
			}
			start := i
			for i < len(data) && data[i] >= 32 && data[i] <= 126 {
				i++
			}
			value := string(data[start:i])
			if !nonVersionStrings[value] && bundleVersionPattern.MatchString(value) {
				if _, ok := seen[value]; !ok {
					seen[value] = struct{}{}
					candidates = append(candidates, versionCandidate{offset: start, value: value})
				}
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("version-like string was not found")
	}

	// In Unity PlayerSettings (especially 2022.3+), multiple version strings may appear in sequence
	// (e.g. visionOSBundleVersion, tvOSBundleVersion, bundleVersion).
	// bundleVersion is placed immediately before preloadedAssets, which makes it the last version string
	// in that cluster. Therefore, the last candidate represents the primary bundleVersion.
	return candidates[len(candidates)-1].value, nil
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 32 || s[i] > 126 {
			return false
		}
	}
	return len(s) > 0
}

func pickLatestVersion(candidates []string) (string, error) {
	best := ""
	var bestCore []int
	for _, candidate := range candidates {
		core, ok := parseVersionCore(candidate)
		if !ok {
			continue
		}
		if best == "" || compareVersionCore(core, bestCore) > 0 {
			best = candidate
			bestCore = core
		}
	}
	if best == "" {
		return "", fmt.Errorf("no valid version candidate found")
	}
	return best, nil
}

func parseVersionCore(version string) ([]int, bool) {
	version = strings.TrimPrefix(version, "v")
	// Strip prerelease metadata (e.g. -beta.1)
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return nil, false
	}
	var nums []int
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		nums = append(nums, n)
	}
	return nums, true
}

func compareVersionCore(a, b []int) int {
	maxLen := max(len(a), len(b))
	for i := 0; i < maxLen; i++ {
		valA := 0
		if i < len(a) {
			valA = a[i]
		}
		valB := 0
		if i < len(b) {
			valB = b[i]
		}
		if valA > valB {
			return 1
		}
		if valA < valB {
			return -1
		}
	}
	return 0
}
