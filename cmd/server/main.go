// Package main is the entry point for the HTTP Remote server.
package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/router"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/upgrade"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

func main() {
	// Check for subcommands first
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "upgrade":
			force := false
			if len(os.Args) > 2 && (os.Args[2] == "-f" || os.Args[2] == "--force") {
				force = true
			}
			if err := upgrade.Run(force); err != nil {
				fmt.Fprintf(os.Stderr, "Upgrade failed: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "version":
			fmt.Printf("HTTP Remote %s\n", version.Version)
			fmt.Printf("Build Time: %s\n", version.BuildTime)
			fmt.Printf("Git Commit: %s\n", version.GitCommit)
			os.Exit(0)
		}
	}

	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("HTTP Remote %s\n", version.Version)
		fmt.Printf("Build Time: %s\n", version.BuildTime)
		fmt.Printf("Git Commit: %s\n", version.GitCommit)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config from %s: %v", *configPath, err)
		log.Println("Using default configuration...")
		// Ignore error for default config as it's already handled
		cfg, _ = config.Load("")
	}

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize crypto service for TOTP secret encryption - REQUIRED
	var cryptoService *services.CryptoService
	if cfg.Security.EncryptionKey == "" {
		log.Println("")
		log.Println("╔══════════════════════════════════════════════════════════════════╗")
		log.Println("║  SECURITY ERROR: Encryption key not configured!                  ║")
		log.Println("║                                                                  ║")
		log.Println("║  Please add 'security.encryption_key' to config.yaml.           ║")
		log.Println("║  Generate a key with: openssl rand -hex 32                       ║")
		log.Println("║                                                                  ║")
		log.Println("║  Example:                                                        ║")
		log.Println("║    security:                                                     ║")
		log.Println("║      encryption_key: \"<64-character-hex-string>\"                 ║")
		log.Println("╚══════════════════════════════════════════════════════════════════╝")
		log.Println("")
		log.Fatalf("Application startup aborted: encryption key is required")
	}

	key, err := hex.DecodeString(cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatalf("Invalid encryption key (must be 64 hex chars for 32 bytes): %v", err)
	}
	if len(key) != 32 {
		log.Fatalf("Invalid encryption key length: expected 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	cryptoService, err = services.NewCryptoService(key)
	if err != nil {
		log.Fatalf("Failed to initialize crypto service: %v", err)
	}
	log.Println("Encryption enabled for sensitive data (TOTP secrets)")

	authService := services.NewAuthService(db, cfg, cryptoService)
	appService := services.NewAppService(db)
	executorService := services.NewExecutorService(db, cfg, appService)
	auditService := services.NewAuditService(db)

	if err := authService.EnsureAdminUser(); err != nil {
		if errors.Is(err, services.ErrDefaultPassword) {
			log.Println("")
			log.Println("╔══════════════════════════════════════════════════════════════════╗")
			log.Println("║  SECURITY ERROR: Default admin password detected!                ║")
			log.Println("║                                                                  ║")
			log.Println("║  Please change 'admin.password' in config.yaml from 'changeme'  ║")
			log.Println("║  to a secure password before starting the application.          ║")
			log.Println("║                                                                  ║")
			log.Println("║  Example:                                                        ║")
			log.Println("║    admin:                                                        ║")
			log.Println("║      username: \"admin\"                                           ║")
			log.Println("║      password: \"YourSecurePassword123!\"                          ║")
			log.Println("╚══════════════════════════════════════════════════════════════════╝")
			log.Println("")
			log.Fatalf("Application startup aborted: %v", err)
		}
		log.Fatalf("Failed to ensure admin user: %v", err)
	}

	r := router.New(cfg, authService, appService, executorService, auditService)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("HTTP Remote %s starting on %s", version.Version, addr)
	log.Printf("Access at: http://%s%s", addr, cfg.Server.PathPrefix)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
