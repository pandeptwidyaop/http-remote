package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type CommandHandler struct {
	appService      *services.AppService
	executorService *services.ExecutorService
	auditService    *services.AuditService
	pathPrefix      string
}

func NewCommandHandler(appService *services.AppService, executorService *services.ExecutorService, auditService *services.AuditService, pathPrefix string) *CommandHandler {
	return &CommandHandler{
		appService:      appService,
		executorService: executorService,
		auditService:    auditService,
		pathPrefix:      pathPrefix,
	}
}

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

func (h *CommandHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Get command info before deleting for audit log
	cmd, _ := h.appService.GetCommandByID(id)

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

	user, _ := c.Get(middleware.UserContextKey)
	u := user.(*models.User)

	execution, err := h.executorService.CreateExecution(cmd.ID, u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go h.executorService.Execute(execution.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"execution_id": execution.ID,
		"stream_url":   h.pathPrefix + "/api/executions/" + execution.ID + "/stream",
	})
}

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
