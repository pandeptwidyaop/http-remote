package models

import "time"

// User represents a user account.
type User struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	TOTPSecret   string    `json:"-"`            // TOTP secret for 2FA (encrypted)
	BackupCodes  string    `json:"-"`            // Backup codes for 2FA recovery (encrypted JSON array)
	ID           int64     `json:"id"`
	TOTPEnabled  bool      `json:"totp_enabled"` // Whether 2FA is enabled
	IsAdmin      bool      `json:"is_admin"`
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
