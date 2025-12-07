// Package service provides systemd service management functionality.
package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

const (
	serviceName     = "http-remote"
	serviceFilePath = "/etc/systemd/system/http-remote.service"
)

// ServiceStatus represents the status of the systemd service.
type ServiceStatus struct {
	IsRunning   bool   `json:"is_running"`
	IsEnabled   bool   `json:"is_enabled"`
	IsInstalled bool   `json:"is_installed"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
}

// ServiceConfig holds configuration for service installation.
type ServiceConfig struct {
	ExecPath   string
	ConfigPath string
	User       string
	WorkingDir string
}

const serviceTemplate = `[Unit]
Description=HTTP Remote - DevOps Deployment Tool
Documentation=https://github.com/pandeptwidyaop/http-remote
After=network.target

[Service]
Type=simple
User={{.User}}
Group={{.User}}
WorkingDirectory={{.WorkingDir}}
ExecStart={{.ExecPath}} -config {{.ConfigPath}}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths={{.WorkingDir}}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsSystemdAvailable checks if systemctl command is available.
func IsSystemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// IsRoot checks if running as root user.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// GenerateServiceFile generates the systemd service file content.
func GenerateServiceFile(cfg ServiceConfig) (string, error) {
	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse service template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("failed to execute service template: %w", err)
	}

	return buf.String(), nil
}

// Install installs and starts the systemd service.
func Install(cfg ServiceConfig) error {
	if !IsLinux() {
		return fmt.Errorf("service installation only supported on Linux")
	}

	if !IsSystemdAvailable() {
		return fmt.Errorf("systemd not available on this system")
	}

	if !IsRoot() {
		return fmt.Errorf("root privileges required for service installation")
	}

	// Generate service file content
	content, err := GenerateServiceFile(cfg)
	if err != nil {
		return err
	}

	// Write service file
	if err := os.WriteFile(serviceFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd daemon
	if err := runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	if err := runSystemctl("enable", serviceName); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Start service
	if err := runSystemctl("start", serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Uninstall stops and removes the systemd service.
func Uninstall() error {
	if !IsLinux() {
		return fmt.Errorf("service uninstallation only supported on Linux")
	}

	if !IsSystemdAvailable() {
		return fmt.Errorf("systemd not available on this system")
	}

	if !IsRoot() {
		return fmt.Errorf("root privileges required for service uninstallation")
	}

	// Stop service (ignore error if not running)
	_ = runSystemctl("stop", serviceName)

	// Disable service (ignore error if not enabled)
	_ = runSystemctl("disable", serviceName)

	// Remove service file
	if err := os.Remove(serviceFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd daemon
	if err := runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	return nil
}

// Status returns the current service status.
func Status() (*ServiceStatus, error) {
	status := &ServiceStatus{}

	if !IsLinux() || !IsSystemdAvailable() {
		return status, nil
	}

	// Check if service file exists
	if _, err := os.Stat(serviceFilePath); err == nil {
		status.IsInstalled = true
	}

	// Get active state
	activeState, err := getSystemctlProperty("ActiveState")
	if err == nil {
		status.ActiveState = activeState
		status.IsRunning = activeState == "active"
	}

	// Get sub state
	subState, err := getSystemctlProperty("SubState")
	if err == nil {
		status.SubState = subState
	}

	// Check if enabled
	output, err := exec.Command("systemctl", "is-enabled", serviceName).Output()
	if err == nil {
		status.IsEnabled = strings.TrimSpace(string(output)) == "enabled"
	}

	return status, nil
}

// Restart restarts the systemd service.
func Restart() error {
	if !IsLinux() {
		return fmt.Errorf("service restart only supported on Linux")
	}

	if !IsSystemdAvailable() {
		return fmt.Errorf("systemd not available on this system")
	}

	return runSystemctl("restart", serviceName)
}

// Stop stops the systemd service.
func Stop() error {
	if !IsLinux() {
		return fmt.Errorf("service stop only supported on Linux")
	}

	if !IsSystemdAvailable() {
		return fmt.Errorf("systemd not available on this system")
	}

	return runSystemctl("stop", serviceName)
}

// Start starts the systemd service.
func Start() error {
	if !IsLinux() {
		return fmt.Errorf("service start only supported on Linux")
	}

	if !IsSystemdAvailable() {
		return fmt.Errorf("systemd not available on this system")
	}

	return runSystemctl("start", serviceName)
}

// IsRunningAsService checks if the current process is running as a systemd service.
func IsRunningAsService() bool {
	// Check if INVOCATION_ID is set (systemd sets this for services)
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}

	// Alternative: check if parent is systemd (PID 1)
	ppid := os.Getppid()
	return ppid == 1
}

// GetDefaultConfig returns default service configuration.
func GetDefaultConfig() ServiceConfig {
	execPath, _ := os.Executable()
	execPath, _ = filepath.EvalSymlinks(execPath)

	return ServiceConfig{
		ExecPath:   execPath,
		ConfigPath: "/etc/http-remote/config.yaml",
		User:       "root",
		WorkingDir: "/etc/http-remote",
	}
}

// runSystemctl executes a systemctl command.
func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// getSystemctlProperty gets a systemd property value.
func getSystemctlProperty(property string) (string, error) {
	cmd := exec.Command("systemctl", "show", serviceName, "--property="+property, "--value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
