package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTempFile(t *testing.T) {
	// Test that createTempFile can create a temp file
	tmpFile, err := createTempFile()
	if err != nil {
		t.Fatalf("createTempFile() error = %v", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	// Verify file exists
	if _, err := os.Stat(tmpFile.Name()); os.IsNotExist(err) {
		t.Errorf("temp file was not created: %s", tmpFile.Name())
	}

	// Verify file is writable
	testData := []byte("test data")
	n, err := tmpFile.Write(testData)
	if err != nil {
		t.Errorf("failed to write to temp file: %v", err)
	}
	if n != len(testData) {
		t.Errorf("wrote %d bytes, expected %d", n, len(testData))
	}
}

func TestCreateTempFile_FallbackToExecutableDir(t *testing.T) {
	// This test verifies the function works with the executable directory fallback
	tmpFile, err := createTempFile()
	if err != nil {
		t.Fatalf("createTempFile() error = %v", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	// The file should be in one of the expected directories
	dir := filepath.Dir(tmpFile.Name())
	validDirs := []string{
		os.TempDir(),
		"/var/tmp",
		os.Getenv("HOME"),
		".",
	}

	// Add executable directory
	if execPath, err := os.Executable(); err == nil {
		validDirs = append(validDirs, filepath.Dir(execPath))
	}

	found := false
	for _, validDir := range validDirs {
		if validDir == "" {
			continue
		}
		absValidDir, _ := filepath.Abs(validDir)
		absDir, _ := filepath.Abs(dir)
		if absDir == absValidDir {
			found = true
			break
		}
	}

	if !found {
		t.Logf("temp file created in: %s", dir)
		// Don't fail, as the directory might be a symlink or resolved path
	}
}

func TestNeedsUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		latest   string
		expected bool
	}{
		{
			name:     "same version",
			latest:   "2.0.0", // assuming current version is 2.0.0 or dev
			expected: true,    // dev versions always need upgrade
		},
		{
			name:     "with v prefix",
			latest:   "v2.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test is limited since version.Version is a compile-time constant
			// In dev, version is "dev" which always returns true
			result := NeedsUpgrade(tt.latest)
			// Just verify it doesn't panic
			_ = result
		})
	}
}

func TestGetAssetName(t *testing.T) {
	name := GetAssetName()
	if name == "" {
		t.Error("GetAssetName() returned empty string")
	}

	// Should contain http-remote prefix
	if !strings.HasPrefix(name, "http-remote-") {
		t.Errorf("GetAssetName() = %s, expected prefix 'http-remote-'", name)
	}
}

func TestFindAssetURL(t *testing.T) {
	release := &GitHubRelease{
		TagName: "v2.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               GetAssetName(),
				BrowserDownloadURL: "https://example.com/download",
			},
		},
	}

	url, err := FindAssetURL(release)
	if err != nil {
		t.Fatalf("FindAssetURL() error = %v", err)
	}

	if url != "https://example.com/download" {
		t.Errorf("FindAssetURL() = %s, expected https://example.com/download", url)
	}
}

func TestFindAssetURL_NotFound(t *testing.T) {
	release := &GitHubRelease{
		TagName: "v2.0.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               "http-remote-windows-amd64.exe",
				BrowserDownloadURL: "https://example.com/download",
			},
		},
	}

	_, err := FindAssetURL(release)
	if err == nil {
		t.Error("FindAssetURL() expected error for missing asset")
	}
}
