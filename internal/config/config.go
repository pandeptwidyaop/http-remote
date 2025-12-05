// Package config provides configuration loading and management for the HTTP Remote application.
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration structure.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	Execution ExecutionConfig `yaml:"execution"`
	Admin     AdminConfig     `yaml:"admin"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	PathPrefix   string `yaml:"path_prefix"`
	SecureCookie bool   `yaml:"secure_cookie"`
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// AuthConfig holds authentication and session configuration.
type AuthConfig struct {
	SessionDuration string `yaml:"session_duration"`
	BcryptCost      int    `yaml:"bcrypt_cost"`
}

// ExecutionConfig holds command execution configuration.
type ExecutionConfig struct {
	DefaultTimeout int `yaml:"default_timeout"`
	MaxTimeout     int `yaml:"max_timeout"`
	MaxOutputSize  int `yaml:"max_output_size"`
}

// AdminConfig holds admin user credentials.
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// GetSessionDuration parses and returns the session duration as time.Duration.
func (c *AuthConfig) GetSessionDuration() time.Duration {
	d, err := time.ParseDuration(c.SessionDuration)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// Load reads and parses a configuration file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	setDefaults(&cfg)

	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.PathPrefix == "" {
		cfg.Server.PathPrefix = "/devops"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/deploy.db"
	}
	if cfg.Auth.SessionDuration == "" {
		cfg.Auth.SessionDuration = "24h"
	}
	if cfg.Auth.BcryptCost == 0 {
		cfg.Auth.BcryptCost = 12
	}
	if cfg.Execution.DefaultTimeout == 0 {
		cfg.Execution.DefaultTimeout = 300
	}
	if cfg.Execution.MaxTimeout == 0 {
		cfg.Execution.MaxTimeout = 3600
	}
	if cfg.Execution.MaxOutputSize == 0 {
		cfg.Execution.MaxOutputSize = 10485760
	}
	if cfg.Admin.Username == "" {
		cfg.Admin.Username = "admin"
	}
	if cfg.Admin.Password == "" {
		cfg.Admin.Password = "changeme"
	}
}
