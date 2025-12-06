package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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
	// ErrPasswordReused indicates the password was used recently
	ErrPasswordReused = errors.New("password was used recently, please choose a different password")
)

// Password History constants
const (
	PasswordHistoryLimit = 5 // Number of previous passwords to remember
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
	role := models.RoleOperator
	if isAdmin {
		role = models.RoleAdmin
	}
	return s.CreateUserWithRole(username, password, role)
}

// CreateUserWithRole creates a new user with a specific role.
func (s *AuthService) CreateUserWithRole(username, password string, role models.UserRole) (*models.User, error) {
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	isAdmin := role == models.RoleAdmin

	result, err := s.db.Exec(
		"INSERT INTO users (username, password_hash, is_admin, role) VALUES (?, ?, ?, ?)",
		username, hash, isAdmin, string(role),
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
	var backupCodes sql.NullString
	var role sql.NullString

	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0), COALESCE(backup_codes, ''), COALESCE(role, 'operator'), created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &totpSecret, &totpEnabled, &backupCodes, &role, &user.CreatedAt, &user.UpdatedAt)

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
	user.BackupCodes = backupCodes.String
	user.Role = models.UserRole(role.String)
	// For backward compatibility
	if user.Role == "" {
		if user.IsAdmin {
			user.Role = models.RoleAdmin
		} else {
			user.Role = models.RoleOperator
		}
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	var totpSecret sql.NullString
	var totpEnabled sql.NullBool
	var backupCodes sql.NullString
	var role sql.NullString

	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, COALESCE(totp_secret, ''), COALESCE(totp_enabled, 0), COALESCE(backup_codes, ''), COALESCE(role, 'operator'), created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &totpSecret, &totpEnabled, &backupCodes, &role, &user.CreatedAt, &user.UpdatedAt)

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
	user.BackupCodes = backupCodes.String
	user.Role = models.UserRole(role.String)
	// For backward compatibility
	if user.Role == "" {
		if user.IsAdmin {
			user.Role = models.RoleAdmin
		} else {
			user.Role = models.RoleOperator
		}
	}
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

// HashUserAgent creates a SHA-256 hash of the User-Agent string
func (s *AuthService) HashUserAgent(userAgent string) string {
	hash := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(hash[:])
}

// CreateSession creates a new session for a user.
func (s *AuthService) CreateSession(userID int64) (*models.Session, error) {
	return s.CreateSessionWithBinding(userID, "", "")
}

// CreateSessionWithBinding creates a new session with IP and User-Agent binding
func (s *AuthService) CreateSessionWithBinding(userID int64, ipAddress, userAgent string) (*models.Session, error) {
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(s.cfg.Auth.GetSessionDuration())
	userAgentHash := s.HashUserAgent(userAgent)

	_, err := s.db.Exec(
		"INSERT INTO sessions (id, user_id, expires_at, ip_address, user_agent_hash) VALUES (?, ?, ?, ?, ?)",
		sessionID, userID, expiresAt, ipAddress, userAgentHash,
	)
	if err != nil {
		return nil, err
	}

	return &models.Session{
		ID:            sessionID,
		UserID:        userID,
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
		IPAddress:     ipAddress,
		UserAgentHash: userAgentHash,
	}, nil
}

// ValidateSession validates a session and returns the associated user.
func (s *AuthService) ValidateSession(sessionID string) (*models.User, error) {
	return s.ValidateSessionWithBinding(sessionID, "", "")
}

// ValidateSessionWithBinding validates a session with IP/User-Agent binding check
func (s *AuthService) ValidateSessionWithBinding(sessionID, ipAddress, userAgent string) (*models.User, error) {
	var session models.Session
	var ipAddr, uaHash sql.NullString
	err := s.db.QueryRow(
		"SELECT id, user_id, expires_at, created_at, ip_address, user_agent_hash FROM sessions WHERE id = ?",
		sessionID,
	).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &ipAddr, &uaHash)

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

	session.IPAddress = ipAddr.String
	session.UserAgentHash = uaHash.String

	// Validate session binding if IP/User-Agent were stored
	if ipAddress != "" && session.IPAddress != "" && session.IPAddress != ipAddress {
		// IP address mismatch - possible session hijacking
		log.Printf("Session binding mismatch: IP changed from %s to %s for session %s",
			session.IPAddress, ipAddress, sessionID)
		// We log but don't invalidate - IP can change legitimately (mobile networks, VPNs)
	}

	if userAgent != "" && session.UserAgentHash != "" {
		currentHash := s.HashUserAgent(userAgent)
		if session.UserAgentHash != currentHash {
			// User-Agent mismatch - likely session hijacking
			log.Printf("Session binding mismatch: User-Agent changed for session %s", sessionID)
			s.DeleteSession(sessionID)
			return nil, ErrSessionNotFound
		}
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

	// Check if new password is same as old
	if s.CheckPassword(newPassword, user.PasswordHash) {
		return ErrPasswordReused
	}

	// Check password history
	inHistory, err := s.IsPasswordInHistory(userID, newPassword)
	if err != nil {
		return err
	}
	if inHistory {
		return ErrPasswordReused
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Add current password to history before changing
	_ = s.AddPasswordToHistory(userID, user.PasswordHash)

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

// SetBackupCodes stores encrypted backup codes for a user
func (s *AuthService) SetBackupCodes(userID int64, codes []string) error {
	// Hash each code before storing (one-way)
	hashedCodes := make([]string, len(codes))
	for i, code := range codes {
		hash := sha256.Sum256([]byte(code))
		hashedCodes[i] = hex.EncodeToString(hash[:])
	}

	// Join hashed codes with comma
	codesStr := ""
	for i, h := range hashedCodes {
		if i > 0 {
			codesStr += ","
		}
		codesStr += h
	}

	// Encrypt the hashed codes
	storedCodes := codesStr
	if s.crypto != nil {
		encrypted, err := s.crypto.Encrypt(codesStr)
		if err != nil {
			return err
		}
		storedCodes = encrypted
	}

	_, err := s.db.Exec("UPDATE users SET backup_codes = ? WHERE id = ?", storedCodes, userID)
	return err
}

// GetBackupCodesCount returns the number of remaining backup codes
func (s *AuthService) GetBackupCodesCount(userID int64) (int, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return 0, err
	}

	if user.BackupCodes == "" {
		return 0, nil
	}

	// Decrypt codes
	codesStr := user.BackupCodes
	if s.crypto != nil {
		decrypted, err := s.crypto.Decrypt(user.BackupCodes)
		if err != nil {
			// Try using as-is (migration path)
			codesStr = user.BackupCodes
		} else {
			codesStr = decrypted
		}
	}

	if codesStr == "" {
		return 0, nil
	}

	// Count non-empty codes (used codes are removed)
	codes := splitNonEmpty(codesStr, ",")
	return len(codes), nil
}

// ValidateBackupCode validates and consumes a backup code
func (s *AuthService) ValidateBackupCode(userID int64, code string) (bool, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	if user.BackupCodes == "" {
		return false, nil
	}

	// Decrypt codes
	codesStr := user.BackupCodes
	if s.crypto != nil {
		decrypted, err := s.crypto.Decrypt(user.BackupCodes)
		if err != nil {
			// Try using as-is (migration path)
			codesStr = user.BackupCodes
		} else {
			codesStr = decrypted
		}
	}

	// Hash the input code
	inputHash := sha256.Sum256([]byte(code))
	inputHashStr := hex.EncodeToString(inputHash[:])

	// Check if code matches any stored hash
	codes := splitNonEmpty(codesStr, ",")
	foundIdx := -1
	for i, storedHash := range codes {
		if storedHash == inputHashStr {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return false, nil
	}

	// Remove used code
	newCodes := make([]string, 0, len(codes)-1)
	for i, c := range codes {
		if i != foundIdx {
			newCodes = append(newCodes, c)
		}
	}

	// Save updated codes
	newCodesStr := ""
	for i, c := range newCodes {
		if i > 0 {
			newCodesStr += ","
		}
		newCodesStr += c
	}

	storedCodes := newCodesStr
	if s.crypto != nil && newCodesStr != "" {
		encrypted, err := s.crypto.Encrypt(newCodesStr)
		if err != nil {
			return false, err
		}
		storedCodes = encrypted
	}

	_, err = s.db.Exec("UPDATE users SET backup_codes = ? WHERE id = ?", storedCodes, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}

// splitNonEmpty splits a string and returns non-empty parts
func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || (i < len(s) && string(s[i]) == sep) {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	return parts
}

// AddPasswordToHistory adds a password hash to the user's password history
func (s *AuthService) AddPasswordToHistory(userID int64, passwordHash string) error {
	_, err := s.db.Exec(
		"INSERT INTO password_history (user_id, password_hash) VALUES (?, ?)",
		userID, passwordHash,
	)
	if err != nil {
		return err
	}

	// Clean up old entries beyond the limit
	_, err = s.db.Exec(`
		DELETE FROM password_history
		WHERE user_id = ?
		AND id NOT IN (
			SELECT id FROM password_history
			WHERE user_id = ?
			ORDER BY created_at DESC
			LIMIT ?
		)
	`, userID, userID, PasswordHistoryLimit)

	return err
}

// IsPasswordInHistory checks if a password matches any in the user's history
func (s *AuthService) IsPasswordInHistory(userID int64, password string) (bool, error) {
	rows, err := s.db.Query(`
		SELECT password_hash FROM password_history
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, PasswordHistoryLimit)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return false, err
		}
		if s.CheckPassword(password, hash) {
			return true, nil
		}
	}

	return false, nil
}

// GetAllUsers retrieves all users with pagination.
func (s *AuthService) GetAllUsers(limit, offset int) ([]*models.User, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(
		"SELECT id, username, is_admin, COALESCE(role, 'operator'), COALESCE(totp_enabled, 0), created_at, updated_at FROM users ORDER BY id ASC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var role sql.NullString
		var totpEnabled sql.NullBool

		err := rows.Scan(&user.ID, &user.Username, &user.IsAdmin, &role, &totpEnabled, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}

		user.Role = models.UserRole(role.String)
		user.TOTPEnabled = totpEnabled.Bool
		// For backward compatibility
		if user.Role == "" {
			if user.IsAdmin {
				user.Role = models.RoleAdmin
			} else {
				user.Role = models.RoleOperator
			}
		}
		users = append(users, &user)
	}

	return users, total, nil
}

// UpdateUser updates a user's information.
func (s *AuthService) UpdateUser(userID int64, username string, role models.UserRole) error {
	isAdmin := role == models.RoleAdmin

	_, err := s.db.Exec(
		"UPDATE users SET username = ?, role = ?, is_admin = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		username, string(role), isAdmin, userID,
	)
	return err
}

// UpdateUserPassword updates a user's password (admin action, no old password required).
func (s *AuthService) UpdateUserPassword(userID int64, newPassword string) error {
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", hashedPassword, userID)
	return err
}

// DeleteUser deletes a user by ID.
func (s *AuthService) DeleteUser(userID int64) error {
	// First delete all sessions for this user
	_, err := s.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return err
	}

	// Then delete the user
	result, err := s.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// CountAdminUsers counts the number of admin users.
func (s *AuthService) CountAdminUsers() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' OR is_admin = 1").Scan(&count)
	return count, err
}
