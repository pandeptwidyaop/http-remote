package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Helper to check if Docker is available for tests
func isDockerAvailable() bool {
	service := services.NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return service.IsDockerAvailable(ctx)
}

func TestNewContainerHandler(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.service != service {
		t.Error("expected service to be set")
	}
}

func TestContainerHandler_CheckDocker(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/containers/status", handler.CheckDocker)

	req, _ := http.NewRequest("GET", "/api/containers/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := response["available"]; !ok {
		t.Error("expected 'available' field in response")
	}
}

func TestContainerHandler_List(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.GET("/api/containers", handler.List)

	// Test without 'all' parameter (default: false)
	req, _ := http.NewRequest("GET", "/api/containers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test with 'all=true'
	req, _ = http.NewRequest("GET", "/api/containers?all=true", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test with 'all=false'
	req, _ = http.NewRequest("GET", "/api/containers?all=false", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestContainerHandler_Get_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.GET("/api/containers/:id", handler.Get)

	// Test with empty ID (route won't match, but let's test the handler directly)
	req, _ := http.NewRequest("GET", "/api/containers/", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Get(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_Get_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.GET("/api/containers/:id", handler.Get)

	req, _ := http.NewRequest("GET", "/api/containers/nonexistent-container-12345", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Start_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers//start", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Start(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_Start_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/start", handler.Start)

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Stop_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers//stop", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Stop(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_Stop_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/stop", handler.Stop)

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Stop_WithTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/stop", handler.Stop)

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/stop?timeout=30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Restart_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers//restart", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Restart(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_Restart_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/restart", handler.Restart)

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Remove_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/api/containers/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.Remove(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_Remove_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.DELETE("/api/containers/:id", handler.Remove)

	req, _ := http.NewRequest("DELETE", "/api/containers/nonexistent-container-12345", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Remove_WithForce(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.DELETE("/api/containers/:id", handler.Remove)

	req, _ := http.NewRequest("DELETE", "/api/containers/nonexistent-container-12345?force=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_StreamLogs_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/containers//logs", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.StreamLogs(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_ExecCommand_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers//exec", strings.NewReader(`{"cmd": ["ls"]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.ExecCommand(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_ExecCommand_MissingCmd(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers/test-container/exec", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-container"}}

	handler.ExecCommand(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_ExecCommand_InvalidJSON(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/containers/test-container/exec", strings.NewReader(`invalid json`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "test-container"}}

	handler.ExecCommand(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestContainerHandler_ExecCommand_NotFound(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/exec", handler.ExecCommand)

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/exec",
		strings.NewReader(`{"cmd": ["ls", "-la"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_ExecInteractive_MissingID(t *testing.T) {
	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/containers//terminal", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.ExecInteractive(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test query parameter parsing
func TestContainerHandler_Stop_InvalidTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/stop", handler.Stop)

	// Invalid timeout should be ignored and use nil
	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/stop?timeout=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should still process the request (and fail with 404)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestContainerHandler_Restart_InvalidTimeout(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/restart", handler.Restart)

	// Invalid timeout should be ignored and use nil
	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/restart?timeout=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should still process the request (and fail with 404)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

// Test user context extraction
func TestContainerHandler_Start_WithUserContext(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.POST("/api/containers/:id/start", func(c *gin.Context) {
		// Simulate session middleware setting user context
		c.Set("userID", int64(1))
		c.Set("username", "testuser")
		handler.Start(c)
	})

	req, _ := http.NewRequest("POST", "/api/containers/nonexistent-container-12345/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 for non-existent container
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

// Test SSE headers for log streaming
func TestContainerHandler_StreamLogs_Headers(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.GET("/api/containers/:id/logs", handler.StreamLogs)

	// This will fail because container doesn't exist, but we can check headers
	req, _ := http.NewRequest("GET", "/api/containers/nonexistent-container-12345/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check that SSE headers were set
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got '%s'", cacheControl)
	}
}

// Test log streaming query parameters
func TestContainerHandler_StreamLogs_QueryParams(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}

	service := services.NewContainerService(nil)
	handler := NewContainerHandler(service)

	r := gin.New()
	r.GET("/api/containers/:id/logs", handler.StreamLogs)

	// Test with various query parameters
	req, _ := http.NewRequest("GET", "/api/containers/nonexistent-container-12345/logs?follow=false&tail=50&timestamps=true&since=2024-01-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Just verify it doesn't panic with various params
	if w.Code == 0 {
		t.Error("expected non-zero status code")
	}
}
