package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/handlers"
)

func TestVersionHandler_CheckUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := handlers.NewVersionHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/version/check", nil)

	handler.CheckUpdate(c)

	// Should always return 200 even if check fails
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Should return valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should have current version
	if response["current"] == nil {
		t.Error("expected current version in response")
	}

	// Should have update_available field
	if _, ok := response["update_available"]; !ok {
		t.Error("expected update_available field in response")
	}
}
