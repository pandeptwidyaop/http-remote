package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	authService      *services.AuthService
	auditService     *services.AuditService
	defaultAdminUser string
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(authService *services.AuthService, auditService *services.AuditService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		authService:      authService,
		auditService:     auditService,
		defaultAdminUser: cfg.Admin.Username,
	}
}

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin operator viewer"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Role     string `json:"role" binding:"required,oneof=admin operator viewer"`
}

// UpdatePasswordRequest represents a request to update a user's password.
type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// List returns all users with pagination.
func (h *UserHandler) List(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, total, err := h.authService.GetAllUsers(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// Get returns a single user by ID.
func (h *UserHandler) Get(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.authService.GetUserByID(id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Create creates a new user.
func (h *UserHandler) Create(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := models.UserRole(req.Role)
	user, err := h.authService.CreateUserWithRole(req.Username, req.Password, role)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &currentUser.ID,
		Username:     currentUser.Username,
		Action:       "user_create",
		ResourceType: "user",
		ResourceID:   strconv.FormatInt(user.ID, 10),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"created_username": user.Username,
			"created_role":     user.Role,
		},
	})

	c.JSON(http.StatusCreated, user)
}

// Update updates a user.
func (h *UserHandler) Update(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Prevent self-demotion from admin
	if id == currentUser.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify your own account here, use settings instead"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	existingUser, err := h.authService.GetUserByID(id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prevent modifying the default admin user's username or role
	if existingUser.Username == h.defaultAdminUser {
		if req.Username != h.defaultAdminUser {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot change the default admin username"})
			return
		}
		if req.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot change the default admin role"})
			return
		}
	}

	// If demoting last admin, prevent it
	if existingUser.Role == models.RoleAdmin && req.Role != "admin" {
		count, err := h.authService.CountAdminUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot demote the last admin user"})
			return
		}
	}

	role := models.UserRole(req.Role)
	if err := h.authService.UpdateUser(id, req.Username, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &currentUser.ID,
		Username:     currentUser.Username,
		Action:       "user_update",
		ResourceType: "user",
		ResourceID:   strconv.FormatInt(id, 10),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"updated_username": req.Username,
			"updated_role":     req.Role,
			"old_username":     existingUser.Username,
			"old_role":         existingUser.Role,
		},
	})

	// Get updated user
	user, _ := h.authService.GetUserByID(id)
	c.JSON(http.StatusOK, user)
}

// UpdatePassword updates a user's password.
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check user exists
	targetUser, err := h.authService.GetUserByID(id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.UpdateUserPassword(id, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invalidate all sessions for this user
	_ = h.authService.InvalidateUserSessions(id)

	// Log audit
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &currentUser.ID,
		Username:     currentUser.Username,
		Action:       "user_password_reset",
		ResourceType: "user",
		ResourceID:   strconv.FormatInt(id, 10),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"target_username": targetUser.Username,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}

// Delete deletes a user.
func (h *UserHandler) Delete(c *gin.Context) {
	// Check permission
	currentUser := c.MustGet(middleware.UserContextKey).(*models.User)
	if !currentUser.CanManageUsers() {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Prevent self-deletion
	if id == currentUser.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
		return
	}

	// Get user before deletion for audit
	targetUser, err := h.authService.GetUserByID(id)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prevent deleting the default admin user from config
	if targetUser.Username == h.defaultAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete the default admin user"})
		return
	}

	// Prevent deleting last admin
	if targetUser.Role == models.RoleAdmin {
		count, err := h.authService.CountAdminUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last admin user"})
			return
		}
	}

	if err := h.authService.DeleteUser(id); err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log audit
	_ = h.auditService.Log(services.AuditLog{
		UserID:       &currentUser.ID,
		Username:     currentUser.Username,
		Action:       "user_delete",
		ResourceType: "user",
		ResourceID:   strconv.FormatInt(id, 10),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"deleted_username": targetUser.Username,
			"deleted_role":     targetUser.Role,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}
