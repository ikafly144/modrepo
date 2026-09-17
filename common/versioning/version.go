package versioning

// repository information
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v91/github"
	"golang.org/x/mod/semver"

	restcommon "github.com/ikafly144/modrepo/common/rest"
)

var (
	repoOwner    = "ikafly144"
	repoName     = "modrepo"
	artifactName = "modrepo_${OS}_${ARCH}.msi"

	versionCacheMu    sync.Mutex
	cachedVersionInfo *restcommon.VersionInfo
	cacheExpiry       time.Time
)

const cacheTTL = 2 * time.Minute

// FetchVersionInfo fetches release information from GitHub Releases, utilizing an in-memory cache
// and falling back to redirect lookup if GitHub API rate limits or errors occur.
func FetchVersionInfo(ctx context.Context) (*restcommon.VersionInfo, error) {
	versionCacheMu.Lock()
	if cachedVersionInfo != nil && time.Now().Before(cacheExpiry) {
		info := cachedVersionInfo
		versionCacheMu.Unlock()
		return info, nil
	}
	versionCacheMu.Unlock()

	info, err := fetchVersionInfoFromGitHub(ctx)
	if err != nil {
		slog.Warn("Failed to fetch releases via GitHub API, attempting redirect fallback", "error", err)
		fallbackInfo, fallbackErr := fetchLatestViaRedirect(ctx)
		if fallbackErr == nil && fallbackInfo != nil {
			versionCacheMu.Lock()
			cachedVersionInfo = fallbackInfo
			cacheExpiry = time.Now().Add(cacheTTL)
			versionCacheMu.Unlock()
			return fallbackInfo, nil
		}
		versionCacheMu.Lock()
		if cachedVersionInfo != nil {
			cached := cachedVersionInfo
			versionCacheMu.Unlock()
			slog.Warn("Using expired cached version info due to network error", "error", err)
			return cached, nil
		}
		versionCacheMu.Unlock()
		return nil, err
	}

	versionCacheMu.Lock()
	cachedVersionInfo = info
	cacheExpiry = time.Now().Add(cacheTTL)
	versionCacheMu.Unlock()

	return info, nil
}

func fetchLatestViaRedirect(ctx context.Context) (*restcommon.VersionInfo, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	url := fmt.Sprintf("https://github.com/%s/%s/releases/latest", repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently && resp.StatusCode != http.StatusSeeOther {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		return nil, errors.New("missing Location header in redirect")
	}
	tag := filepath.Base(loc)
	if tag == "" || tag == "latest" {
		return nil, fmt.Errorf("could not extract tag from location: %s", loc)
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	return &restcommon.VersionInfo{
		Branches: []restcommon.BranchInfo{
			{
				Name:    BranchStable.String(),
				Version: tag,
				Title:   tag,
			},
		},
	}, nil
}

func fetchVersionInfoFromGitHub(ctx context.Context) (*restcommon.VersionInfo, error) {
	var opts []github.ClientOptionsFunc
	ghClient, err := github.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	releases, _, err := ghClient.Repositories.ListReleases(ctx, repoOwner, repoName, &github.ListOptions{PerPage: 30})
	if err != nil {
		return nil, err
	}

	branchMap := make(map[Branch]restcommon.BranchInfo)

	for _, rel := range releases {
		if rel == nil || rel.GetDraft() {
			continue
		}
		tag := rel.GetTagName()
		if !semver.IsValid(tag) {
			continue
		}
		prerelease := strings.TrimPrefix(semver.Prerelease(tag), "-")
		before, _, _ := strings.Cut(prerelease, ".")

		title := rel.GetName()
		if title == "" {
			title = tag
		}
		body := rel.GetBody()

		for b := BranchStable; b <= BranchDev; b++ {
			if !b.match(before) {
				continue
			}
			cur, exists := branchMap[b]
			if !exists || semver.Compare(tag, cur.Version) > 0 {
				branchMap[b] = restcommon.BranchInfo{
					Name:         b.String(),
					Version:      tag,
					Title:        title,
					ReleaseNotes: body,
				}
			}
		}
	}

	branches := make([]restcommon.BranchInfo, 0, len(branchMap))
	for b := BranchStable; b <= BranchDev; b++ {
		if info, ok := branchMap[b]; ok {
			branches = append(branches, info)
		}
	}

	return &restcommon.VersionInfo{Branches: branches}, nil
}

// CheckForUpdatesDetailed checks if an update is available for the given branch and returns the detailed branch info.
func CheckForUpdatesDetailed(ctx context.Context, branch Branch, currentVersion string) (targetBranchInfo *restcommon.BranchInfo, latestStable *restcommon.BranchInfo, err error) {
	info, err := FetchVersionInfo(ctx)
	if err != nil {
		return nil, nil, err
	}

	target := FindBranchInfo(info, branch.String())
	stable := FindBranchInfo(info, BranchStable.String())

	var retTarget *restcommon.BranchInfo
	if target != nil && (currentVersion == "" || semver.Compare(target.Version, currentVersion) > 0) {
		retTarget = target
	}

	var retStable *restcommon.BranchInfo
	if stable != nil && (currentVersion == "" || semver.Compare(stable.Version, currentVersion) > 0) {
		retStable = stable
	}

	return retTarget, retStable, nil
}

func CheckForUpdates(ctx context.Context, branch Branch, currentVersion string) (releaseTag string, latestStable string, err error) {
	target, stable, err := CheckForUpdatesDetailed(ctx, branch, currentVersion)
	if err != nil {
		return "", "", err
	}
	var tag, stab string
	if target != nil {
		tag = target.Version
	}
	if stable != nil {
		stab = stable.Version
	}
	return tag, stab, nil
}

// ProgressCallback is called during download with bytes downloaded and total size in bytes.
type ProgressCallback func(downloaded int64, total int64)

type progressWriter struct {
	total      int64
	downloaded int64
	onProgress ProgressCallback
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	if pw.onProgress != nil {
		pw.onProgress(pw.downloaded, pw.total)
	}
	return n, nil
}

func DownloadUpdate(ctx context.Context, tag string) (string, error) {
	return DownloadUpdateWithProgress(ctx, tag, nil)
}

func DownloadUpdateWithProgress(ctx context.Context, tag string, onProgress ProgressCallback) (string, error) {
	assetName := replaceOSAndArch(artifactName)

	var downloadURL string
	var checksumsURL string

	// 1. Try to get asset URLs from GitHub Releases API
	var opts []github.ClientOptionsFunc
	ghClient, err := github.NewClient(opts...)
	if err == nil {
		release, _, relErr := ghClient.Repositories.GetReleaseByTag(ctx, repoOwner, repoName, tag)
		if relErr == nil && release != nil {
			for _, asset := range release.Assets {
				if asset.GetName() == assetName {
					downloadURL = asset.GetBrowserDownloadURL()
				} else if asset.GetName() == "checksums.txt" {
					checksumsURL = asset.GetBrowserDownloadURL()
				}
			}
		}
	}

	// 2. Fallback to direct release download URLs if API didn't resolve them
	if downloadURL == "" {
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", repoOwner, repoName, tag, assetName)
	}
	if checksumsURL == "" {
		checksumsURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt", repoOwner, repoName, tag)
	}

	// 3. Download checksums.txt
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for checksums.txt: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksums.txt: status code %d", resp.StatusCode)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", fmt.Errorf("failed to read checksums.txt: %w", err)
	}

	var checkSum []byte
	lines := strings.SplitSeq(buf.String(), "\n")
	for line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			checkSum, err = hex.DecodeString(parts[0])
			if err != nil {
				return "", fmt.Errorf("invalid checksum hex: %w", err)
			}
			break
		}
	}

	if len(checkSum) == 0 {
		return "", errors.New("checksum for MSI not found in checksums.txt")
	}

	// 4. Download MSI
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download MSI: status code %d", resp.StatusCode)
	}

	hasher := sha256.New()
	tempFile, err := os.CreateTemp("", "modrepo-*.msi")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	total := resp.ContentLength
	if onProgress != nil {
		onProgress(0, total)
	}
	pw := &progressWriter{
		total:      total,
		onProgress: onProgress,
	}

	reader := io.TeeReader(resp.Body, io.MultiWriter(hasher, pw))
	if _, err := io.Copy(tempFile, reader); err != nil {
		return "", err
	}

	if !bytes.Equal(checkSum, hasher.Sum(nil)) {
		return "", errors.New("checksum verification failed for downloaded MSI")
	}

	if err := tempFile.Close(); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

func UpdateWithProgress(ctx context.Context, tag string, onProgress ProgressCallback) (bool, error) {
	msiPath, err := DownloadUpdateWithProgress(ctx, tag, onProgress)
	if err != nil {
		return false, err
	}

	cmd := exec.Command("msiexec", "/i", msiPath, "/norestart")
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("failed to start installer: %w", err)
	}
	slog.Info("Started MSI installer", "path", msiPath)
	return true, nil
}

func Update(ctx context.Context, tag string) (bool, error) {
	return UpdateWithProgress(ctx, tag, nil)
}

func RunMsiPassive(ctx context.Context, msiPath string) error {
	cmd := exec.CommandContext(ctx, "msiexec", "/i", msiPath, "/passive", "/norestart")
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		code := exitErr.ExitCode()
		// 0: SUCCESS, 3010: ERROR_SUCCESS_REBOOT_REQUIRED, 1641: ERROR_SUCCESS_REBOOT_INITIATED
		if code == 0 || code == 3010 || code == 1641 {
			return nil
		}
		return fmt.Errorf("msiexec exited with code %d", code)
	}
	return err
}

func replaceOSAndArch(name string) string {
	// Replace OS
	var osStr string
	switch runtime.GOOS {
	case "windows":
		osStr = "windows"
	case "linux":
		osStr = "linux"
	case "darwin":
		osStr = "darwin"
	default:
		osStr = runtime.GOOS
	}
	name = strings.ReplaceAll(name, "${OS}", osStr)

	// Replace Arch
	var archStr string
	switch runtime.GOARCH {
	case "amd64":
		archStr = "x86_64"
	case "386":
		archStr = "i386"
	default:
		archStr = runtime.GOARCH
	}
	name = strings.ReplaceAll(name, "${ARCH}", archStr)

	if osStr == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}

	return name
}

func BranchFromString(s string) Branch {
	for b, str := range branchString {
		if str == s {
			return b
		}
	}
	return BranchStable
}

type Branch int

const (
	BranchStable Branch = iota
	BranchPreview
	BranchBeta
	BranchCanary
	BranchDev
)

var branchString = map[Branch]string{
	BranchStable:  "stable",
	BranchPreview: "preview",
	BranchBeta:    "beta",
	BranchCanary:  "canary",
	BranchDev:     "dev",
}

var prereleaseToBranch = map[string]Branch{
	"rc":    BranchPreview,
	"pre":   BranchBeta,
	"beta":  BranchCanary,
	"alpha": BranchDev,
}

func (b Branch) match(prerelease string) bool {
	if p, ok := prereleaseToBranch[prerelease]; (ok && p <= b) || prerelease == "" {
		return true
	}
	return false
}

func (b Branch) String() string {
	return branchString[b]
}
