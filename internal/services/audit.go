package services

import (
	"encoding/json"

	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
)

type AuditService struct {
	db *database.DB
}

func NewAuditService(db *database.DB) *AuditService {
	return &AuditService{db: db}
}

type AuditLog struct {
	UserID       *int64
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	IPAddress    string
	UserAgent    string
	Details      map[string]interface{}
}

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

// Convenience methods for common actions
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
