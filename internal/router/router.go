// Package router sets up HTTP routes and middleware for the web application.
package router

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/assets"
	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/handlers"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

// New creates and configures a new Gin router with all routes and middleware.
func New(cfg *config.Config, authService *services.AuthService, appService *services.AppService, executorService *services.ExecutorService, auditService *services.AuditService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.StrictTransportSecurity(31536000))
	r.Use(middleware.PathPrefix(cfg.Server.PathPrefix))
	r.Use(middleware.DefaultBodyLimit()) // 1MB request body limit

	// Initialize CSRF store for token management
	csrfStore := middleware.NewCSRFStore()

	// Serve SPA static files from embedded filesystem
	distFS, err := fs.Sub(assets.EmbeddedFiles, "web/dist")
	if err != nil {
		panic("Failed to load embedded SPA assets: " + err.Error())
	}

	// Serve static assets (JS, CSS, etc.) under path prefix
	// This ensures assets are served from /{prefix}/assets
	assetHandler := func(c *gin.Context) {
		filepath := c.Param("filepath")
		assetPath := "assets" + filepath

		data, err := fs.ReadFile(distFS, assetPath)
		if err != nil {
			c.String(http.StatusNotFound, "Asset not found")
			return
		}

		// Determine content type based on file extension
		contentType := "application/octet-stream"
		if len(filepath) > 3 && filepath[len(filepath)-3:] == ".js" {
			contentType = "application/javascript"
		} else if len(filepath) > 4 && filepath[len(filepath)-4:] == ".css" {
			contentType = "text/css"
		}

		c.Data(http.StatusOK, contentType, data)
	}

	// Handle path_prefix "/" or "" by avoiding double slashes
	assetPath := "/assets/*filepath"
	if cfg.Server.PathPrefix != "" && cfg.Server.PathPrefix != "/" {
		assetPath = cfg.Server.PathPrefix + "/assets/*filepath"
	}
	r.GET(assetPath, assetHandler)
	r.HEAD(assetPath, assetHandler)

	prefix := r.Group(cfg.Server.PathPrefix)

	authHandler := handlers.NewAuthHandler(authService, auditService, cfg.Server.PathPrefix, cfg.Server.SecureCookie)
	twoFAHandler := handlers.NewTwoFAHandler(authService, auditService)
	appHandler := handlers.NewAppHandler(appService, auditService, cfg.Server.PathPrefix)
	commandHandler := handlers.NewCommandHandler(appService, executorService, auditService, cfg.Server.PathPrefix)
	streamHandler := handlers.NewStreamHandler(executorService)
	deployHandler := handlers.NewDeployHandler(appService, executorService, cfg.Server.PathPrefix)
	auditHandler := handlers.NewAuditHandler(auditService, cfg.Server.PathPrefix)
	versionHandler := handlers.NewVersionHandler()
	terminalHandler := handlers.NewTerminalHandler(&cfg.Terminal, auditService, cfg.Server.AllowedOrigins)
	backupHandler := handlers.NewBackupHandler(appService, auditService)
	fileHandler := handlers.NewFileHandler(cfg, auditService)
	userHandler := handlers.NewUserHandler(authService, auditService, cfg)

	// Rate limiters
	loginLimiter := middleware.NewRateLimiter(5, time.Minute)   // 5 req/min for login
	apiLimiter := middleware.NewRateLimiter(60, time.Minute)    // 60 req/min for API
	deployLimiter := middleware.NewRateLimiter(30, time.Minute) // 30 req/min for deploy
	twoFALimiter := middleware.NewRateLimiter(10, time.Minute)  // 10 req/min for 2FA operations

	// Public deploy endpoint (token auth) with rate limiting
	prefix.POST("/deploy/:app_id", deployLimiter.Middleware(), deployHandler.Deploy)
	prefix.GET("/deploy/:app_id/status/:execution_id", apiLimiter.Middleware(), deployHandler.DeployStatus)
	prefix.GET("/deploy/:app_id/stream/:execution_id", apiLimiter.Middleware(), deployHandler.DeployStream)

	api := prefix.Group("/api")
	// Apply CSRF protection to all API routes
	api.Use(middleware.CSRFProtection(csrfStore, cfg.Server.PathPrefix, cfg.Server.SecureCookie))
	{
		// Public version endpoint
		api.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, version.Info())
		})

		api.POST("/auth/login", loginLimiter.Middleware(), authHandler.Login)
		api.POST("/auth/logout", authHandler.Logout)

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(authService))
		{
			protected.GET("/auth/me", authHandler.Me)

			// 2FA endpoints (rate limited to prevent brute force)
			protected.GET("/2fa/status", twoFAHandler.GetStatus)
			protected.POST("/2fa/generate-secret", twoFALimiter.Middleware(), twoFAHandler.GenerateSecret)
			protected.GET("/2fa/qrcode", twoFAHandler.GetQRCode)
			protected.POST("/2fa/enable", twoFALimiter.Middleware(), twoFAHandler.EnableTOTP)
			protected.POST("/2fa/disable", twoFALimiter.Middleware(), twoFAHandler.DisableTOTP)

			// Password management
			protected.POST("/auth/change-password", authHandler.ChangePassword)

			protected.GET("/apps", appHandler.List)
			protected.POST("/apps", appHandler.Create)
			protected.GET("/apps/:id", appHandler.Get)
			protected.PUT("/apps/:id", appHandler.Update)
			protected.DELETE("/apps/:id", appHandler.Delete)
			protected.POST("/apps/:id/regenerate-token", appHandler.RegenerateToken)
			protected.GET("/apps/:id/commands", appHandler.ListCommands)
			protected.POST("/apps/:id/commands", appHandler.CreateCommand)
			protected.POST("/apps/:id/commands/reorder", appHandler.ReorderCommands)

			protected.GET("/commands/:id", commandHandler.Get)
			protected.PUT("/commands/:id", commandHandler.Update)
			protected.DELETE("/commands/:id", commandHandler.Delete)
			protected.POST("/commands/:id/execute", commandHandler.Execute)

			protected.GET("/executions", commandHandler.ListExecutions)
			protected.GET("/executions/:id", commandHandler.GetExecution)
			protected.GET("/executions/:id/stream", streamHandler.Stream)

			protected.GET("/audit-logs", auditHandler.List)
			protected.GET("/version/check", versionHandler.CheckUpdate)

			// Backup endpoints
			protected.GET("/backup/export", backupHandler.Export)
			protected.POST("/backup/import", backupHandler.Import)
			protected.GET("/apps/:id/export", backupHandler.ExportApp)

			// Terminal endpoints
			protected.GET("/terminal/ws", terminalHandler.HandleWebSocket)
			protected.GET("/terminal/sessions", terminalHandler.ListSessions)
			protected.POST("/terminal/sessions", terminalHandler.CreateSession)
			protected.DELETE("/terminal/sessions/:session_id", terminalHandler.CloseSession)

			// File management endpoints
			protected.GET("/files", fileHandler.ListFiles)
			protected.GET("/files/default-path", fileHandler.GetDefaultPath)
			protected.GET("/files/read", fileHandler.ReadFile)
			protected.GET("/files/download", fileHandler.DownloadFile)
			protected.POST("/files/upload", fileHandler.UploadFile)
			protected.POST("/files/mkdir", fileHandler.CreateDirectory)
			protected.POST("/files/save", fileHandler.SaveFile)
			protected.POST("/files/rename", fileHandler.RenameFile)
			protected.POST("/files/copy", fileHandler.CopyFile)
			protected.DELETE("/files", fileHandler.DeleteFile)

			// User management endpoints (admin only)
			protected.GET("/users", userHandler.List)
			protected.POST("/users", userHandler.Create)
			protected.GET("/users/:id", userHandler.Get)
			protected.PUT("/users/:id", userHandler.Update)
			protected.PUT("/users/:id/password", userHandler.UpdatePassword)
			protected.DELETE("/users/:id", userHandler.Delete)
		}
	}

	// Serve SPA at the path prefix
	// This handles React Router's HashRouter
	spaPath := "/"
	if cfg.Server.PathPrefix != "" && cfg.Server.PathPrefix != "/" {
		spaPath = cfg.Server.PathPrefix + "/"
	}
	r.GET(spaPath, func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load index.html: "+err.Error())
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// Also handle without trailing slash
	if cfg.Server.PathPrefix != "" && cfg.Server.PathPrefix != "/" {
		r.GET(cfg.Server.PathPrefix, func(c *gin.Context) {
			data, err := fs.ReadFile(distFS, "index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to load index.html: "+err.Error())
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	}

	// Redirect root to path prefix (only if prefix is not empty)
	if cfg.Server.PathPrefix != "" && cfg.Server.PathPrefix != "/" {
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, cfg.Server.PathPrefix+"/")
		})
	}

	// NoRoute catch-all for any unmatched routes
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return r
}
