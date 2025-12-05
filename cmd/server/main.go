package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/router"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

func main() {
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
		cfg = &config.Config{}
		config.Load("")
	}

	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	authService := services.NewAuthService(db, cfg)
	appService := services.NewAppService(db)
	executorService := services.NewExecutorService(db, cfg, appService)

	if err := authService.EnsureAdminUser(); err != nil {
		log.Printf("Warning: Could not ensure admin user: %v", err)
	}

	r := router.New(cfg, authService, appService, executorService)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("HTTP Remote %s starting on %s", version.Version, addr)
	log.Printf("Access at: http://%s%s", addr, cfg.Server.PathPrefix)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
