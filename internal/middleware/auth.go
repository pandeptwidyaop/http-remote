// Package middleware provides HTTP middleware for authentication, logging, and rate limiting.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

const (
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session_id"
	// UserContextKey is the key for storing user in request context.
	UserContextKey = "user"
)

// AuthRequired is a middleware that requires authentication.
func AuthRequired(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil || sessionID == "" {
			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
			redirectToLogin(c)
			return
		}

		user, err := authService.ValidateSession(sessionID)
		if err != nil {
			c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
				c.Abort()
				return
			}
			redirectToLogin(c)
			return
		}

		c.Set(UserContextKey, user)
		c.Next()
	}
}

func isAPIRequest(c *gin.Context) bool {
	return len(c.Request.URL.Path) > 4 && c.Request.URL.Path[len(c.Request.URL.Path)-4:] != "html" &&
		c.GetHeader("Accept") == "application/json" ||
		c.GetHeader("Content-Type") == "application/json" ||
		c.Request.URL.Path[0:4] == "/api" ||
		contains(c.Request.URL.Path, "/api/")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func redirectToLogin(c *gin.Context) {
	pathPrefix := c.GetString("path_prefix")
	c.Redirect(http.StatusFound, pathPrefix+"/login")
	c.Abort()
}

// RequireRole is a middleware that requires a specific role.
func RequireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userObj, exists := c.Get(UserContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		user, ok := userObj.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
			c.Abort()
			return
		}

		if !user.HasPermission(requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin is a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequireOperator is a middleware that requires operator role or higher.
func RequireOperator() gin.HandlerFunc {
	return RequireRole(models.RoleOperator)
}
