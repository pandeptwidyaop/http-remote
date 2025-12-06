package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// BackupHandler handles backup and restore operations
type BackupHandler struct {
	appService   *services.AppService
	auditService *services.AuditService
}

// NewBackupHandler creates a new BackupHandler instance
func NewBackupHandler(appService *services.AppService, auditService *services.AuditService) *BackupHandler {
	return &BackupHandler{
		appService:   appService,
		auditService: auditService,
	}
}

// Export exports all apps and their commands as JSON
func (h *BackupHandler) Export(c *gin.Context) {
	apps, err := h.appService.GetAllApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get apps"})
		return
	}

	backupApps := make([]models.AppBackup, 0, len(apps))
	for _, app := range apps {
		commands, err := h.appService.GetCommandsByAppID(app.ID)
		if err != nil {
			continue
		}

		cmdBackups := make([]models.CommandBackup, 0, len(commands))
		for _, cmd := range commands {
			cmdBackups = append(cmdBackups, models.CommandBackup{
				Name:           cmd.Name,
				Description:    cmd.Description,
				Command:        cmd.Command,
				TimeoutSeconds: cmd.TimeoutSeconds,
			})
		}

		backupApps = append(backupApps, models.AppBackup{
			Name:        app.Name,
			Description: app.Description,
			WorkingDir:  app.WorkingDir,
			Commands:    cmdBackups,
		})
	}

	backup := models.BackupData{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Apps:       backupApps,
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "export_backup",
			ResourceType: "backup",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"app_count": len(backupApps)},
		})
	}

	c.Header("Content-Disposition", "attachment; filename=backup.json")
	c.JSON(http.StatusOK, backup)
}

// Import imports apps and commands from a backup JSON
func (h *BackupHandler) Import(c *gin.Context) {
	var backup models.BackupData
	if err := c.ShouldBindJSON(&backup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup data: " + err.Error()})
		return
	}

	if len(backup.Apps) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no apps found in backup"})
		return
	}

	imported := 0
	skipped := 0
	errors := []string{}

	for _, appBackup := range backup.Apps {
		// Check if app already exists
		existing, _ := h.appService.GetAppByName(appBackup.Name)
		if existing != nil {
			skipped++
			continue
		}

		// Create app
		app, err := h.appService.CreateApp(&models.CreateAppRequest{
			Name:        appBackup.Name,
			Description: appBackup.Description,
			WorkingDir:  appBackup.WorkingDir,
		})
		if err != nil {
			errors = append(errors, "failed to create app '"+appBackup.Name+"': "+err.Error())
			continue
		}

		// Create commands for this app
		for _, cmdBackup := range appBackup.Commands {
			_, err := h.appService.CreateCommand(app.ID, &models.CreateCommandRequest{
				Name:           cmdBackup.Name,
				Description:    cmdBackup.Description,
				Command:        cmdBackup.Command,
				TimeoutSeconds: cmdBackup.TimeoutSeconds,
			})
			if err != nil {
				errors = append(errors, "failed to create command '"+cmdBackup.Name+"' for app '"+appBackup.Name+"'")
			}
		}

		imported++
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "import_backup",
			ResourceType: "backup",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"imported": imported,
				"skipped":  skipped,
				"errors":   len(errors),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "import completed",
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}

// ExportApp exports a single app and its commands as JSON
func (h *BackupHandler) ExportApp(c *gin.Context) {
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

	commands, err := h.appService.GetCommandsByAppID(app.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get commands"})
		return
	}

	cmdBackups := make([]models.CommandBackup, 0, len(commands))
	for _, cmd := range commands {
		cmdBackups = append(cmdBackups, models.CommandBackup{
			Name:           cmd.Name,
			Description:    cmd.Description,
			Command:        cmd.Command,
			TimeoutSeconds: cmd.TimeoutSeconds,
		})
	}

	appBackup := models.AppBackup{
		Name:        app.Name,
		Description: app.Description,
		WorkingDir:  app.WorkingDir,
		Commands:    cmdBackups,
	}

	// Audit log
	user, _ := c.Get(middleware.UserContextKey)
	if u, ok := user.(*models.User); ok {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &u.ID,
			Username:     u.Username,
			Action:       "export_app",
			ResourceType: "app",
			ResourceID:   app.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details:      map[string]interface{}{"app_name": app.Name},
		})
	}

	c.Header("Content-Disposition", "attachment; filename="+app.Name+"-backup.json")
	c.JSON(http.StatusOK, appBackup)
}
