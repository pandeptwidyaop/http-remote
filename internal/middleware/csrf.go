package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenHeader is the header name for CSRF token
	CSRFTokenHeader = "X-CSRF-Token" // #nosec G101 - not a credential, just a header name
	// CSRFTokenCookie is the cookie name for CSRF token
	CSRFTokenCookie = "csrf_token"
	// CSRFContextKey is the key for storing CSRF token in request context
	CSRFContextKey = "csrf_token"
)

// CSRFToken represents a CSRF token with expiration
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFStore stores and manages CSRF tokens
type CSRFStore struct {
	tokens map[string]CSRFToken
	mu     sync.RWMutex
}

// NewCSRFStore creates a new CSRF token store
func NewCSRFStore() *CSRFStore {
	store := &CSRFStore{
		tokens: make(map[string]CSRFToken),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// GenerateToken creates a new CSRF token
func (s *CSRFStore) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	s.mu.Lock()
	s.tokens[token] = CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s.mu.Unlock()

	return token, nil
}

// ValidateToken checks if a CSRF token is valid
func (s *CSRFStore) ValidateToken(token string) bool {
	s.mu.RLock()
	csrfToken, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(csrfToken.ExpiresAt) {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return false
	}

	return true
}

// cleanup removes expired tokens periodically
func (s *CSRFStore) cleanup() {
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, data := range s.tokens {
			if now.After(data.ExpiresAt) {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

// CSRFProtection creates a middleware that protects against CSRF attacks
func CSRFProtection(store *CSRFStore, pathPrefix string, secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF check for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Ensure a CSRF token exists for the response
			ensureCSRFToken(c, store, pathPrefix, secureCookie)
			c.Next()
			return
		}

		// Skip CSRF for API deploy endpoints (they use token auth)
		if contains(c.Request.URL.Path, "/deploy/") {
			c.Next()
			return
		}

		// Skip CSRF for login endpoint (no session yet)
		if contains(c.Request.URL.Path, "/auth/login") {
			c.Next()
			return
		}

		// For state-changing methods, validate CSRF token
		token := c.GetHeader(CSRFTokenHeader)
		if token == "" {
			// Also check form data
			token = c.PostForm("_csrf")
		}

		if token == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			c.Abort()
			return
		}

		if !store.ValidateToken(token) {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token invalid"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ensureCSRFToken makes sure a CSRF token cookie exists
func ensureCSRFToken(c *gin.Context, store *CSRFStore, _ string, secureCookie bool) {
	// Check if token already exists in cookie
	existingToken, err := c.Cookie(CSRFTokenCookie)
	if err == nil && existingToken != "" && store.ValidateToken(existingToken) {
		c.Set(CSRFContextKey, existingToken)
		return
	}

	// Generate new token
	token, err := store.GenerateToken()
	if err != nil {
		return
	}

	// Set cookie (24h expiration, HttpOnly=false so JS can read it)
	c.SetCookie(
		CSRFTokenCookie,
		token,
		86400, // 24 hours
		"/",
		"",
		secureCookie,
		false, // HttpOnly=false - JS needs to read this
	)
	c.Set(CSRFContextKey, token)
}

// GetCSRFToken returns the CSRF token for the current request
func GetCSRFToken(c *gin.Context) string {
	token, exists := c.Get(CSRFContextKey)
	if !exists {
		return ""
	}
	return token.(string)
}
