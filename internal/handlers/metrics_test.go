package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/handlers"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

func setupMetricsHandlerTest(t *testing.T) (*gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "metrics_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := tempDir + "/test.db"

	// Create database
	db, err := database.New(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create test user in database for foreign key constraint
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash) VALUES (1, 'testuser', 'hash')`)
	if err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create metrics config
	enabled := true
	cfg := &config.MetricsConfig{
		Enabled:             &enabled,
		CollectionInterval:  "1m",
		RetentionDays:       7,
		HourlyRetentionDays: 30,
		DailyRetentionDays:  365,
	}

	// Create metrics collector
	collector := services.NewMetricsCollector(db.DB, cfg)

	// Create handler
	handler := handlers.NewMetricsHandler(collector, dbPath)

	// Create router with test user middleware
	router := gin.New()
	router.Use(func(c *gin.Context) {
		user := &models.User{
			ID:       1,
			Username: "testuser",
		}
		c.Set(middleware.UserContextKey, user)
		c.Next()
	})

	// Register routes
	router.GET("/api/metrics/system", handler.GetSystem)
	router.GET("/api/metrics/docker", handler.GetDocker)
	router.GET("/api/metrics/docker/:id", handler.GetContainer)
	router.GET("/api/metrics/docker/:id/history", handler.GetContainerHistory)
	router.GET("/api/metrics/summary", handler.GetSummary)
	router.GET("/api/metrics/history", handler.GetHistorical)
	router.GET("/api/metrics/storage", handler.GetDatabaseInfo)
	router.POST("/api/metrics/prune", handler.PruneMetrics)
	router.POST("/api/metrics/vacuum", handler.VacuumDatabase)

	cleanup := func() {
		collector.Stop()
		_ = db.Close()
		os.RemoveAll(tempDir)
	}

	return router, cleanup
}

func TestMetricsHandler_GetSystem(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/metrics/system", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check for expected fields
	if _, ok := response["cpu"]; !ok {
		t.Error("expected 'cpu' field in response")
	}
	if _, ok := response["memory"]; !ok {
		t.Error("expected 'memory' field in response")
	}
	if _, ok := response["disks"]; !ok {
		t.Error("expected 'disks' field in response")
	}
	if _, ok := response["network"]; !ok {
		t.Error("expected 'network' field in response")
	}
}

func TestMetricsHandler_GetDocker(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/metrics/docker", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check for expected fields
	if _, ok := response["available"]; !ok {
		t.Error("expected 'available' field in response")
	}
	if _, ok := response["summary"]; !ok {
		t.Error("expected 'summary' field in response")
	}
}

func TestMetricsHandler_GetSummary(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/metrics/summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check for expected fields
	if _, ok := response["system"]; !ok {
		t.Error("expected 'system' field in response")
	}
	if _, ok := response["docker"]; !ok {
		t.Error("expected 'docker' field in response")
	}

	// Check system struct has expected nested fields
	if system, ok := response["system"].(map[string]interface{}); ok {
		if _, ok := system["cpu_percent"]; !ok {
			t.Error("expected 'cpu_percent' field in system")
		}
		if _, ok := system["memory_percent"]; !ok {
			t.Error("expected 'memory_percent' field in system")
		}
	}
}

func TestMetricsHandler_GetHistorical(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	t.Run("default parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/history", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if _, ok := response["data"]; !ok {
			t.Error("expected 'data' field in response")
		}
		if _, ok := response["resolution"]; !ok {
			t.Error("expected 'resolution' field in response")
		}
	})

	t.Run("with time range", func(t *testing.T) {
		from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		to := time.Now().Format(time.RFC3339)

		w := httptest.NewRecorder()
		// URL encode the timestamps properly
		req := httptest.NewRequest("GET", "/api/metrics/history", nil)
		q := req.URL.Query()
		q.Add("from", from)
		q.Add("to", to)
		q.Add("resolution", "raw")
		req.URL.RawQuery = q.Encode()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid from timestamp", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/history?from=invalid", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid to timestamp", func(t *testing.T) {
		from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/history", nil)
		q := req.URL.Query()
		q.Add("from", from)
		q.Add("to", "invalid")
		req.URL.RawQuery = q.Encode()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("different resolutions", func(t *testing.T) {
		resolutions := []string{"raw", "hourly", "daily"}
		for _, res := range resolutions {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/metrics/history?resolution="+res, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("resolution %s: expected status 200, got %d", res, w.Code)
			}
		}
	})
}

func TestMetricsHandler_GetContainer(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	// Test with a non-existent container ID
	// This should return an error since the container doesn't exist
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/metrics/docker/nonexistent123", nil)
	router.ServeHTTP(w, req)

	// The exact response depends on Docker availability
	// Just verify we get a response
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Errorf("unexpected status %d", w.Code)
	}
}

func TestMetricsHandler_GetContainerHistory(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	t.Run("valid container ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/docker/abc123/history", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if _, ok := response["container_id"]; !ok {
			t.Error("expected 'container_id' field in response")
		}
		if _, ok := response["data"]; !ok {
			t.Error("expected 'data' field in response")
		}
	})

	t.Run("invalid from timestamp", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/docker/abc123/history?from=invalid", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid to timestamp", func(t *testing.T) {
		from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/docker/abc123/history", nil)
		q := req.URL.Query()
		q.Add("from", from)
		q.Add("to", "invalid")
		req.URL.RawQuery = q.Encode()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestMetricsHandler_GetDatabaseInfo(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/metrics/storage", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check for expected fields
	if _, ok := response["path"]; !ok {
		t.Error("expected 'path' field in response")
	}
	if _, ok := response["metrics_count"]; !ok {
		t.Error("expected 'metrics_count' field in response")
	}
	if _, ok := response["size_bytes"]; !ok {
		t.Error("expected 'size_bytes' field in response")
	}
	if _, ok := response["size_formatted"]; !ok {
		t.Error("expected 'size_formatted' field in response")
	}
}

func TestMetricsHandler_PruneMetrics(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	t.Run("valid prune request", func(t *testing.T) {
		before := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
		body := `{"before":"` + before + `"}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/metrics/prune", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("expected success to be true")
		}
	})

	t.Run("missing before timestamp", func(t *testing.T) {
		body := `{}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/metrics/prune", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid before timestamp format", func(t *testing.T) {
		body := `{"before":"invalid-date"}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/metrics/prune", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestMetricsHandler_VacuumDatabase(t *testing.T) {
	router, cleanup := setupMetricsHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/metrics/vacuum", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("expected success to be true")
	}
	if _, ok := response["size_before"]; !ok {
		t.Error("expected 'size_before' field in response")
	}
	if _, ok := response["size_after"]; !ok {
		t.Error("expected 'size_after' field in response")
	}
	if _, ok := response["space_reclaimed"]; !ok {
		t.Error("expected 'space_reclaimed' field in response")
	}
}

func TestMetricsHandler_NilCollector(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create handler with nil collector
	handler := handlers.NewMetricsHandler(nil, "/nonexistent/path")

	router := gin.New()
	router.Use(func(c *gin.Context) {
		user := &models.User{
			ID:       1,
			Username: "testuser",
		}
		c.Set(middleware.UserContextKey, user)
		c.Next()
	})

	router.GET("/api/metrics/history", handler.GetHistorical)
	router.GET("/api/metrics/docker/:id/history", handler.GetContainerHistory)
	router.GET("/api/metrics/storage", handler.GetDatabaseInfo)
	router.POST("/api/metrics/prune", handler.PruneMetrics)
	router.POST("/api/metrics/vacuum", handler.VacuumDatabase)

	t.Run("GetHistorical with nil collector", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/history", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("GetContainerHistory with nil collector", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/docker/abc123/history", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("GetDatabaseInfo with nil collector", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/metrics/storage", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("PruneMetrics with nil collector", func(t *testing.T) {
		before := time.Now().Format(time.RFC3339)
		body := `{"before":"` + before + `"}`

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/metrics/prune", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("VacuumDatabase with nil collector", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/metrics/vacuum", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}
