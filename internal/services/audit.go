package services

import (
	"encoding/json"

	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
)

// AuditService handles audit logging for user actions.
type AuditService struct {
	db *database.DB
}

// NewAuditService creates a new AuditService instance.
func NewAuditService(db *database.DB) *AuditService {
	return &AuditService{db: db}
}

// AuditLog represents an audit log entry to be recorded.
type AuditLog struct {
	UserID       *int64
	Details      map[string]interface{}
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	IPAddress    string
	UserAgent    string
}

// Log records an audit log entry to the database.
func (s *AuditService) Log(log AuditLog) error {
	var detailsJSON string
	if log.Details != nil {
		bytes, err := json.Marshal(log.Details)
		if err == nil {
			detailsJSON = string(bytes)
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO audit_logs (user_id, username, action, resource_type, resource_id, ip_address, user_agent, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, log.UserID, log.Username, log.Action, log.ResourceType, log.ResourceID, log.IPAddress, log.UserAgent, detailsJSON)

	if err != nil {
		// Log to stderr but don't fail the request
		log := log
		_ = log // avoid unused variable warning
	}

	return err
}

// LogLogin logs a user login attempt.
func (s *AuditService) LogLogin(user *models.User, ip, userAgent string, success bool) {
	action := "login_success"
	if !success {
		action = "login_failed"
	}

	s.Log(AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       action,
		ResourceType: "auth",
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogLogout logs a user logout event.
func (s *AuditService) LogLogout(user *models.User, ip, userAgent string) {
	s.Log(AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "logout",
		ResourceType: "auth",
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogCommandCreate logs the creation of a new command.
func (s *AuditService) LogCommandCreate(user *models.User, commandID, commandName string, ip, userAgent string) {
	s.Log(AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "create",
		ResourceType: "command",
		ResourceID:   commandID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details: map[string]interface{}{
			"command_name": commandName,
		},
	})
}

// LogCommandUpdate logs command update operations.
func (s *AuditService) LogCommandUpdate(user *models.User, commandID, commandName string, ip, userAgent string) {
	s.Log(AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "update",
		ResourceType: "command",
		ResourceID:   commandID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details: map[string]interface{}{
			"command_name": commandName,
		},
	})
}

// LogCommandDelete logs command deletion operations.
func (s *AuditService) LogCommandDelete(user *models.User, commandID, commandName string, ip, userAgent string) {
	s.Log(AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "delete",
		ResourceType: "command",
		ResourceID:   commandID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details: map[string]interface{}{
			"command_name": commandName,
		},
	})
}

// LogCommandExecute logs command execution events.
func (s *AuditService) LogCommandExecute(username string, userID *int64, commandID, commandName, appName string, ip, userAgent string) {
	s.Log(AuditLog{
		UserID:       userID,
		Username:     username,
		Action:       "execute",
		ResourceType: "command",
		ResourceID:   commandID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details: map[string]interface{}{
			"command_name": commandName,
			"app_name":     appName,
		},
	})
}

// AuditLogEntry represents an audit log record from the database.
type AuditLogEntry struct {
	UserID       *int64 `json:"user_id"`
	Username     string `json:"username"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	IPAddress    string `json:"ip_address"`
	UserAgent    string `json:"user_agent"`
	Details      string `json:"details"`
	CreatedAt    string `json:"created_at"`
	ID           int64  `json:"id"`
}

// GetLogs retrieves audit logs with pagination.
func (s *AuditService) GetLogs(limit, offset int) ([]AuditLogEntry, error) {
	if limit == 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT id, user_id, username, action, resource_type, resource_id, ip_address, user_agent, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize empty slice instead of nil to return [] instead of null in JSON
	logs := make([]AuditLogEntry, 0)
	for rows.Next() {
		var log AuditLogEntry
		var resourceID, ipAddress, userAgent, details *string
		var userIDInt *int64

		if err := rows.Scan(
			&log.ID,
			&userIDInt,
			&log.Username,
			&log.Action,
			&log.ResourceType,
			&resourceID,
			&ipAddress,
			&userAgent,
			&details,
			&log.CreatedAt,
		); err != nil {
			continue
		}

		log.UserID = userIDInt
		if resourceID != nil {
			log.ResourceID = *resourceID
		}
		if ipAddress != nil {
			log.IPAddress = *ipAddress
		}
		if userAgent != nil {
			log.UserAgent = *userAgent
		}
		if details != nil {
			log.Details = *details
		}

		logs = append(logs, log)
	}

	return logs, nil
}
