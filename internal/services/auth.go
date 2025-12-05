package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionNotFound    = errors.New("session not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
)

type AuthService struct {
	db  *database.DB
	cfg *config.Config
}

func NewAuthService(db *database.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.Auth.BcryptCost)
	return string(bytes), err
}

func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

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

func (s *AuthService) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, is_admin, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

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

func (s *AuthService) DeleteSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

func (s *AuthService) CleanExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// GenerateSecurePassword generates a random password
func (s *AuthService) GenerateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

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
