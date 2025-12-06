// Package upgrade provides self-update functionality for the application.
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pandeptwidyaop/http-remote/internal/version"
)

const (
	githubRepo   = "pandeptwidyaop/http-remote"
	githubAPI    = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	downloadBase = "https://github.com/" + githubRepo + "/releases/download"
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

// Download downloads the new binary to a temporary file
func Download(url string, progressFn func(downloaded, total int64)) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "http-remote-upgrade-*")
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

	// Backup current binary
	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary to executable path
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(execPath, 0755); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Remove backup
	_ = os.Remove(backupPath)

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
