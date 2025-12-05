package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	pathPrefix  string
}

func NewAuthHandler(authService *services.AuthService, pathPrefix string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		pathPrefix:  pathPrefix,
	}
}

type LoginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"PathPrefix": h.pathPrefix,
	})
}

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

	c.SetCookie(
		middleware.SessionCookieName,
		session.ID,
		int(session.ExpiresAt.Unix()-session.CreatedAt.Unix()),
		"/",
		"",
		false,
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

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(middleware.SessionCookieName)
	if sessionID != "" {
		h.authService.DeleteSession(sessionID)
	}

	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)

	if c.GetHeader("Content-Type") == "application/json" || c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	c.Redirect(http.StatusFound, h.pathPrefix+"/login")
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	u := user.(*models.User)
	c.JSON(http.StatusOK, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"is_admin": u.IsAdmin,
	})
}
