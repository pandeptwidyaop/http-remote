package services_test

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func setupAuthTestDB(t *testing.T) (*database.DB, *sql.DB, *config.Config) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db := &database.DB{DB: sqlDB}

	// Create tables with 2FA columns, role, and session binding
	_, err = sqlDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			is_admin BOOLEAN DEFAULT 0,
			totp_secret TEXT,
			totp_enabled BOOLEAN DEFAULT FALSE,
			backup_codes TEXT,
			role TEXT DEFAULT 'operator',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ip_address TEXT,
			user_agent_hash TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);

		CREATE TABLE login_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			ip_address TEXT NOT NULL,
			success BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE password_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{
			BcryptCost:      10,
			SessionDuration: "24h",
		},
		Admin: config.AdminConfig{
			Username: "admin",
			Password: "admin123",
		},
	}

	return db, sqlDB, cfg
}

func TestAuthService_CreateUser(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	user, err := authSvc.CreateUser("testuser", "password123", false)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}

	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}

	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}

	if user.IsAdmin {
		t.Error("expected user to not be admin")
	}
}

func TestAuthService_CreateUser_Duplicate(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create first user
	_, err := authSvc.CreateUser("duplicate", "password123", false)
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	// Try to create duplicate
	_, err = authSvc.CreateUser("duplicate", "password123", false)
	if err != services.ErrUserExists {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_GetUserByUsername(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user
	created, _ := authSvc.CreateUser("findme", "password123", false)

	// Get user by username
	user, err := authSvc.GetUserByUsername("findme")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if user.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, user.ID)
	}

	// Test non-existent user
	_, err = authSvc.GetUserByUsername("nonexistent")
	if err != services.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestAuthService_CheckPassword(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	password := "mysecretpassword"
	hash, err := authSvc.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Test correct password
	if !authSvc.CheckPassword(password, hash) {
		t.Error("expected password to match")
	}

	// Test incorrect password
	if authSvc.CheckPassword("wrongpassword", hash) {
		t.Error("expected password to not match")
	}
}

func TestAuthService_Login(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user
	_, _ = authSvc.CreateUser("loginuser", "password123", false)

	// Test successful login
	session, err := authSvc.Login("loginuser", "password123")
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	if session.ID == "" {
		t.Error("expected session ID to be set")
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("expected session to not be expired")
	}

	// Test login with wrong password
	_, err = authSvc.Login("loginuser", "wrongpassword")
	if err != services.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// Test login with non-existent user
	_, err = authSvc.Login("nonexistent", "password123")
	if err != services.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_ValidateSession(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user and login
	user, _ := authSvc.CreateUser("sessionuser", "password123", false)
	session, _ := authSvc.Login("sessionuser", "password123")

	// Validate session
	validatedUser, err := authSvc.ValidateSession(session.ID)
	if err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, validatedUser.ID)
	}

	// Test invalid session
	_, err = authSvc.ValidateSession("invalid-session-id")
	if err != services.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestAuthService_ValidateSession_Expired(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	// Set very short session duration
	cfg.Auth.SessionDuration = "1ms"

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user and login
	_, _ = authSvc.CreateUser("expireuser", "password123", false)
	session, _ := authSvc.Login("expireuser", "password123")

	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired session
	_, err := authSvc.ValidateSession(session.ID)
	if err != services.ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestAuthService_DeleteSession(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user and login
	_, _ = authSvc.CreateUser("logoutuser", "password123", false)
	session, _ := authSvc.Login("logoutuser", "password123")

	// Delete session
	err := authSvc.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify session is deleted
	_, err = authSvc.ValidateSession(session.ID)
	if err != services.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after deletion, got %v", err)
	}
}

func TestAuthService_InvalidateUserSessions(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user and multiple sessions
	user, _ := authSvc.CreateUser("multiuser", "password123", false)
	session1, _ := authSvc.CreateSession(user.ID)
	session2, _ := authSvc.CreateSession(user.ID)

	// Invalidate all sessions
	err := authSvc.InvalidateUserSessions(user.ID)
	if err != nil {
		t.Fatalf("failed to invalidate sessions: %v", err)
	}

	// Verify both sessions are invalidated
	_, err = authSvc.ValidateSession(session1.ID)
	if err != services.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound for session1, got %v", err)
	}

	_, err = authSvc.ValidateSession(session2.ID)
	if err != services.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound for session2, got %v", err)
	}
}

func TestAuthService_CleanExpiredSessions(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Create user with short session duration
	cfg.Auth.SessionDuration = "1ms"
	user, _ := authSvc.CreateUser("cleanuser", "password123", false)
	session, _ := authSvc.CreateSession(user.ID)

	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)

	// Clean expired sessions
	err := authSvc.CleanExpiredSessions()
	if err != nil {
		t.Fatalf("failed to clean expired sessions: %v", err)
	}

	// Verify session is deleted
	_, err = authSvc.ValidateSession(session.ID)
	if err != services.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after cleanup, got %v", err)
	}
}

func TestAuthService_GenerateSecurePassword(t *testing.T) {
	db, sqlDB, cfg := setupAuthTestDB(t)
	defer func() { _ = sqlDB.Close() }()

	authSvc := services.NewAuthService(db, cfg, nil)

	// Generate passwords
	pass1, err := authSvc.GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	pass2, err := authSvc.GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	// Verify length
	if len(pass1) != 16 {
		t.Errorf("expected password length 16, got %d", len(pass1))
	}

	// Verify uniqueness
	if pass1 == pass2 {
		t.Error("expected generated passwords to be different")
	}
}
