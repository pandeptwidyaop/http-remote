package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
	"github.com/pandeptwidyaop/http-remote/internal/handlers"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// resolvePath resolves symlinks in a path (needed for macOS where /tmp -> /private/tmp)
// If the file doesn't exist, it resolves the parent directory and joins with the filename
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// File might not exist (e.g., after deletion), try resolving parent
		parentDir := filepath.Dir(path)
		resolvedParent, parentErr := filepath.EvalSymlinks(parentDir)
		if parentErr != nil {
			return path
		}
		return filepath.Join(resolvedParent, filepath.Base(path))
	}
	return resolved
}

func setupFileHandlerTest(t *testing.T) (*services.AuditService, *gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Create in-memory database
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create test user in database for foreign key constraint
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash) VALUES (1, 'testuser', 'hash')`)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	cfg := &config.Config{}
	auditService := services.NewAuditService(db)
	handler := handlers.NewFileHandler(cfg, auditService)

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
	router.GET("/api/files", handler.ListFiles)
	router.GET("/api/files/read", handler.ReadFile)
	router.GET("/api/files/download", handler.DownloadFile)
	router.POST("/api/files/upload", handler.UploadFile)
	router.POST("/api/files/mkdir", handler.CreateDirectory)
	router.POST("/api/files/save", handler.SaveFile)
	router.POST("/api/files/rename", handler.RenameFile)
	router.POST("/api/files/copy", handler.CopyFile)
	router.DELETE("/api/files", handler.DeleteFile)

	cleanup := func() {
		_ = db.Close()
	}

	return auditService, router, cleanup
}

func TestFileHandler_ListFiles(t *testing.T) {
	_, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	// Create temp directory with some files
	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	// Create test files
	_ = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("hello"), 0644) // #nosec G306
	_ = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)                        // #nosec G301

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files?path="+tempDir, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Fatal("expected files array in response")
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFileHandler_ListFiles_NotFound(t *testing.T) {
	_, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files?path=/nonexistent/path", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestFileHandler_ReadFile(t *testing.T) {
	_, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	// Create temp file
	tempFile, err := os.CreateTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) { _ = os.Remove(name) }(tempFile.Name())

	content := "Hello, World!"
	_, _ = tempFile.WriteString(content)
	_ = tempFile.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files/read?path="+tempFile.Name(), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["content"] != content {
		t.Errorf("expected content '%s', got '%s'", content, response["content"])
	}

	if response["is_binary"] != false {
		t.Error("expected is_binary to be false")
	}
}

func TestFileHandler_ReadFile_Directory(t *testing.T) {
	_, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files/read?path="+tempDir, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestFileHandler_CreateDirectory(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	newDirPath := filepath.Join(tempDir, "newdir")
	body := `{"path":"` + newDirPath + `"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/mkdir", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify directory was created
	if _, err := os.Stat(newDirPath); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	for _, log := range logs {
		// Check action is mkdir and path contains "newdir"
		if log.Action == "mkdir" && strings.Contains(log.ResourceID, "newdir") {
			found = true
			if log.Username != "testuser" {
				t.Errorf("expected username 'testuser', got '%s'", log.Username)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected audit log for mkdir action, got logs: %+v", logs)
	}
}

func TestFileHandler_SaveFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	filePath := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	body := `{"path":"` + filePath + `","content":"` + content + `"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/save", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was created with content
	data, err := os.ReadFile(filePath) // #nosec G304 - test file
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedFilePath := resolvePath(filePath)
	for _, log := range logs {
		if log.Action == "save" && log.ResourceID == resolvedFilePath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for save action")
	}
}

func TestFileHandler_DeleteFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	// Create temp file to delete
	tempFile, err := os.CreateTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_ = tempFile.Close()
	filePath := tempFile.Name()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/files?path="+filePath, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedFilePath := resolvePath(filePath)
	for _, log := range logs {
		if log.Action == "delete" && log.ResourceID == resolvedFilePath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for delete action")
	}
}

func TestFileHandler_DeleteFile_SystemPath(t *testing.T) {
	_, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	systemPaths := []string{"/", "/bin", "/etc", "/usr", "/var"}

	for _, path := range systemPaths {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/api/files?path="+path, nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403 for path %s, got %d", path, w.Code)
		}
	}
}

func TestFileHandler_RenameFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	oldPath := filepath.Join(tempDir, "old.txt")
	newPath := filepath.Join(tempDir, "new.txt")
	_ = os.WriteFile(oldPath, []byte("content"), 0644) // #nosec G306 - test file

	body := `{"old_path":"` + oldPath + `","new_path":"` + newPath + `"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/rename", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify old file doesn't exist
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected old file to not exist")
	}

	// Verify new file exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("expected new file to exist")
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedNewPath := resolvePath(newPath)
	for _, log := range logs {
		if log.Action == "rename" && log.ResourceID == resolvedNewPath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for rename action")
	}
}

func TestFileHandler_CopyFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	sourcePath := filepath.Join(tempDir, "source.txt")
	destPath := filepath.Join(tempDir, "dest.txt")
	content := "test content"
	_ = os.WriteFile(sourcePath, []byte(content), 0644) // #nosec G306 - test file

	body := `{"source_path":"` + sourcePath + `","dest_path":"` + destPath + `"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/copy", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify source still exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		t.Error("expected source file to still exist")
	}

	// Verify dest exists with same content
	data, err := os.ReadFile(destPath) // #nosec G304
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content '%s', got '%s'", content, string(data))
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedDestPath := resolvePath(destPath)
	for _, log := range logs {
		if log.Action == "copy" && log.ResourceID == resolvedDestPath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for copy action")
	}
}

func TestFileHandler_UploadFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("path", tempDir)
	part, err := writer.CreateFormFile("file", "uploaded.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("uploaded content"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was uploaded
	uploadedPath := filepath.Join(tempDir, "uploaded.txt")
	if _, err := os.Stat(uploadedPath); os.IsNotExist(err) {
		t.Error("expected uploaded file to exist")
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedUploadedPath := resolvePath(uploadedPath)
	for _, log := range logs {
		if log.Action == "upload" && log.ResourceID == resolvedUploadedPath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for upload action")
	}
}

func TestFileHandler_DownloadFile(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	// Create temp file to download
	tempFile, err := os.CreateTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func(name string) { _ = os.Remove(name) }(tempFile.Name())

	content := "download content"
	_, _ = tempFile.WriteString(content)
	_ = tempFile.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files/download?path="+tempFile.Name(), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify content-disposition header
	contentDisp := w.Header().Get("Content-Disposition")
	if contentDisp == "" {
		t.Error("expected Content-Disposition header")
	}

	// Verify audit log was created
	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	found := false
	resolvedFilePath := resolvePath(tempFile.Name())
	for _, log := range logs {
		if log.Action == "download" && log.ResourceID == resolvedFilePath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log for download action")
	}
}

func TestFileHandler_AuditLogDetails(t *testing.T) {
	auditService, router, cleanup := setupFileHandlerTest(t)
	defer cleanup()

	tempDir, err := os.MkdirTemp("", "filehandler_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tempDir)

	// Test that save action includes file_size in details
	filePath := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	body := `{"path":"` + filePath + `","content":"` + content + `"}`

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/files/save", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("expected at least one audit log")
	}

	for _, log := range logs {
		if log.Action == "save" && strings.Contains(log.ResourceID, "test.txt") {
			// Check that details contains file_size
			if log.Details == "" {
				t.Error("expected details to contain file_size")
			}
			var details map[string]interface{}
			if err := json.Unmarshal([]byte(log.Details), &details); err != nil {
				t.Fatalf("failed to unmarshal details: %v", err)
			}
			if _, ok := details["file_size"]; !ok {
				t.Error("expected details to contain file_size")
			}
			break
		}
	}
}

func TestAuditServiceDirect(t *testing.T) {
	// Direct test of audit service
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create test user for foreign key constraint
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash) VALUES (1, 'testuser', 'hash')`)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	auditService := services.NewAuditService(db)

	userID := int64(1)
	err = auditService.Log(services.AuditLog{
		UserID:       &userID,
		Username:     "testuser",
		Action:       "test_action",
		ResourceType: "file",
		ResourceID:   "/test/path",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test",
	})
	if err != nil {
		t.Fatalf("failed to log audit: %v", err)
	}

	logs, err := auditService.GetLogs(10, 0)
	if err != nil {
		t.Fatalf("failed to get audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Action != "test_action" {
		t.Errorf("expected action 'test_action', got '%s'", logs[0].Action)
	}
}
