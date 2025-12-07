// Package main is the entry point for the HTTP Remote server.
package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/router"
	"github.com/pandeptwidyaop/http-remote/internal/service"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/upgrade"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

func main() {
	// Check for subcommands first
	if len(os.Args) > 1 {
		// Skip flag-like arguments (they'll be handled by flag.Parse)
		arg := os.Args[1]
		if len(arg) > 0 && arg[0] != '-' {
			switch arg {
			case "help":
				printHelp()
				os.Exit(0)
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
			case "install-service":
				runInstallService()
				os.Exit(0)
			case "uninstall-service":
				runUninstallService()
				os.Exit(0)
			case "service":
				if len(os.Args) > 2 {
					runServiceCommand(os.Args[2])
				} else {
					fmt.Println("Usage: http-remote service <status|start|stop|restart>")
				}
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n\n", arg)
				printHelp()
				os.Exit(1)
			}
		}
	}

	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version information")
	showHelp := flag.Bool("help", false, "show help information")
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

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

// runInstallService handles the install-service subcommand.
func runInstallService() {
	if !service.IsLinux() {
		fmt.Println("Warning: Service installation is only supported on Linux with systemd.")
		fmt.Println("On other platforms, please run the binary directly or use your OS's service manager.")
		os.Exit(1)
	}

	if !service.IsSystemdAvailable() {
		fmt.Println("Error: systemd is not available on this system.")
		os.Exit(1)
	}

	if !service.IsRoot() {
		fmt.Println("Error: This command requires root privileges.")
		fmt.Println("Please run with sudo: sudo ./http-remote install-service")
		os.Exit(1)
	}

	// Parse flags for install-service
	fs := flag.NewFlagSet("install-service", flag.ExitOnError)
	user := fs.String("user", "root", "User to run the service as")
	configPath := fs.String("config", "/etc/http-remote/config.yaml", "Path to config file")
	workDir := fs.String("workdir", "/etc/http-remote", "Working directory for the service")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving executable path: %v\n", err)
		os.Exit(1)
	}

	cfg := service.ServiceConfig{
		ExecPath:   execPath,
		ConfigPath: *configPath,
		User:       *user,
		WorkingDir: *workDir,
	}

	fmt.Println("Installing HTTP Remote as systemd service...")
	fmt.Printf("  Binary: %s\n", cfg.ExecPath)
	fmt.Printf("  Config: %s\n", cfg.ConfigPath)
	fmt.Printf("  User: %s\n", cfg.User)
	fmt.Printf("  Working Dir: %s\n", cfg.WorkingDir)

	if err := service.Install(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println("Service installed and started successfully!")
	fmt.Println("")
	fmt.Println("Useful commands:")
	fmt.Println("  sudo systemctl status http-remote   - Check service status")
	fmt.Println("  sudo systemctl restart http-remote  - Restart service")
	fmt.Println("  sudo journalctl -u http-remote -f   - View logs")
}

// runUninstallService handles the uninstall-service subcommand.
func runUninstallService() {
	if !service.IsLinux() {
		fmt.Println("Warning: Service uninstallation is only supported on Linux with systemd.")
		os.Exit(1)
	}

	if !service.IsSystemdAvailable() {
		fmt.Println("Error: systemd is not available on this system.")
		os.Exit(1)
	}

	if !service.IsRoot() {
		fmt.Println("Error: This command requires root privileges.")
		fmt.Println("Please run with sudo: sudo ./http-remote uninstall-service")
		os.Exit(1)
	}

	fmt.Println("Uninstalling HTTP Remote systemd service...")

	if err := service.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Error uninstalling service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service uninstalled successfully!")
}

// printHelp displays all available commands.
func printHelp() {
	fmt.Printf("HTTP Remote %s - DevOps Deployment Tool\n\n", version.Version)
	fmt.Println("Usage:")
	fmt.Println("  http-remote [command] [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  (no command)       Start the HTTP server")
	fmt.Println("  help               Show this help message")
	fmt.Println("  version            Show version information")
	fmt.Println("  upgrade [-f]       Upgrade to the latest version (-f to force)")
	fmt.Println("  install-service    Install as systemd service (Linux only, requires sudo)")
	fmt.Println("  uninstall-service  Uninstall systemd service (Linux only, requires sudo)")
	fmt.Println("  service <cmd>      Manage systemd service (Linux only)")
	fmt.Println("                     Commands: status, start, stop, restart")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -config string     Path to config file (default \"config.yaml\")")
	fmt.Println("  -version           Show version information")
	fmt.Println("  -help              Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  http-remote                          # Start server with default config")
	fmt.Println("  http-remote -config /etc/app.yaml    # Start server with custom config")
	fmt.Println("  http-remote upgrade                  # Upgrade to latest version")
	fmt.Println("  http-remote service status           # Check service status")
	fmt.Println("  sudo http-remote install-service     # Install as systemd service")
}

// runServiceCommand handles service management subcommands.
func runServiceCommand(cmd string) {
	if !service.IsLinux() {
		fmt.Println("Warning: Service management is only supported on Linux with systemd.")
		os.Exit(1)
	}

	if !service.IsSystemdAvailable() {
		fmt.Println("Error: systemd is not available on this system.")
		os.Exit(1)
	}

	switch cmd {
	case "status":
		status, err := service.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting service status: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("HTTP Remote Service Status:")
		fmt.Printf("  Installed: %v\n", status.IsInstalled)
		fmt.Printf("  Enabled: %v\n", status.IsEnabled)
		fmt.Printf("  Running: %v\n", status.IsRunning)
		fmt.Printf("  State: %s (%s)\n", status.ActiveState, status.SubState)

	case "start":
		if !service.IsRoot() {
			fmt.Println("Error: This command requires root privileges.")
			os.Exit(1)
		}
		if err := service.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service started successfully!")

	case "stop":
		if !service.IsRoot() {
			fmt.Println("Error: This command requires root privileges.")
			os.Exit(1)
		}
		if err := service.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service stopped successfully!")

	case "restart":
		if !service.IsRoot() {
			fmt.Println("Error: This command requires root privileges.")
			os.Exit(1)
		}
		if err := service.Restart(); err != nil {
			fmt.Fprintf(os.Stderr, "Error restarting service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service restarted successfully!")

	default:
		fmt.Printf("Unknown service command: %s\n", cmd)
		fmt.Println("Usage: http-remote service <status|start|stop|restart>")
		os.Exit(1)
	}
}
