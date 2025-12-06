// Package handlers provides HTTP request handlers for the web UI and API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// AppHandler handles HTTP requests for application management.
type AppHandler struct {
	appService   *services.AppService
	auditService *services.AuditService
	pathPrefix   string
}

// NewAppHandler creates a new AppHandler instance.
func NewAppHandler(appService *services.AppService, auditService *services.AuditService, pathPrefix string) *AppHandler {
	return &AppHandler{
		appService:   appService,
		auditService: auditService,
		pathPrefix:   pathPrefix,
	}
}

// List returns all applications as JSON.
func (h *AppHandler) List(c *gin.Context) {
	apps, err := h.appService.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

// Get returns a single application by ID.
func (h *AppHandler) Get(c *gin.Context) {
	id := c.Param("id")

	app, err := h.appService.GetAppByID(id)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

// Create creates a new application.
func (h *AppHandler) Create(c *gin.Context) {
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.appService.CreateApp(&req)
	if err != nil {
		if err == services.ErrAppExists {
			c.JSON(http.StatusConflict, gin.H{"error": "app already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "create",
			ResourceType: "app",
			ResourceID:   app.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"app_name": app.Name},
		})
	}

	c.JSON(http.StatusCreated, app)
}

// Update updates an existing application.
func (h *AppHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.appService.UpdateApp(id, &req)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "update",
			ResourceType: "app",
			ResourceID:   app.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"app_name": app.Name},
		})
	}

	c.JSON(http.StatusOK, app)
}

// Delete deletes an application.
func (h *AppHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Get app info before deleting for audit log
	app, _ := h.appService.GetAppByID(id)

	if err := h.appService.DeleteApp(id); err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		details := map[string]interface{}{}
		if app != nil {
			details["app_name"] = app.Name
		}
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "delete",
			ResourceType: "app",
			ResourceID:   id,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      details,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

// RegenerateToken generates a new authentication token for an application.
func (h *AppHandler) RegenerateToken(c *gin.Context) {
	id := c.Param("id")

	app, err := h.appService.RegenerateToken(id)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "regenerate_token",
			ResourceType: "app",
			ResourceID:   app.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"app_name": app.Name},
		})
	}

	c.JSON(http.StatusOK, app)
}

// ListCommands returns all commands for an application.
func (h *AppHandler) ListCommands(c *gin.Context) {
	appID := c.Param("id")

	commands, err := h.appService.GetCommandsByAppID(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, commands)
}

// CreateCommand creates a new command for an application.
func (h *AppHandler) CreateCommand(c *gin.Context) {
	appID := c.Param("id")

	var req models.CreateCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd, err := h.appService.CreateCommand(appID, &req)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "create",
			ResourceType: "command",
			ResourceID:   cmd.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"command_name": cmd.Name, "app_id": appID},
		})
	}

	c.JSON(http.StatusCreated, cmd)
}
