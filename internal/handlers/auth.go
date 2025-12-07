package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pandeptwidyaop/http-remote/internal/validation"
	"github.com/pquerna/otp/totp"
)

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

// AuthHandler handles HTTP requests for user authentication.
type AuthHandler struct {
	authService  *services.AuthService
	auditService *services.AuditService
	pathPrefix   string
	secureCookie bool
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *services.AuthService, auditService *services.AuditService, pathPrefix string, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		auditService: auditService,
		pathPrefix:   pathPrefix,
		secureCookie: secureCookie,
	}
}

// LoginRequest contains user login credentials.
type LoginRequest struct {
	Username   string `json:"username" form:"username" binding:"required"`
	Password   string `json:"password" form:"password" binding:"required"`
	TOTPCode   string `json:"totp_code,omitempty" form:"totp_code"`
	BackupCode string `json:"backup_code,omitempty" form:"backup_code"`
}

// LoginPage renders the login page.
func (h *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"PathPrefix": h.pathPrefix,
	})
}

// Login handles user authentication.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if c.GetHeader("Content-Type") == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
	} else {
		if err := c.ShouldBind(&req); err != nil {
			c.HTML(http.StatusBadRequest, "login.html", gin.H{
				"PathPrefix": h.pathPrefix,
				"Error":      "Username and password are required",
			})
			return
		}
	}

	// Check if account is locked
	if locked, remaining := h.authService.IsAccountLocked(req.Username); locked {
		_ = h.auditService.Log(services.AuditLog{
			Username:     req.Username,
			Action:       "login_blocked_locked",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})

		if c.GetHeader("Content-Type") == "application/json" {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               "account temporarily locked",
				"retry_after_seconds": int(remaining.Seconds()),
			})
			return
		}
		c.HTML(http.StatusTooManyRequests, "login.html", gin.H{
			"PathPrefix": h.pathPrefix,
			"Error":      "Account temporarily locked. Please try again in " + formatDuration(remaining),
		})
		return
	}

	// Verify credentials using timing-safe comparison
	// This prevents username enumeration via timing attacks
	user, valid := h.authService.VerifyCredentials(req.Username, req.Password)
	if !valid {
		// Record failed login attempt
		_ = h.authService.RecordLoginAttempt(req.Username, c.ClientIP(), false)

		// Audit failed login attempt
		_ = h.auditService.Log(services.AuditLog{
			Username:     req.Username,
			Action:       "login_failed",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})

		// Check remaining attempts
		failedAttempts := h.authService.GetRecentFailedAttempts(req.Username)
		maxAttempts := 5 // Default
		remainingAttempts := maxAttempts - failedAttempts
		if remainingAttempts < 0 {
			remainingAttempts = 0
		}

		if c.GetHeader("Content-Type") == "application/json" {
			response := gin.H{"error": "invalid credentials"}
			if remainingAttempts > 0 && remainingAttempts <= 3 {
				response["remaining_attempts"] = remainingAttempts
			}
			c.JSON(http.StatusUnauthorized, response)
			return
		}

		errorMsg := "Invalid username or password"
		if remainingAttempts > 0 && remainingAttempts <= 3 {
			errorMsg = fmt.Sprintf("%s. %d attempts remaining", errorMsg, remainingAttempts)
		}
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"PathPrefix": h.pathPrefix,
			"Error":      errorMsg,
		})
		return
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		// TOTP or backup code is required
		if req.TOTPCode == "" && req.BackupCode == "" {
			// Return response indicating 2FA is required
			if c.GetHeader("Content-Type") == "application/json" {
				c.JSON(http.StatusOK, gin.H{
					"requires_totp": true,
					"message":       "2FA code required",
				})
				return
			}
			c.HTML(http.StatusOK, "login.html", gin.H{
				"PathPrefix":   h.pathPrefix,
				"RequiresTOTP": true,
				"Username":     req.Username,
			})
			return
		}

		var valid bool

		// Try backup code first if provided
		if req.BackupCode != "" {
			var err error
			valid, err = h.authService.ValidateBackupCode(user.ID, req.BackupCode)
			if err != nil {
				if c.GetHeader("Content-Type") == "application/json" {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate backup code"})
					return
				}
				c.HTML(http.StatusInternalServerError, "login.html", gin.H{
					"PathPrefix":   h.pathPrefix,
					"RequiresTOTP": true,
					"Username":     req.Username,
					"Error":        "Failed to validate backup code",
				})
				return
			}
			if valid {
				// Audit successful backup code usage
				_ = h.auditService.Log(services.AuditLog{
					UserID:       &user.ID,
					Username:     user.Username,
					Action:       "backup_code_used",
					ResourceType: "auth",
					IPAddress:    c.ClientIP(),
					UserAgent:    c.GetHeader("User-Agent"),
				})
			}
		} else if req.TOTPCode != "" {
			// Verify TOTP code
			valid = totp.Validate(req.TOTPCode, user.TOTPSecret)
		}

		if !valid {
			// Audit failed 2FA attempt
			_ = h.auditService.Log(services.AuditLog{
				UserID:       &user.ID,
				Username:     user.Username,
				Action:       "2fa_failed",
				ResourceType: "auth",
				IPAddress:    c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
			})

			if c.GetHeader("Content-Type") == "application/json" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid 2FA code or backup code"})
				return
			}
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{
				"PathPrefix":   h.pathPrefix,
				"RequiresTOTP": true,
				"Username":     req.Username,
				"Error":        "Invalid 2FA code or backup code",
			})
			return
		}
	}

	// Create session after successful authentication (including 2FA if enabled)
	// Clear failed login attempts on successful login
	_ = h.authService.ClearLoginAttempts(req.Username)
	_ = h.authService.RecordLoginAttempt(req.Username, c.ClientIP(), true)

	_ = h.authService.InvalidateUserSessions(user.ID)
	session, err := h.authService.CreateSessionWithBinding(user.ID, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if c.GetHeader("Content-Type") == "application/json" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"PathPrefix": h.pathPrefix,
			"Error":      "Failed to create session",
		})
		return
	}

	// Audit successful login
	h.auditService.LogLogin(user, c.ClientIP(), c.GetHeader("User-Agent"), true)

	c.SetCookie(
		middleware.SessionCookieName,
		session.ID,
		int(session.ExpiresAt.Unix()-session.CreatedAt.Unix()),
		"/",
		"",
		h.secureCookie, // Use config value
		true,
	)

	if c.GetHeader("Content-Type") == "application/json" {
		c.JSON(http.StatusOK, gin.H{
			"message":    "login successful",
			"expires_at": session.ExpiresAt,
		})
		return
	}

	c.Redirect(http.StatusFound, h.pathPrefix+"/")
}

// Logout logs out the current user.
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get user from context for audit log
	userVal, exists := c.Get(middleware.UserContextKey)
	if exists {
		if user, ok := userVal.(*models.User); ok {
			h.auditService.LogLogout(user, c.ClientIP(), c.GetHeader("User-Agent"))
		}
	}

	sessionID, err := c.Cookie(middleware.SessionCookieName)
	if err == nil && sessionID != "" {
		_ = h.authService.DeleteSession(sessionID)
	}

	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", h.secureCookie, true)

	if c.GetHeader("Content-Type") == "application/json" || c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	c.Redirect(http.StatusFound, h.pathPrefix+"/login")
}

// Me returns the current user's information.
func (h *AuthHandler) Me(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	u, ok := user.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"is_admin": u.IsAdmin,
	})
}

// ChangePasswordRequest represents a request to change user password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword handles password change requests for authenticated users.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userObj, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password complexity
	if err := validation.ValidatePasswordWithDefault(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Change password
	if err := h.authService.ChangePassword(user.ID, req.OldPassword, req.NewPassword); err != nil {
		if err == services.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid old password"})
			return
		}
		if err == services.ErrPasswordReused {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		return
	}

	// Audit password change
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "password_changed",
		ResourceType: "auth",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully",
	})
}
