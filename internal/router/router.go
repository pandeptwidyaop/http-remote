package router

import (
	"html/template"
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

func New(cfg *config.Config, authService *services.AuthService, appService *services.AppService, executorService *services.ExecutorService, auditService *services.AuditService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.PathPrefix(cfg.Server.PathPrefix))

	// Load templates from embedded filesystem
	tmpl := template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})

	templatesFS := assets.GetTemplatesFS()
	tmpl, err := tmpl.ParseFS(templatesFS, "*.html")
	if err != nil {
		panic("Failed to parse templates: " + err.Error())
	}
	r.SetHTMLTemplate(tmpl)

	// Serve static files from embedded filesystem using individual routes
	staticFS, _ := fs.Sub(assets.EmbeddedFiles, "web/static")
	staticHandler := http.FileServer(http.FS(staticFS))
	r.GET(cfg.Server.PathPrefix+"/static/*filepath", func(c *gin.Context) {
		c.Request.URL.Path = c.Param("filepath")
		staticHandler.ServeHTTP(c.Writer, c.Request)
	})

	prefix := r.Group(cfg.Server.PathPrefix)

	authHandler := handlers.NewAuthHandler(authService, auditService, cfg.Server.PathPrefix, cfg.Server.SecureCookie)
	appHandler := handlers.NewAppHandler(appService, cfg.Server.PathPrefix)
	commandHandler := handlers.NewCommandHandler(appService, executorService, auditService, cfg.Server.PathPrefix)
	streamHandler := handlers.NewStreamHandler(executorService)
	webHandler := handlers.NewWebHandler(appService, executorService, cfg.Server.PathPrefix)
	deployHandler := handlers.NewDeployHandler(appService, executorService, cfg.Server.PathPrefix)

	// Rate limiters
	loginLimiter := middleware.NewRateLimiter(5, time.Minute)     // 5 req/min for login
	apiLimiter := middleware.NewRateLimiter(60, time.Minute)      // 60 req/min for API
	deployLimiter := middleware.NewRateLimiter(30, time.Minute)   // 30 req/min for deploy

	prefix.GET("/login", authHandler.LoginPage)
	prefix.POST("/login", loginLimiter.Middleware(), authHandler.Login)

	// Public deploy endpoint (token auth) with rate limiting
	prefix.POST("/deploy/:app_id", deployLimiter.Middleware(), deployHandler.Deploy)
	prefix.GET("/deploy/:app_id/status/:execution_id", apiLimiter.Middleware(), deployHandler.DeployStatus)

	api := prefix.Group("/api")
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

			protected.GET("/apps", appHandler.List)
			protected.POST("/apps", appHandler.Create)
			protected.GET("/apps/:id", appHandler.Get)
			protected.PUT("/apps/:id", appHandler.Update)
			protected.DELETE("/apps/:id", appHandler.Delete)
			protected.POST("/apps/:id/regenerate-token", appHandler.RegenerateToken)
			protected.GET("/apps/:id/commands", appHandler.ListCommands)
			protected.POST("/apps/:id/commands", appHandler.CreateCommand)

			protected.GET("/commands/:id", commandHandler.Get)
			protected.PUT("/commands/:id", commandHandler.Update)
			protected.DELETE("/commands/:id", commandHandler.Delete)
			protected.POST("/commands/:id/execute", commandHandler.Execute)

			protected.GET("/executions", commandHandler.ListExecutions)
			protected.GET("/executions/:id", commandHandler.GetExecution)
			protected.GET("/executions/:id/stream", streamHandler.Stream)
		}
	}

	web := prefix.Group("")
	web.Use(middleware.AuthRequired(authService))
	{
		web.GET("/", webHandler.Dashboard)
		web.GET("/apps", webHandler.AppsPage)
		web.GET("/apps/:id", webHandler.AppDetailPage)
		web.GET("/execute/:id", webHandler.ExecutePage)
		web.GET("/executions", webHandler.ExecutionsPage)
		web.POST("/logout", authHandler.Logout)
	}

	// Redirect root to path prefix (only if prefix is not empty)
	if cfg.Server.PathPrefix != "" && cfg.Server.PathPrefix != "/" {
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, cfg.Server.PathPrefix+"/")
		})
	}

	return r
}
