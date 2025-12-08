// Package upgrade provides self-update functionality for the application.
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/pandeptwidyaop/http-remote/internal/version"
)

const (
	githubRepo   = "pandeptwidyaop/http-remote"
	githubAPI    = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	downloadBase = "https://github.com/" + githubRepo + "/releases/download"
	maxBackups   = 3 // Maximum number of backup versions to keep
)

// GitHubRelease represents a GitHub release with its metadata.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// BackupInfo represents information about a backup version.
type BackupInfo struct {
	Path      string    `json:"path"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
}

// Progress represents progress information during upgrade.
type Progress struct {
	Stage      string `json:"stage"`       // "checking", "downloading", "installing", "complete", "error"
	Percent    int    `json:"percent"`     // 0-100
	Message    string `json:"message"`     // Human-readable message
	NewVersion string `json:"new_version"` // Version being installed
}

// CheckLatestVersion checks GitHub for the latest release version
func CheckLatestVersion() (*GitHubRelease, error) {
	resp, err := http.Get(githubAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to check for updates: HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// NeedsUpgrade compares current version with latest
func NeedsUpgrade(latest string) bool {
	current := version.Version
	// Remove 'v' prefix for comparison
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// If current version contains commit hash (dev build), always offer upgrade
	if strings.Contains(current, "-") || current == "dev" {
		return true
	}

	return current != latest
}

// GetAssetName returns the expected asset name for current platform
func GetAssetName() string {
	return fmt.Sprintf("http-remote-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// FindAssetURL finds the download URL for current platform
func FindAssetURL(release *GitHubRelease) (string, error) {
	expectedName := GetAssetName()

	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

// createTempFile tries to create a temp file in multiple locations
// This handles cases where /tmp is read-only (e.g., systemd PrivateTmp, containers)
func createTempFile() (*os.File, error) {
	// Try multiple temp directories in order of preference
	tempDirs := []string{
		os.TempDir(),      // System temp (usually /tmp)
		"/var/tmp",        // Persistent temp (survives reboots)
		os.Getenv("HOME"), // User home directory
		".",               // Current working directory
	}

	// Also try directory of current executable
	if execPath, err := os.Executable(); err == nil {
		tempDirs = append([]string{filepath.Dir(execPath)}, tempDirs...)
	}

	var lastErr error
	for _, dir := range tempDirs {
		if dir == "" {
			continue
		}

		// Check if directory exists and is writable
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}

		tmpFile, err := os.CreateTemp(dir, "http-remote-upgrade-*")
		if err != nil {
			lastErr = err
			continue
		}

		return tmpFile, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all temp directories failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no writable temp directory found")
}

// Download downloads the new binary to a temporary file
func Download(url string, progressFn func(downloaded, total int64)) (string, error) {
	// #nosec G107 - URL is from GitHub API and validated by caller
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Try to create temp file in different locations
	// Some systems have read-only /tmp (e.g., Docker containers, restricted servers)
	tmpFile, err := createTempFile()
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()

	// Download with progress
	var downloaded int64
	total := resp.ContentLength
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := tmpFile.Write(buf[:n])
			if writeErr != nil {
				_ = os.Remove(tmpFile.Name())
				return "", fmt.Errorf("failed to write temp file: %w", writeErr)
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.Remove(tmpFile.Name())
			return "", fmt.Errorf("failed to download: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

// Install replaces the current binary with the new one
func Install(tmpPath string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Create versioned backup with timestamp
	backupPath := fmt.Sprintf("%s.backup.%s.%s", execPath, version.Version, time.Now().Format("20060102150405"))
	if err := copyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Rotate old backups (keep only maxBackups)
	if err := rotateBackups(execPath); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to rotate backups: %v\n", err)
	}

	// Remove current binary
	if err := os.Remove(execPath); err != nil {
		return fmt.Errorf("failed to remove current binary: %w", err)
	}

	// Move new binary to executable path
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore from backup
		_ = copyFile(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Make executable
	// #nosec G302 - executable needs to be executable by owner and group
	if err := os.Chmod(execPath, 0755); err != nil {
		// Try to restore from backup
		_ = copyFile(backupPath, execPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
// #nosec G304 - src/dst are internal paths from backup/rollback operations
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// rotateBackups removes old backups, keeping only the most recent maxBackups
func rotateBackups(_ string) error {
	backups, err := ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= maxBackups {
		return nil
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	// Remove older backups
	for i := maxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", backups[i].Path, err)
		}
	}

	return nil
}

// ListBackups returns all available backup versions
func ListBackups() ([]BackupInfo, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	dir := filepath.Dir(execPath)
	base := filepath.Base(execPath)

	// Pattern: http-remote.backup.v1.2.3.20241207123456
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.backup\.(v[0-9]+\.[0-9]+\.[0-9]+)\.([0-9]{14})$`)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := pattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			// Also check for simple .backup extension (legacy)
			if entry.Name() == base+".backup" {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				backups = append(backups, BackupInfo{
					Path:      filepath.Join(dir, entry.Name()),
					Version:   "unknown",
					Timestamp: info.ModTime(),
					Size:      info.Size(),
				})
			}
			continue
		}

		ver := matches[1]
		tsStr := matches[2]

		ts, err := time.Parse("20060102150405", tsStr)
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      filepath.Join(dir, entry.Name()),
			Version:   ver,
			Timestamp: ts,
			Size:      info.Size(),
		})
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// Rollback restores a previous version from backup
func Rollback(backupPath string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Create backup of current version before rollback
	currentBackupPath := fmt.Sprintf("%s.backup.%s.%s", execPath, version.Version, time.Now().Format("20060102150405"))
	if err := copyFile(execPath, currentBackupPath); err != nil {
		return fmt.Errorf("failed to backup current binary before rollback: %w", err)
	}

	// Remove current binary
	if err := os.Remove(execPath); err != nil {
		return fmt.Errorf("failed to remove current binary: %w", err)
	}

	// Copy backup to executable path
	if err := copyFile(backupPath, execPath); err != nil {
		// Try to restore from our new backup
		_ = copyFile(currentBackupPath, execPath)
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Make executable
	// #nosec G302 - executable needs to be executable by owner and group
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Rotate backups
	_ = rotateBackups(execPath)

	return nil
}

// Run performs the full upgrade process
func Run(force bool) error {
	fmt.Printf("Current version: %s\n", version.Version)
	fmt.Println("Checking for updates...")

	release, err := CheckLatestVersion()
	if err != nil {
		return err
	}

	fmt.Printf("Latest version: %s\n", release.TagName)

	if !force && !NeedsUpgrade(release.TagName) {
		fmt.Println("You are already running the latest version.")
		return nil
	}

	assetURL, err := FindAssetURL(release)
	if err != nil {
		return err
	}

	fmt.Printf("Downloading %s...\n", GetAssetName())

	tmpPath, err := Download(assetURL, func(downloaded, total int64) {
		if total > 0 {
			pct := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rDownloading: %.1f%% (%d/%d bytes)", pct, downloaded, total)
		} else {
			fmt.Printf("\rDownloading: %d bytes", downloaded)
		}
	})
	if err != nil {
		return err
	}
	fmt.Println()

	fmt.Println("Installing...")
	if err := Install(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	fmt.Printf("Successfully upgraded to %s!\n", release.TagName)
	fmt.Println("Please restart the application to use the new version.")

	return nil
}

// RunWithProgress performs the upgrade with progress callbacks for API use
func RunWithProgress(force bool, progressFn func(Progress)) (*GitHubRelease, error) {
	progressFn(Progress{
		Stage:   "checking",
		Percent: 0,
		Message: "Checking for updates...",
	})

	release, err := CheckLatestVersion()
	if err != nil {
		progressFn(Progress{
			Stage:   "error",
			Percent: 0,
			Message: err.Error(),
		})
		return nil, err
	}

	progressFn(Progress{
		Stage:      "checking",
		Percent:    10,
		Message:    fmt.Sprintf("Latest version: %s", release.TagName),
		NewVersion: release.TagName,
	})

	if !force && !NeedsUpgrade(release.TagName) {
		progressFn(Progress{
			Stage:      "complete",
			Percent:    100,
			Message:    "Already running the latest version",
			NewVersion: release.TagName,
		})
		return release, nil
	}

	assetURL, err := FindAssetURL(release)
	if err != nil {
		progressFn(Progress{
			Stage:   "error",
			Percent: 10,
			Message: err.Error(),
		})
		return nil, err
	}

	progressFn(Progress{
		Stage:      "downloading",
		Percent:    15,
		Message:    "Starting download...",
		NewVersion: release.TagName,
	})

	tmpPath, err := Download(assetURL, func(downloaded, total int64) {
		if total > 0 {
			// Map download progress from 15% to 85%
			pct := 15 + int(float64(downloaded)/float64(total)*70)
			progressFn(Progress{
				Stage:      "downloading",
				Percent:    pct,
				Message:    fmt.Sprintf("Downloading: %d%%", int(float64(downloaded)/float64(total)*100)),
				NewVersion: release.TagName,
			})
		}
	})
	if err != nil {
		progressFn(Progress{
			Stage:   "error",
			Percent: 50,
			Message: err.Error(),
		})
		return nil, err
	}

	progressFn(Progress{
		Stage:      "installing",
		Percent:    90,
		Message:    "Installing new version...",
		NewVersion: release.TagName,
	})

	if err := Install(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		progressFn(Progress{
			Stage:   "error",
			Percent: 90,
			Message: err.Error(),
		})
		return nil, err
	}

	progressFn(Progress{
		Stage:      "complete",
		Percent:    100,
		Message:    fmt.Sprintf("Successfully upgraded to %s! Restart required.", release.TagName),
		NewVersion: release.TagName,
	})

	return release, nil
}
