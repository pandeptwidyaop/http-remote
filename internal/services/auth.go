package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
)

var (
	// ErrInvalidCredentials indicates incorrect username or password.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrSessionExpired indicates the session has expired.
	ErrSessionExpired = errors.New("session expired")
	// ErrSessionNotFound indicates the session does not exist.
	ErrSessionNotFound = errors.New("session not found")
	// ErrUserNotFound indicates the user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists indicates a user with the same username already exists.
	ErrUserExists = errors.New("user already exists")
	// ErrAccountLocked indicates the account is temporarily locked due to too many failed attempts.
	ErrAccountLocked = errors.New("account temporarily locked")
)

// AuthService handles user authentication and session management.
type AuthService struct {
	db     *database.DB
	cfg    *config.Config
	crypto *CryptoService
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(db *database.DB, cfg *config.Config, crypto *CryptoService) *AuthService {
	return &AuthService{db: db, cfg: cfg, crypto: crypto}
}

// HashPassword hashes a password using bcrypt.
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.Auth.BcryptCost)
	return string(bytes), err
}

// CheckPassword verifies if a password matches the hashed password.
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser creates a new user with a hashed password.
func (s *AuthService) CreateUser(username, password string, isAdmin bool) (*models.User, error) {
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(
		"INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)",
		username, hash, isAdmin,
	)
	if err != nil {
		return nil, ErrUserExists
	}

	id, _ := result.LastInsertId()
	return s.GetUserByID(id)
}

// GetUserByID retrieves a user by their ID.
func (s *AuthService) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	var totpSecret sql.NullString
	var totpEnabled sql.NullBool

	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0), created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &totpSecret, &totpEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	// Decrypt TOTP secret if crypto service is available and secret exists
	if totpSecret.String != "" && s.crypto != nil {
		decrypted, err := s.crypto.Decrypt(totpSecret.String)
		if err != nil {
			// If decryption fails, assume it's a plaintext secret (migration path)
			user.TOTPSecret = totpSecret.String
		} else {
			user.TOTPSecret = decrypted
		}
	} else {
		user.TOTPSecret = totpSecret.String
	}

	user.TOTPEnabled = totpEnabled.Bool
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	var totpSecret sql.NullString
	var totpEnabled sql.NullBool

	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0), created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &totpSecret, &totpEnabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	// Decrypt TOTP secret if crypto service is available and secret exists
	if totpSecret.String != "" && s.crypto != nil {
		decrypted, err := s.crypto.Decrypt(totpSecret.String)
		if err != nil {
			// If decryption fails, assume it's a plaintext secret (migration path)
			user.TOTPSecret = totpSecret.String
		} else {
			user.TOTPSecret = decrypted
		}
	} else {
		user.TOTPSecret = totpSecret.String
	}

	user.TOTPEnabled = totpEnabled.Bool
	return &user, nil
}

// Login authenticates a user and creates a new session.
func (s *AuthService) Login(username, password string) (*models.Session, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Invalidate old sessions for this user (session regeneration)
	s.InvalidateUserSessions(user.ID)

	return s.CreateSession(user.ID)
}

// InvalidateUserSessions removes all sessions for a user
func (s *AuthService) InvalidateUserSessions(userID int64) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// CreateSession creates a new session for a user.
func (s *AuthService) CreateSession(userID int64) (*models.Session, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(s.cfg.Auth.GetSessionDuration())

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)",
		sessionID, userID, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

// ValidateSession validates a session and returns the associated user.
func (s *AuthService) ValidateSession(sessionID string) (*models.User, error) {
	var session models.Session
	err := s.db.QueryRow(
		"SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?",
		sessionID,
	).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}

	return s.GetUserByID(session.UserID)
}

// DeleteSession deletes a session by its ID.
func (s *AuthService) DeleteSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

// CleanExpiredSessions removes all expired sessions from the database.
func (s *AuthService) CleanExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// GenerateSecurePassword generates a random password of the specified length.
func (s *AuthService) GenerateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// EnsureAdminUser ensures an admin user exists, creating one if needed.
func (s *AuthService) EnsureAdminUser() error {
	_, err := s.GetUserByUsername(s.cfg.Admin.Username)
	if err == ErrUserNotFound {
		password := s.cfg.Admin.Password

		// If default password is still "changeme", generate a random one
		if password == "changeme" {
			generated, err := s.GenerateSecurePassword(16)
			if err != nil {
				return err
			}
			password = generated
			log.Printf("⚠️  WARNING: Default admin password detected!")
			log.Printf("📝 Generated secure admin password: %s", password)
			log.Printf("🔒 Please save this password and change it after first login")
			log.Printf("   Username: %s", s.cfg.Admin.Username)
		}

		_, err = s.CreateUser(s.cfg.Admin.Username, password, true)
		return err
	}
	return nil
}

// SetTOTPSecret stores the TOTP secret for a user (encrypted if crypto service available)
func (s *AuthService) SetTOTPSecret(userID int64, secret string) error {
	// Encrypt the secret if crypto service is available
	storedSecret := secret
	if s.crypto != nil {
		encrypted, err := s.crypto.Encrypt(secret)
		if err != nil {
			return err
		}
		storedSecret = encrypted
	}

	_, err := s.db.Exec("UPDATE users SET totp_secret = ? WHERE id = ?", storedSecret, userID)
	return err
}

// EnableTOTP enables 2FA for a user
func (s *AuthService) EnableTOTP(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET totp_enabled = 1 WHERE id = ?", userID)
	return err
}

// DisableTOTP disables 2FA for a user and clears the secret
func (s *AuthService) DisableTOTP(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET totp_enabled = 0, totp_secret = NULL WHERE id = ?", userID)
	return err
}

// ChangePassword changes the password for a user after verifying the old password
func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	// Get user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !s.CheckPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	_, err = s.db.Exec("UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", hashedPassword, userID)
	return err
}

// RecordLoginAttempt records a login attempt (success or failure)
func (s *AuthService) RecordLoginAttempt(username, ipAddress string, success bool) error {
	_, err := s.db.Exec(
		"INSERT INTO login_attempts (username, ip_address, success) VALUES (?, ?, ?)",
		username, ipAddress, success,
	)
	return err
}

// IsAccountLocked checks if an account is locked due to too many failed login attempts
func (s *AuthService) IsAccountLocked(username string) (bool, time.Duration) {
	maxAttempts := s.cfg.Security.GetMaxLoginAttempts()
	lockoutDuration := s.cfg.Security.GetLockoutDuration()
	windowStart := time.Now().Add(-lockoutDuration)

	// Count failed attempts within the lockout window
	var failedCount int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM login_attempts
		WHERE username = ? AND success = 0 AND created_at > ?
	`, username, windowStart).Scan(&failedCount)

	if err != nil {
		return false, 0
	}

	if failedCount >= maxAttempts {
		// Get the time of the last failed attempt to calculate remaining lockout time
		var lastAttemptTime time.Time
		err := s.db.QueryRow(`
			SELECT created_at FROM login_attempts
			WHERE username = ? AND success = 0
			ORDER BY created_at DESC LIMIT 1
		`, username).Scan(&lastAttemptTime)

		if err != nil {
			return true, lockoutDuration
		}

		unlockTime := lastAttemptTime.Add(lockoutDuration)
		remaining := time.Until(unlockTime)
		if remaining > 0 {
			return true, remaining
		}
	}

	return false, 0
}

// ClearLoginAttempts clears failed login attempts for a user (called after successful login)
func (s *AuthService) ClearLoginAttempts(username string) error {
	_, err := s.db.Exec("DELETE FROM login_attempts WHERE username = ?", username)
	return err
}

// CleanOldLoginAttempts removes login attempts older than the lockout duration
func (s *AuthService) CleanOldLoginAttempts() error {
	lockoutDuration := s.cfg.Security.GetLockoutDuration()
	cutoff := time.Now().Add(-lockoutDuration * 2) // Keep 2x lockout duration for safety
	_, err := s.db.Exec("DELETE FROM login_attempts WHERE created_at < ?", cutoff)
	return err
}

// GetRecentFailedAttempts returns the count of recent failed login attempts
func (s *AuthService) GetRecentFailedAttempts(username string) int {
	lockoutDuration := s.cfg.Security.GetLockoutDuration()
	windowStart := time.Now().Add(-lockoutDuration)

	var count int
	_ = s.db.QueryRow(`
		SELECT COUNT(*) FROM login_attempts
		WHERE username = ? AND success = 0 AND created_at > ?
	`, username, windowStart).Scan(&count)

	return count
}
