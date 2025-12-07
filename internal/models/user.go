package models

import "time"

// UserRole represents the role of a user
type UserRole string

// User role constants defining different permission levels.
const (
	RoleAdmin    UserRole = "admin"    // Full access to everything
	RoleOperator UserRole = "operator" // Can execute commands, manage apps, but cannot manage users
	RoleViewer   UserRole = "viewer"   // Read-only access
)

// User represents a user account.
type User struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	TOTPSecret   string    `json:"-"`    // TOTP secret for 2FA (encrypted)
	BackupCodes  string    `json:"-"`    // Backup codes for 2FA recovery (encrypted JSON array)
	Role         UserRole  `json:"role"` // User role (admin, operator, viewer)
	ID           int64     `json:"id"`
	TOTPEnabled  bool      `json:"totp_enabled"` // Whether 2FA is enabled
	IsAdmin      bool      `json:"is_admin"`     // Deprecated: use Role instead
}

// IsRole checks if user has the specified role
func (u *User) IsRole(role UserRole) bool {
	return u.Role == role
}

// HasPermission checks if user has permission based on role hierarchy
// admin > operator > viewer
func (u *User) HasPermission(requiredRole UserRole) bool {
	switch u.Role {
	case RoleAdmin:
		return true
	case RoleOperator:
		return requiredRole == RoleOperator || requiredRole == RoleViewer
	case RoleViewer:
		return requiredRole == RoleViewer
	default:
		// For backward compatibility, check IsAdmin
		if u.IsAdmin {
			return true
		}
		return false
	}
}

// CanManageUsers returns true if user can manage other users
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin || u.IsAdmin
}

// CanExecuteCommands returns true if user can execute commands
func (u *User) CanExecuteCommands() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator || u.IsAdmin
}

// CanManageApps returns true if user can manage apps
func (u *User) CanManageApps() bool {
	return u.Role == RoleAdmin || u.Role == RoleOperator || u.IsAdmin
}

// Session represents a user session.
type Session struct {
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	ID            string    `json:"id"`
	IPAddress     string    `json:"ip_address"`      // IP address when session was created
	UserAgentHash string    `json:"user_agent_hash"` // Hash of User-Agent header
	UserID        int64     `json:"user_id"`
}
