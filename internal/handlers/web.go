package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

// WebHandler handles web page rendering.
type WebHandler struct {
	appService      *services.AppService
	executorService *services.ExecutorService
	pathPrefix      string
}

// NewWebHandler creates a new WebHandler instance.
func NewWebHandler(appService *services.AppService, executorService *services.ExecutorService, pathPrefix string) *WebHandler {
	return &WebHandler{
		appService:      appService,
		executorService: executorService,
		pathPrefix:      pathPrefix,
	}
}

// Dashboard renders the dashboard page.
func (h *WebHandler) Dashboard(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}

	apps, _ := h.appService.GetAllApps()
	executions, _ := h.executorService.GetExecutions(10, 0)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"PathPrefix": h.pathPrefix,
		"User":       u,
		"Apps":       apps,
		"Executions": executions,
		"Version":    version.Version,
	})
}

// AppsPage renders the applications list page.
func (h *WebHandler) AppsPage(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}

	apps, _ := h.appService.GetAllApps()

	c.HTML(http.StatusOK, "apps.html", gin.H{
		"PathPrefix": h.pathPrefix,
		"User":       u,
		"Apps":       apps,
		"Version":    version.Version,
	})
}

// AppDetailPage renders the application detail page.
func (h *WebHandler) AppDetailPage(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}

	id := c.Param("id")

	app, err := h.appService.GetAppByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, h.pathPrefix+"/apps")
		return
	}

	commands, _ := h.appService.GetCommandsByAppID(id)

	c.HTML(http.StatusOK, "app_detail.html", gin.H{
		"PathPrefix": h.pathPrefix,
		"User":       u,
		"App":        app,
		"Commands":   commands,
		"Version":    version.Version,
	})
}

// ExecutePage renders the command execution page.
func (h *WebHandler) ExecutePage(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}

	id := c.Param("id")

	cmd, err := h.appService.GetCommandByID(id)
	if err != nil {
		c.Redirect(http.StatusFound, h.pathPrefix+"/")
		return
	}

	app, _ := h.appService.GetAppByID(cmd.AppID)

	c.HTML(http.StatusOK, "execute.html", gin.H{
		"PathPrefix": h.pathPrefix,
		"User":       u,
		"Command":    cmd,
		"App":        app,
		"Version":    version.Version,
	})
}

// ExecutionsPage renders the executions list page.
func (h *WebHandler) ExecutionsPage(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.Redirect(http.StatusFound, h.pathPrefix+"/login")
		return
	}

	executions, _ := h.executorService.GetExecutions(50, 0)

	c.HTML(http.StatusOK, "executions.html", gin.H{
		"PathPrefix": h.pathPrefix,
		"User":       u,
		"Executions": executions,
		"Version":    version.Version,
	})
}
