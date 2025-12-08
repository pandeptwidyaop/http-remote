package handlers

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/service"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/upgrade"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

// SystemHandler handles system management endpoints.
type SystemHandler struct {
	auditService *services.AuditService
}

// NewSystemHandler creates a new SystemHandler instance.
func NewSystemHandler(auditService *services.AuditService) *SystemHandler {
	return &SystemHandler{
		auditService: auditService,
	}
}

// SystemStatus represents the system status response.
type SystemStatus struct {
	Platform       string `json:"platform"`
	Arch           string `json:"arch"`
	IsLinux        bool   `json:"is_linux"`
	IsSystemd      bool   `json:"is_systemd"`
	IsService      bool   `json:"is_service"`
	ServiceStatus  string `json:"service_status"`
	CanUpgrade     bool   `json:"can_upgrade"`
	CanRestart     bool   `json:"can_restart"`
	CurrentVersion string `json:"current_version"`
}

// Status returns the current system status.
// GET /api/system/status
func (h *SystemHandler) Status(c *gin.Context) {
	status := SystemStatus{
		Platform:       runtime.GOOS,
		Arch:           runtime.GOARCH,
		IsLinux:        service.IsLinux(),
		IsSystemd:      service.IsSystemdAvailable(),
		IsService:      service.IsRunningAsService(),
		CanUpgrade:     service.IsLinux(), // Upgrade only supported on Linux
		CanRestart:     service.IsLinux() && service.IsSystemdAvailable() && service.IsRunningAsService(),
		CurrentVersion: version.Version,
	}

	// Get service status if on Linux with systemd
	if status.IsSystemd {
		svcStatus, err := service.Status()
		if err == nil {
			if svcStatus.IsRunning {
				status.ServiceStatus = "running"
			} else if svcStatus.IsInstalled {
				status.ServiceStatus = "stopped"
			} else {
				status.ServiceStatus = "not_installed"
			}
		} else {
			status.ServiceStatus = "unknown"
		}
	} else {
		status.ServiceStatus = "not_available"
	}

	c.JSON(http.StatusOK, status)
}

// Upgrade performs the system upgrade.
// POST /api/system/upgrade
func (h *SystemHandler) Upgrade(c *gin.Context) {
	// Get authenticated user for audit logging
	user, _ := c.Get(middleware.UserContextKey)
	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if running on Linux
	if !service.IsLinux() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Upgrade is only supported on Linux",
			"message": "Please upgrade manually using: ./http-remote upgrade",
		})
		return
	}

	// Parse request body
	var req struct {
		Force bool `json:"force"`
	}
	_ = c.ShouldBindJSON(&req)

	// Audit log the upgrade attempt
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       "system_upgrade_started",
		ResourceType: "system",
		Details:      map[string]interface{}{"message": "User initiated system upgrade"},
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	// Perform upgrade with progress tracking
	var lastProgress upgrade.Progress
	release, err := upgrade.RunWithProgress(req.Force, func(progress upgrade.Progress) {
		lastProgress = progress
	})

	if err != nil {
		// Audit log the failure
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "system_upgrade_failed",
			ResourceType: "system",
			Details:      map[string]interface{}{"error": err.Error()},
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    err.Error(),
			"progress": lastProgress,
		})
		return
	}

	// Audit log success
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       "system_upgrade_completed",
		ResourceType: "system",
		Details:      map[string]interface{}{"new_version": release.TagName},
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"new_version":  release.TagName,
		"need_restart": true,
		"message":      lastProgress.Message,
	})
}

// Restart restarts the systemd service.
// POST /api/system/restart
func (h *SystemHandler) Restart(c *gin.Context) {
	// Get authenticated user for audit logging
	user, _ := c.Get(middleware.UserContextKey)
	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check requirements
	if !service.IsLinux() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Service restart is only supported on Linux",
			"message": "Please restart the service manually",
		})
		return
	}

	if !service.IsSystemdAvailable() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "systemd is not available",
			"message": "Please restart the service manually",
		})
		return
	}

	if !service.IsRunningAsService() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Not running as a systemd service",
			"message": "The application must be running as a systemd service to use this feature",
		})
		return
	}

	// Audit log the restart
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       "system_restart",
		ResourceType: "system",
		Details:      map[string]interface{}{"message": "User initiated service restart"},
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	// Send response before restarting
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service is restarting...",
	})

	// Restart in a goroutine to allow response to be sent
	go func() {
		_ = service.Restart()
	}()
}

// ListRollbackVersions returns available backup versions for rollback.
// GET /api/system/rollback-versions
func (h *SystemHandler) ListRollbackVersions(c *gin.Context) {
	backups, err := upgrade.ListBackups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backups":         backups,
		"current_version": version.Version,
	})
}

// Rollback restores a previous version from backup.
// POST /api/system/rollback
func (h *SystemHandler) Rollback(c *gin.Context) {
	// Get authenticated user for audit logging
	user, _ := c.Get(middleware.UserContextKey)
	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if running on Linux
	if !service.IsLinux() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Rollback is only supported on Linux",
			"message": "Please rollback manually",
		})
		return
	}

	var req struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup_path is required"})
		return
	}

	// Verify the backup exists in our list (security check)
	backups, err := upgrade.ListBackups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups"})
		return
	}

	found := false
	var backupVersion string
	for _, b := range backups {
		if b.Path == req.BackupPath {
			found = true
			backupVersion = b.Version
			break
		}
	}

	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup path"})
		return
	}

	// Audit log the rollback attempt
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       "system_rollback_started",
		ResourceType: "system",
		Details:      map[string]interface{}{"target_version": backupVersion},
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	// Perform rollback
	if err := upgrade.Rollback(req.BackupPath); err != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "system_rollback_failed",
			ResourceType: "system",
			Details:      map[string]interface{}{"error": err.Error()},
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit log success
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       "system_rollback_completed",
		ResourceType: "system",
		Details:      map[string]interface{}{"rolled_back_to": backupVersion},
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"version":      backupVersion,
		"need_restart": true,
		"message":      "Rolled back to " + backupVersion + ". Restart required.",
	})
}
