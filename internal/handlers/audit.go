package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type AuditHandler struct {
	auditService *services.AuditService
	pathPrefix   string
}

func NewAuditHandler(auditService *services.AuditService, pathPrefix string) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
		pathPrefix:   pathPrefix,
	}
}

// List returns audit logs (API)
func (h *AuditHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.auditService.GetLogs(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// Page renders the audit logs web page
func (h *AuditHandler) Page(c *gin.Context) {
	c.HTML(http.StatusOK, "audit_logs.html", gin.H{
		"PathPrefix": h.pathPrefix,
	})
}
