package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

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
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
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

	session, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		// Audit failed login attempt
		_ = h.auditService.Log(services.AuditLog{
			Username:     req.Username,
			Action:       "login_failed",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})

		if c.GetHeader("Content-Type") == "application/json" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"PathPrefix": h.pathPrefix,
			"Error":      "Invalid username or password",
		})
		return
	}

	// Get user for audit log
	if user, err := h.authService.GetUserByID(session.UserID); err == nil && user != nil {
		h.auditService.LogLogin(user, c.ClientIP(), c.GetHeader("User-Agent"), true)
	}

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
