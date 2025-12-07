package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create temp config file
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
server:
  host: "127.0.0.1"
  port: 9090
  path_prefix: "/api"
  secure_cookie: true

database:
  path: "/data/test.db"

auth:
  session_duration: "12h"
  bcrypt_cost: 10

execution:
  default_timeout: 600
  max_timeout: 7200
  max_output_size: 5242880

admin:
  username: "testadmin"
  password: "testpass"

terminal:
  shell: "/bin/zsh"
  args: ["-i"]
  env:
    - "TERM=xterm-256color"

security:
  encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  max_login_attempts: 3
  lockout_duration: "30m"

files:
  allowed_paths:
    - "/home/user"
    - "/opt/apps"
  blocked_paths:
    - "/home/user/.ssh"
`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify server config
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.PathPrefix != "/api" {
		t.Errorf("expected path_prefix '/api', got '%s'", cfg.Server.PathPrefix)
	}
	if !cfg.Server.SecureCookie {
		t.Error("expected secure_cookie to be true")
	}

	// Verify database config
	if cfg.Database.Path != "/data/test.db" {
		t.Errorf("expected database path '/data/test.db', got '%s'", cfg.Database.Path)
	}

	// Verify auth config
	if cfg.Auth.SessionDuration != "12h" {
		t.Errorf("expected session_duration '12h', got '%s'", cfg.Auth.SessionDuration)
	}
	if cfg.Auth.BcryptCost != 10 {
		t.Errorf("expected bcrypt_cost 10, got %d", cfg.Auth.BcryptCost)
	}

	// Verify execution config
	if cfg.Execution.DefaultTimeout != 600 {
		t.Errorf("expected default_timeout 600, got %d", cfg.Execution.DefaultTimeout)
	}
	if cfg.Execution.MaxTimeout != 7200 {
		t.Errorf("expected max_timeout 7200, got %d", cfg.Execution.MaxTimeout)
	}
	if cfg.Execution.MaxOutputSize != 5242880 {
		t.Errorf("expected max_output_size 5242880, got %d", cfg.Execution.MaxOutputSize)
	}

	// Verify admin config
	if cfg.Admin.Username != "testadmin" {
		t.Errorf("expected username 'testadmin', got '%s'", cfg.Admin.Username)
	}
	if cfg.Admin.Password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", cfg.Admin.Password)
	}

	// Verify terminal config
	if cfg.Terminal.Shell != "/bin/zsh" {
		t.Errorf("expected shell '/bin/zsh', got '%s'", cfg.Terminal.Shell)
	}
	if len(cfg.Terminal.Args) != 1 || cfg.Terminal.Args[0] != "-i" {
		t.Errorf("expected args ['-i'], got %v", cfg.Terminal.Args)
	}
	if len(cfg.Terminal.Env) != 1 || cfg.Terminal.Env[0] != "TERM=xterm-256color" {
		t.Errorf("expected env ['TERM=xterm-256color'], got %v", cfg.Terminal.Env)
	}

	// Verify security config
	if cfg.Security.MaxLoginAttempts != 3 {
		t.Errorf("expected max_login_attempts 3, got %d", cfg.Security.MaxLoginAttempts)
	}
	if cfg.Security.LockoutDuration != "30m" {
		t.Errorf("expected lockout_duration '30m', got '%s'", cfg.Security.LockoutDuration)
	}

	// Verify files config
	if len(cfg.Files.AllowedPaths) != 2 {
		t.Errorf("expected 2 allowed_paths, got %d", len(cfg.Files.AllowedPaths))
	}
	if cfg.Files.AllowedPaths[0] != "/home/user" {
		t.Errorf("expected first allowed_path '/home/user', got '%s'", cfg.Files.AllowedPaths[0])
	}
	if cfg.Files.AllowedPaths[1] != "/opt/apps" {
		t.Errorf("expected second allowed_path '/opt/apps', got '%s'", cfg.Files.AllowedPaths[1])
	}
	if len(cfg.Files.BlockedPaths) != 1 {
		t.Errorf("expected 1 blocked_path, got %d", len(cfg.Files.BlockedPaths))
	}
	if cfg.Files.BlockedPaths[0] != "/home/user/.ssh" {
		t.Errorf("expected blocked_path '/home/user/.ssh', got '%s'", cfg.Files.BlockedPaths[0])
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Create minimal config file
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "config.yaml")
	// Empty config to test defaults
	err = os.WriteFile(configPath, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify defaults
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.PathPrefix != "/devops" {
		t.Errorf("expected default path_prefix '/devops', got '%s'", cfg.Server.PathPrefix)
	}
	if cfg.Database.Path != "./data/deploy.db" {
		t.Errorf("expected default database path './data/deploy.db', got '%s'", cfg.Database.Path)
	}
	if cfg.Auth.SessionDuration != "24h" {
		t.Errorf("expected default session_duration '24h', got '%s'", cfg.Auth.SessionDuration)
	}
	if cfg.Auth.BcryptCost != 12 {
		t.Errorf("expected default bcrypt_cost 12, got %d", cfg.Auth.BcryptCost)
	}
	if cfg.Execution.DefaultTimeout != 300 {
		t.Errorf("expected default default_timeout 300, got %d", cfg.Execution.DefaultTimeout)
	}
	if cfg.Execution.MaxTimeout != 3600 {
		t.Errorf("expected default max_timeout 3600, got %d", cfg.Execution.MaxTimeout)
	}
	if cfg.Execution.MaxOutputSize != 10485760 {
		t.Errorf("expected default max_output_size 10485760, got %d", cfg.Execution.MaxOutputSize)
	}
	if cfg.Admin.Username != "admin" {
		t.Errorf("expected default username 'admin', got '%s'", cfg.Admin.Username)
	}
	if cfg.Admin.Password != "changeme" {
		t.Errorf("expected default password 'changeme', got '%s'", cfg.Admin.Password)
	}
	if cfg.Terminal.Shell != "/bin/bash" {
		t.Errorf("expected default shell '/bin/bash', got '%s'", cfg.Terminal.Shell)
	}
	if len(cfg.Terminal.Args) != 1 || cfg.Terminal.Args[0] != "-l" {
		t.Errorf("expected default args ['-l'], got %v", cfg.Terminal.Args)
	}

	// Files config should be empty by default
	if len(cfg.Files.AllowedPaths) != 0 {
		t.Errorf("expected empty allowed_paths by default, got %v", cfg.Files.AllowedPaths)
	}
	if len(cfg.Files.BlockedPaths) != 0 {
		t.Errorf("expected empty blocked_paths by default, got %v", cfg.Files.BlockedPaths)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent config file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "config.yaml")
	// Invalid YAML content
	err = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestTerminalConfig_IsEnabled(t *testing.T) {
	// Test default (nil) - should be enabled
	cfg := &TerminalConfig{}
	if !cfg.IsEnabled() {
		t.Error("expected terminal to be enabled by default")
	}

	// Test explicitly enabled
	enabled := true
	cfg.Enabled = &enabled
	if !cfg.IsEnabled() {
		t.Error("expected terminal to be enabled when set to true")
	}

	// Test explicitly disabled
	disabled := false
	cfg.Enabled = &disabled
	if cfg.IsEnabled() {
		t.Error("expected terminal to be disabled when set to false")
	}
}

func TestSecurityConfig_GetLockoutDuration(t *testing.T) {
	// Test default (empty string)
	cfg := &SecurityConfig{}
	if cfg.GetLockoutDuration() != 15*time.Minute {
		t.Errorf("expected default lockout duration 15m, got %v", cfg.GetLockoutDuration())
	}

	// Test valid duration
	cfg.LockoutDuration = "30m"
	if cfg.GetLockoutDuration() != 30*time.Minute {
		t.Errorf("expected lockout duration 30m, got %v", cfg.GetLockoutDuration())
	}

	// Test invalid duration (should return default)
	cfg.LockoutDuration = "invalid"
	if cfg.GetLockoutDuration() != 15*time.Minute {
		t.Errorf("expected default lockout duration for invalid input, got %v", cfg.GetLockoutDuration())
	}
}

func TestSecurityConfig_GetMaxLoginAttempts(t *testing.T) {
	// Test default (zero value)
	cfg := &SecurityConfig{}
	if cfg.GetMaxLoginAttempts() != 5 {
		t.Errorf("expected default max login attempts 5, got %d", cfg.GetMaxLoginAttempts())
	}

	// Test custom value
	cfg.MaxLoginAttempts = 10
	if cfg.GetMaxLoginAttempts() != 10 {
		t.Errorf("expected max login attempts 10, got %d", cfg.GetMaxLoginAttempts())
	}
}

func TestAuthConfig_GetSessionDuration(t *testing.T) {
	// Test default (empty string) - should return 24h
	cfg := &AuthConfig{}
	if cfg.GetSessionDuration() != 24*time.Hour {
		t.Errorf("expected default session duration 24h, got %v", cfg.GetSessionDuration())
	}

	// Test valid duration
	cfg.SessionDuration = "12h"
	if cfg.GetSessionDuration() != 12*time.Hour {
		t.Errorf("expected session duration 12h, got %v", cfg.GetSessionDuration())
	}

	// Test invalid duration (should return default)
	cfg.SessionDuration = "invalid"
	if cfg.GetSessionDuration() != 24*time.Hour {
		t.Errorf("expected default session duration for invalid input, got %v", cfg.GetSessionDuration())
	}
}

func TestFilesConfig_EmptyByDefault(t *testing.T) {
	cfg := &FilesConfig{}

	if len(cfg.AllowedPaths) > 0 {
		t.Error("expected AllowedPaths to be empty by default")
	}

	if len(cfg.BlockedPaths) > 0 {
		t.Error("expected BlockedPaths to be empty by default")
	}
}

func TestLoad_FilesConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("only allowed_paths", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "allowed_only.yaml")
		content := `
files:
  allowed_paths:
    - "/home/user"
    - "/var/www"
`
		err = os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if len(cfg.Files.AllowedPaths) != 2 {
			t.Errorf("expected 2 allowed paths, got %d", len(cfg.Files.AllowedPaths))
		}
		if len(cfg.Files.BlockedPaths) != 0 {
			t.Errorf("expected 0 blocked paths, got %d", len(cfg.Files.BlockedPaths))
		}
	})

	t.Run("only blocked_paths", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "blocked_only.yaml")
		content := `
files:
  blocked_paths:
    - "/etc/passwd"
    - "/root"
`
		err = os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if len(cfg.Files.AllowedPaths) != 0 {
			t.Errorf("expected 0 allowed paths, got %d", len(cfg.Files.AllowedPaths))
		}
		if len(cfg.Files.BlockedPaths) != 2 {
			t.Errorf("expected 2 blocked paths, got %d", len(cfg.Files.BlockedPaths))
		}
	})

	t.Run("both allowed and blocked", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "both.yaml")
		content := `
files:
  allowed_paths:
    - "/home/devops"
  blocked_paths:
    - "/home/devops/.ssh"
    - "/home/devops/.gnupg"
`
		err = os.WriteFile(configPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if len(cfg.Files.AllowedPaths) != 1 {
			t.Errorf("expected 1 allowed path, got %d", len(cfg.Files.AllowedPaths))
		}
		if len(cfg.Files.BlockedPaths) != 2 {
			t.Errorf("expected 2 blocked paths, got %d", len(cfg.Files.BlockedPaths))
		}
	})
}
