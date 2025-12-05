package handlers

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type DeployHandler struct {
	appService      *services.AppService
	executorService *services.ExecutorService
	pathPrefix      string
}

func NewDeployHandler(appService *services.AppService, executorService *services.ExecutorService, pathPrefix string) *DeployHandler {
	return &DeployHandler{
		appService:      appService,
		executorService: executorService,
		pathPrefix:      pathPrefix,
	}
}

type DeployRequest struct {
	CommandID string `json:"command_id"`
}

// Deploy executes the default command for an app using token authentication
// POST /devops/deploy/:app_id
// Header: X-Deploy-Token: <token>
func (h *DeployHandler) Deploy(c *gin.Context) {
	appID := c.Param("app_id")
	token := c.GetHeader("X-Deploy-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-Deploy-Token header"})
		return
	}

	// Verify app exists and token matches
	app, err := h.appService.GetAppByID(appID)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(app.Token), []byte(token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Get command - either from request body or default
	var req DeployRequest
	c.ShouldBindJSON(&req)

	var commandID string
	if req.CommandID != "" {
		// Verify command belongs to this app
		cmd, err := h.appService.GetCommandByID(req.CommandID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}
		if cmd.AppID != appID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "command does not belong to this app"})
			return
		}
		commandID = cmd.ID
	} else {
		// Get default command (first command)
		cmd, err := h.appService.GetDefaultCommandByAppID(appID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no commands configured for this app"})
			return
		}
		commandID = cmd.ID
	}

	// Create execution with system user (user_id = 0 for API calls)
	execution, err := h.executorService.CreateExecution(commandID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Execute in background
	go h.executorService.Execute(execution.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"message":      "deployment started",
		"execution_id": execution.ID,
		"app_id":       appID,
		"app_name":     app.Name,
		"stream_url":   h.pathPrefix + "/api/executions/" + execution.ID + "/stream",
		"status_url":   h.pathPrefix + "/api/executions/" + execution.ID,
	})
}

// DeployStatus gets the status of a deployment
// GET /devops/deploy/:app_id/status/:execution_id
// Header: X-Deploy-Token: <token>
func (h *DeployHandler) DeployStatus(c *gin.Context) {
	appID := c.Param("app_id")
	executionID := c.Param("execution_id")
	token := c.GetHeader("X-Deploy-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-Deploy-Token header"})
		return
	}

	// Verify app exists and token matches
	app, err := h.appService.GetAppByID(appID)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(app.Token), []byte(token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	execution, err := h.executorService.GetExecutionByID(executionID)
	if err != nil {
		if err == services.ErrExecutionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// DeployStream streams the execution output using SSE
// GET /devops/deploy/:app_id/stream/:execution_id
// Header: X-Deploy-Token: <token>
func (h *DeployHandler) DeployStream(c *gin.Context) {
	appID := c.Param("app_id")
	executionID := c.Param("execution_id")
	token := c.GetHeader("X-Deploy-Token")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-Deploy-Token header"})
		return
	}

	// Verify app exists and token matches
	app, err := h.appService.GetAppByID(appID)
	if err != nil {
		if err == services.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(app.Token), []byte(token)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Verify execution exists
	_, err = h.executorService.GetExecutionByID(executionID)
	if err != nil {
		if err == services.ErrExecutionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delegate to stream handler logic
	streamHandler := NewStreamHandler(h.executorService)
	streamHandler.streamExecution(c, executionID)
}
