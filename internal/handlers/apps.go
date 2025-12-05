package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type AppHandler struct {
	appService *services.AppService
	pathPrefix string
}

func NewAppHandler(appService *services.AppService, pathPrefix string) *AppHandler {
	return &AppHandler{
		appService: appService,
		pathPrefix: pathPrefix,
	}
}

func (h *AppHandler) List(c *gin.Context) {
	apps, err := h.appService.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apps)
}

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

	c.JSON(http.StatusCreated, app)
}

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

	c.JSON(http.StatusOK, app)
}

func (h *AppHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.DeleteApp(id); err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

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

	c.JSON(http.StatusOK, app)
}

func (h *AppHandler) ListCommands(c *gin.Context) {
	appID := c.Param("id")

	commands, err := h.appService.GetCommandsByAppID(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, commands)
}

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

	c.JSON(http.StatusCreated, cmd)
}
