package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// CommandHandler handles HTTP requests for command operations.
type CommandHandler struct {
	appService      *services.AppService
	executorService *services.ExecutorService
	auditService    *services.AuditService
	pathPrefix      string
}

// NewCommandHandler creates a new CommandHandler instance.
func NewCommandHandler(appService *services.AppService, executorService *services.ExecutorService, auditService *services.AuditService, pathPrefix string) *CommandHandler {
	return &CommandHandler{
		appService:      appService,
		executorService: executorService,
		auditService:    auditService,
		pathPrefix:      pathPrefix,
	}
}

// Get retrieves a command by ID.
func (h *CommandHandler) Get(c *gin.Context) {
	id := c.Param("id")

	cmd, err := h.appService.GetCommandByID(id)
	if err != nil {
		if err == services.ErrCommandNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cmd)
}

// Update updates a command.
func (h *CommandHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd, err := h.appService.UpdateCommand(id, &req)
	if err != nil {
		if err == services.ErrCommandNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		h.auditService.LogCommandUpdate(u, cmd.ID, cmd.Name, c.ClientIP(), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, cmd)
}

// Delete deletes a command.
func (h *CommandHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Get command info before deleting for audit log
	cmd, err := h.appService.GetCommandByID(id)
	if err != nil {
		// If we can't get command, just ignore for audit purposes
		cmd = nil
	}

	if err := h.appService.DeleteCommand(id); err != nil {
		if err == services.ErrCommandNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok && cmd != nil {
		h.auditService.LogCommandDelete(u, cmd.ID, cmd.Name, c.ClientIP(), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, gin.H{"message": "command deleted"})
}

// Execute executes a command asynchronously.
func (h *CommandHandler) Execute(c *gin.Context) {
	id := c.Param("id")

	cmd, err := h.appService.GetCommandByID(id)
	if err != nil {
		if err == services.ErrCommandNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	execution, err := h.executorService.CreateExecution(cmd.ID, u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get app for audit logging
	app, _ := h.appService.GetAppByID(cmd.AppID)
	appName := ""
	if app != nil {
		appName = app.Name
	}

	// Audit log command execution
	h.auditService.LogCommandExecute(u.Username, &u.ID, cmd.ID, cmd.Name, appName, c.ClientIP(), c.GetHeader("User-Agent"))

	go func() {
		_ = h.executorService.Execute(execution.ID)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"execution_id": execution.ID,
		"stream_url":   h.pathPrefix + "/api/executions/" + execution.ID + "/stream",
	})
}

// ListExecutions lists all executions.
func (h *CommandHandler) ListExecutions(c *gin.Context) {
	limit := 50
	offset := 0

	executions, err := h.executorService.GetExecutions(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetExecution retrieves an execution by ID.
func (h *CommandHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")

	execution, err := h.executorService.GetExecutionByID(id)
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
