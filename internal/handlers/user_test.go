package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/handlers"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func setupUserHandlerTest(t *testing.T, role models.UserRole) (*handlers.UserHandler, *services.AuthService, *services.AuditService, *gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{
			BcryptCost: 4, // Low cost for faster tests
		},
	}

	authService := services.NewAuthService(db, cfg)
	auditService := services.NewAuditService(db)
	handler := handlers.NewUserHandler(authService, auditService)

	// Create test admin user
	_, err = authService.CreateUserWithRole("admin", "password", models.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set the current user based on role parameter
		user := &models.User{
			ID:       1,
			Username: "admin",
			Role:     role,
			IsAdmin:  role == models.RoleAdmin,
		}
		c.Set(middleware.UserContextKey, user)
		c.Next()
	})

	router.GET("/api/users", handler.List)
	router.POST("/api/users", handler.Create)
	router.GET("/api/users/:id", handler.Get)
	router.PUT("/api/users/:id", handler.Update)
	router.PUT("/api/users/:id/password", handler.UpdatePassword)
	router.DELETE("/api/users/:id", handler.Delete)

	cleanup := func() {
		db.Close()
	}

	return handler, authService, auditService, router, cleanup
}

func TestUserHandler_List_AsAdmin(t *testing.T) {
	_, _, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["users"] == nil {
		t.Error("expected users in response")
	}
}

func TestUserHandler_List_AsViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{BcryptCost: 4},
	}

	authService := services.NewAuthService(db, cfg)
	auditService := services.NewAuditService(db)
	handler := handlers.NewUserHandler(authService, auditService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		user := &models.User{
			ID:       1,
			Username: "viewer",
			Role:     models.RoleViewer,
			IsAdmin:  false,
		}
		c.Set(middleware.UserContextKey, user)
		c.Next()
	})
	router.GET("/api/users", handler.List)

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestUserHandler_Create(t *testing.T) {
	_, _, auditService, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	body := map[string]string{
		"username": "newuser",
		"password": "password123",
		"role":     "operator",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var user map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if user["username"] != "newuser" {
		t.Errorf("expected username 'newuser', got %v", user["username"])
	}

	if user["role"] != "operator" {
		t.Errorf("expected role 'operator', got %v", user["role"])
	}

	// Verify audit log
	logs, _ := auditService.GetLogs(10, 0)
	found := false
	for _, log := range logs {
		if log.Action == "user_create" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for user_create action")
	}
}

func TestUserHandler_Create_DuplicateUsername(t *testing.T) {
	_, authService, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	// Create a user first
	authService.CreateUserWithRole("existing", "password", models.RoleOperator)

	body := map[string]string{
		"username": "existing",
		"password": "password123",
		"role":     "operator",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestUserHandler_Update(t *testing.T) {
	_, authService, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	// Create a user to update
	user, _ := authService.CreateUserWithRole("updateme", "password", models.RoleOperator)

	body := map[string]string{
		"username": "updated",
		"role":     "viewer",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/users/"+string(rune(user.ID+48)), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Note: This will fail because user ID 1 is the current user (cannot modify own account)
	// Let's test with user ID 2
	req = httptest.NewRequest("PUT", "/api/users/2", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_Delete_CannotDeleteSelf(t *testing.T) {
	_, _, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/api/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "cannot delete your own account" {
		t.Errorf("expected 'cannot delete your own account' error, got %s", response["error"])
	}
}

func TestUserHandler_Delete_CannotDeleteLastAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{BcryptCost: 4},
	}

	authService := services.NewAuthService(db, cfg)
	auditService := services.NewAuditService(db)
	handler := handlers.NewUserHandler(authService, auditService)

	// Create only one admin
	admin, _ := authService.CreateUserWithRole("admin", "password", models.RoleAdmin)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set a different admin user as current
		user := &models.User{
			ID:       999, // Different ID
			Username: "otheradmin",
			Role:     models.RoleAdmin,
			IsAdmin:  true,
		}
		c.Set(middleware.UserContextKey, user)
		c.Next()
	})
	router.DELETE("/api/users/:id", handler.Delete)

	req := httptest.NewRequest("DELETE", "/api/users/"+string(rune(admin.ID+48)), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail because it's the last admin
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_UpdatePassword(t *testing.T) {
	_, authService, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	// Create a user to update password
	authService.CreateUserWithRole("passworduser", "oldpassword", models.RoleOperator)

	body := map[string]string{
		"password": "newpassword123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/users/2/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_Get(t *testing.T) {
	_, _, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var user map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if user["username"] != "admin" {
		t.Errorf("expected username 'admin', got %v", user["username"])
	}
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	_, _, _, router, cleanup := setupUserHandlerTest(t, models.RoleAdmin)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/users/999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
