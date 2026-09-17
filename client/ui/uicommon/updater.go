package uicommon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/mod/semver"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	restcommon "github.com/ikafly144/modrepo/common/rest"
	"github.com/ikafly144/modrepo/common/versioning"
)

func FindBranchVersion(info *restcommon.VersionInfo, branch string) string {
	return versioning.FindBranchVersion(info, branch)
}

// CheckAvailableUpdateDetailed queries the server to check if an update is available for the current branch and version.
// It returns the target branch BranchInfo (nil if up to date), whether the update is mandatory, and any error.
func (s *State) CheckAvailableUpdateDetailed() (branchInfo *restcommon.BranchInfo, isMandatory bool, err error) {
	if s.Rest == nil {
		return nil, false, errors.New("cannot check for updates in offline mode")
	}

	branchName := "stable"
	if app := fyne.CurrentApp(); app != nil && app.Preferences() != nil {
		branchName = app.Preferences().StringWithFallback("core.update_branch", "stable")
	}
	branch := versioning.BranchFromString(branchName)

	info, err := s.Rest.GetVersionInfo()
	if err != nil {
		return nil, false, err
	}

	targetBranch := versioning.FindBranchInfo(info, branch.String())
	stableBranch := versioning.FindBranchInfo(info, versioning.BranchStable.String())
	if targetBranch != nil && targetBranch.Version != "" && semver.Compare(targetBranch.Version, s.Version) > 0 {
		tag := targetBranch.Version
		isMandatory = s.Version != "(devel)" && semver.Prerelease(tag) == "" && semver.Build(tag) == "" &&
			stableBranch != nil && stableBranch.Version != "" && semver.Compare(stableBranch.Version, s.Version) > 0
		return targetBranch, isMandatory, nil
	}
	return nil, false, nil
}

// CheckAvailableUpdate queries the server to check if an update is available for the current branch and version.
// It returns the target branch release tag (empty if up to date), whether the update is mandatory, and any error.
func (s *State) CheckAvailableUpdate() (tag string, isMandatory bool, err error) {
	info, mandatory, err := s.CheckAvailableUpdateDetailed()
	if err != nil {
		return "", false, err
	}
	if info != nil {
		return info.Version, mandatory, nil
	}
	return "", false, nil
}

func (s *State) CheckForUpdates(ctx context.Context, interactive bool) {
	if s.Rest == nil {
		if interactive {
			s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.error.offline", "Cannot check for updates in offline mode.")))
		}
		return
	}

	branchInfo, isMandatory, err := s.CheckAvailableUpdateDetailed()
	if err != nil {
		slog.Error("Failed to check for updates via server", "error", err)
		if interactive {
			s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.check_failed", "Failed to check for updates: {{.Error}}", map[string]any{"Error": err.Error()})))
		}
		return
	}

	if branchInfo != nil && branchInfo.Version != "" {
		slog.Info("Update available", "version", branchInfo.Version, "current", s.Version)
		s.ShowUpdateDialogDetailed(branchInfo, isMandatory)
	} else {
		slog.Info("No updates available", "current", s.Version)
		if interactive {
			s.ShowInfoDialog(
				lang.LocalizeKey("update.title", "Update"),
				lang.LocalizeKey("update.latest", "You are using the latest version ({{.Version}}).", map[string]any{"Version": s.Version}),
			)
		}
	}
}

func (s *State) ShowUpdateDialog(tag string, isMandatory bool) {
	s.ShowUpdateDialogDetailed(&restcommon.BranchInfo{
		Version: tag,
		Title:   tag,
	}, isMandatory)
}

func (s *State) ShowUpdateDialogDetailed(branchInfo *restcommon.BranchInfo, isMandatory bool) {
	if s.Window == nil || branchInfo == nil || branchInfo.Version == "" {
		return
	}

	fyne.Do(func() {
		s.dialogLock.Lock()
		if s.activeDialog != nil {
			s.dialogLock.Unlock()
			return
		}
		s.dialogLock.Unlock()

		confirmMsg := lang.LocalizeKey("update.available", "New version \"{{.Version}}\" is available. Do you want to update now?", map[string]any{"Version": branchInfo.Version})
		msgLabel := widget.NewLabel(confirmMsg)
		msgLabel.Wrapping = fyne.TextWrapWord

		var content fyne.CanvasObject
		if branchInfo.ReleaseNotes != "" {
			notesLabel := widget.NewLabelWithStyle(lang.LocalizeKey("update.release_notes", "Release Notes:"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			richText := widget.NewRichTextFromMarkdown(branchInfo.ReleaseNotes)
			richText.Wrapping = fyne.TextWrapWord
			scroll := container.NewVScroll(richText)
			scroll.SetMinSize(fyne.NewSize(450, 200))

			content = container.NewVBox(
				msgLabel,
				widget.NewSeparator(),
				notesLabel,
				scroll,
			)
		} else {
			content = msgLabel
		}

		confirmDialog := dialog.NewCustomConfirm(
			lang.LocalizeKey("update.title", "Update Available"),
			lang.LocalizeKey("update.now", "Update Now"),
			lang.LocalizeKey("update.later", "Later"),
			content,
			func(yes bool) {
				if yes {
					s.PerformUpdate(branchInfo.Version)
				} else if isMandatory {
					s.showMandatoryUpdateRequiredDialog()
				}
			},
			s.Window,
		)
		if branchInfo.ReleaseNotes != "" {
			confirmDialog.Resize(fyne.NewSize(500, 360))
		}

		var d dialog.Dialog = confirmDialog

		s.dialogLock.Lock()
		s.activeDialog = d
		s.dialogLock.Unlock()

		d.SetOnClosed(func() {
			s.dialogLock.Lock()
			if s.activeDialog == d {
				s.activeDialog = nil
			}
			s.dialogLock.Unlock()
		})

		d.Show()
	})
}

func (s *State) ResolveLatestUpdateTag(tag string) string {
	latestTag, _, err := s.CheckAvailableUpdate()
	if err != nil {
		slog.Warn("Failed to fetch latest version info before update download; using previous tag", "error", err, "tag", tag)
		return tag
	}
	if latestTag != "" && (tag == "" || semver.Compare(latestTag, tag) > 0) {
		slog.Info("Newer version available at download time, upgrading target tag", "original", tag, "latest", latestTag)
		return latestTag
	}
	return tag
}

func (s *State) PerformUpdate(tag string) {
	if s.Window == nil {
		return
	}

	statusLabel := widget.NewLabel(lang.LocalizeKey("update.downloading", "Downloading and applying update..."))
	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)
	content := container.NewVBox(
		statusLabel,
		progressBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons(
		lang.LocalizeKey("update.title", "Update"),
		content,
		s.Window,
	)
	progressDialog.Resize(fyne.NewSize(380, 100))
	progressDialog.Show()

	go func() {
		targetTag := s.ResolveLatestUpdateTag(tag)
		installerLaunched, err := versioning.UpdateWithProgress(context.Background(), targetTag, func(downloaded, total int64) {
			if total > 0 {
				ratio := float64(downloaded) / float64(total)
				if ratio > 1.0 {
					ratio = 1.0
				}
				fyne.Do(func() {
					progressBar.SetValue(ratio)
					statusLabel.SetText(fmt.Sprintf("%s (%s / %s)",
						lang.LocalizeKey("update.downloading", "Downloading and applying update..."),
						formatBytes(downloaded),
						formatBytes(total),
					))
				})
			}
		})
		if err != nil {
			slog.Error("Failed to update", "error", err)
			fyne.Do(func() {
				progressDialog.Hide()
				s.ShowErrorDialog(errors.New(lang.LocalizeKey("update.failed", "Update failed: {{.Error}}", map[string]any{"Error": err.Error()})))
			})
			return
		}

		fyne.Do(func() {
			progressDialog.Hide()
			if installerLaunched {
				slog.Info("Installer launched, exiting to allow update")
				if app := fyne.CurrentApp(); app != nil {
					app.Quit()
				}
				return
			}
			execCmd := exec.Command(os.Args[0], os.Args[1:]...)
			execCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if err := execCmd.Start(); err != nil {
				slog.Error("Failed to restart application", "error", err)
			}
			if app := fyne.CurrentApp(); app != nil {
				app.Quit()
			}
		})
	}()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (s *State) showMandatoryUpdateRequiredDialog() {
	if s.Window == nil {
		if app := fyne.CurrentApp(); app != nil {
			app.Quit()
		}
		return
	}
	errDialog := dialog.NewError(
		errors.New(lang.LocalizeKey("update.required", "Update is required to continue. Please update to the latest version and restart the application.")),
		s.Window,
	)
	errDialog.SetOnClosed(func() {
		if app := fyne.CurrentApp(); app != nil {
			app.Quit()
		}
	})
	errDialog.Show()
}

func (s *State) StartPeriodicUpdateChecker(ctx context.Context) {
	go func() {
		// Initial check after 3 seconds (skip if updater already performed initial check)
		if !s.IsInitial {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				s.CheckForUpdates(ctx, false)
			}
		}

		// Periodic check every 1 hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.Core != nil && s.Core.IsAnyProfileBusy() {
					continue
				}
				s.CheckForUpdates(ctx, false)
			}
		}
	}()
}
