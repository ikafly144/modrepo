package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	_ "unsafe"

	"github.com/josephspurrier/goversioninfo"
	_ "golang.org/x/mod/semver"
)

//nolint:unused
type parsed struct {
	major      string
	minor      string
	patch      string
	short      string
	prerelease string
	build      string
}

//go:linkname parse golang.org/x/mod/semver.parse
func parse(v string) (parsed, bool)

func versionStrToNum(versionString string) ([]int, error) {
	vn := make([]int, 4)
	version, ok := parse(versionString)
	if !ok {
		return nil, fmt.Errorf("%s: invalid semantic version", versionString)
	}
	var err error
	if vn[0], err = strconv.Atoi(version.major); err != nil {
		return nil, fmt.Errorf("%s: invalid major version", versionString)
	}
	if vn[1], err = strconv.Atoi(version.minor); err != nil {
		return nil, fmt.Errorf("%s: invalid minor version", versionString)
	}
	if vn[2], err = strconv.Atoi(version.patch); err != nil {
		return nil, fmt.Errorf("%s: invalid patch version", versionString)
	}
	// Build number is unused in semver, so we set it to 0
	vn[3] = 0
	if vn[0] == 0 && vn[1] == 0 && vn[2] == 0 {
		vn[3] = 1
	}
	return vn[:], nil
}

func getVersionData() (string, []int, error) {
	if envVer := os.Getenv("VERSION"); envVer != "" {
		if !strings.HasPrefix(envVer, "v") {
			envVer = "v" + envVer
		}
		num, err := versionStrToNum(envVer)
		if err == nil {
			return envVer, num, nil
		}
	}

	bin, err := exec.Command("git", "describe", "--tags").Output()
	if err == nil {
		str := strings.TrimSpace(string(bin))
		num, err := versionStrToNum(str)
		if err == nil {
			return str, num, nil
		}
	}

	// Fallback when no tags exist in the git repository (e.g. fresh repo, dev branch)
	str := "v0.0.1-dev"
	num, _ := versionStrToNum(str)
	return str, num, nil
}

var (
	iconFlag     = flag.String("icon", "icon.ico", "Path to the icon file")
	archFlag     = flag.String("arch", "64", "Architecture (32 or 64)")
	manifestFlag = flag.String("manifest", "app.exe.manifest", "Path to the manifest file")
	outputFlag   = flag.String("o", "modrepo.syso", "Output .syso file path")
)

func main() {
	flag.Parse()
	fileVerStr, fileVerNum, err := getVersionData()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if iconFlag == nil || *iconFlag == "" {
		fmt.Fprintln(os.Stderr, "Icon path is required")
		os.Exit(1)
	}

	if archFlag == nil {
		fmt.Fprintln(os.Stderr, "Architecture must be either 32 or 64")
		os.Exit(1)
	}
	vi := &goversioninfo.VersionInfo{
		IconPath:     *iconFlag,
		ManifestPath: *manifestFlag,
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion:    goversioninfo.FileVersion{Major: fileVerNum[0], Minor: fileVerNum[1], Patch: fileVerNum[2], Build: fileVerNum[3]},
			ProductVersion: goversioninfo.FileVersion{Major: fileVerNum[0], Minor: fileVerNum[1], Patch: fileVerNum[2], Build: fileVerNum[3]},
			FileFlagsMask:  "3f",
			FileFlags:      "10",
			FileOS:         "040004",
			FileType:       "01",
			FileSubType:    "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			CompanyName:      "ikafly144",
			FileDescription:  "MODREPO - R.E.P.O. Mod Manager",
			FileVersion:      strings.TrimPrefix(fileVerStr, "v"),
			LegalCopyright:   "Copyright (C) 2026 ikafly144.",
			OriginalFilename: "MODREPO.EXE",
			ProductName:      "MODREPO",
			ProductVersion:   strings.TrimPrefix(fileVerStr, "v"),
		},
		LangID:    goversioninfo.LngJapanese,
		CharsetID: goversioninfo.CsMultilingual,
	}

	vi.Build()
	vi.Walk()

	if err := vi.WriteSyso(*outputFlag, *archFlag); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
