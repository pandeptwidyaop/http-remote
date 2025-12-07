package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// dangerousPaths contains system paths that should never be modified
var dangerousPaths = []string{
	"/", "/bin", "/boot", "/dev", "/etc", "/lib", "/lib64",
	"/proc", "/root", "/sbin", "/sys", "/usr", "/var",
	// macOS symlinked paths
	"/private/etc", "/private/var", "/private/tmp",
}

// sanitizeFilename removes potentially dangerous characters from filenames
// to prevent path traversal and other security issues
func sanitizeFilename(filename string) string {
	// Remove path separators and null bytes
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove leading dots (hidden files can be security risk)
	filename = strings.TrimLeft(filename, ".")

	// Remove control characters and non-printable characters
	var sanitized strings.Builder
	for _, r := range filename {
		if unicode.IsPrint(r) && r != '\t' && r != '\n' && r != '\r' {
			sanitized.WriteRune(r)
		}
	}
	filename = sanitized.String()

	// Remove dangerous patterns
	dangerousPatterns := regexp.MustCompile(`\.\.+|^-|^\s+|\s+$`)
	filename = dangerousPatterns.ReplaceAllString(filename, "_")

	// Limit filename length (255 is typical filesystem limit)
	if len(filename) > 200 {
		ext := filepath.Ext(filename)
		if len(ext) > 50 {
			ext = ext[:50]
		}
		filename = filename[:200-len(ext)] + ext
	}

	// If filename is empty after sanitization, use a default
	if filename == "" {
		filename = "unnamed_file"
	}

	return filename
}

// securePath validates and resolves a path to prevent path traversal attacks
// Returns the resolved absolute path and any error
func securePath(inputPath string) (string, error) {
	// Clean the path first
	cleanPath := filepath.Clean(inputPath)

	// Resolve any symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If EvalSymlinks fails because path doesn't exist, use the cleaned path
		// This is needed for operations that create new files/dirs
		if os.IsNotExist(err) {
			// For non-existent paths, resolve the parent directory
			parentDir := filepath.Dir(cleanPath)
			realParent, parentErr := filepath.EvalSymlinks(parentDir)
			if parentErr != nil {
				if os.IsNotExist(parentErr) {
					// Parent also doesn't exist, just use cleaned path
					realPath = cleanPath
				} else {
					return "", fmt.Errorf("invalid path: %w", parentErr)
				}
			} else {
				realPath = filepath.Join(realParent, filepath.Base(cleanPath))
			}
		} else {
			return "", fmt.Errorf("invalid path: %w", err)
		}
	}

	// Check against dangerous system paths
	for _, dp := range dangerousPaths {
		if realPath == dp {
			return "", fmt.Errorf("access to system path %s is forbidden", dp)
		}
	}

	return realPath, nil
}

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"`
}

// FileHandler handles file operations
type FileHandler struct {
	cfg          *config.Config
	auditService *services.AuditService
}

// NewFileHandler creates a new FileHandler instance
func NewFileHandler(cfg *config.Config, auditService *services.AuditService) *FileHandler {
	return &FileHandler{cfg: cfg, auditService: auditService}
}

// logFileAction logs a file operation to audit log
func (h *FileHandler) logFileAction(c *gin.Context, action, path string, details map[string]interface{}) {
	user, exists := c.Get(middleware.UserContextKey)
	if !exists {
		return
	}

	u, ok := user.(*models.User)
	if !ok {
		return
	}

	_ = h.auditService.Log(services.AuditLog{
		UserID:       &u.ID,
		Username:     u.Username,
		Action:       action,
		ResourceType: "file",
		ResourceID:   path,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details:      details,
	})
}

// ListFiles lists files and directories in a given path
func (h *FileHandler) ListFiles(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	// Secure path validation (prevents symlink attacks)
	path, err := securePath(path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If it's a file, return file info
	if !info.IsDir() {
		c.JSON(http.StatusOK, gin.H{
			"path":    path,
			"is_file": true,
			"file": FileInfo{
				Name:        info.Name(),
				Path:        path,
				IsDir:       false,
				Size:        info.Size(),
				ModTime:     info.ModTime(),
				Permissions: info.Mode().String(),
			},
		})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())
		files = append(files, FileInfo{
			Name:        entry.Name(),
			Path:        fullPath,
			IsDir:       entry.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Get parent path
	parentPath := filepath.Dir(path)
	if parentPath == path {
		parentPath = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"path":        path,
		"parent_path": parentPath,
		"files":       files,
	})
}

// ReadFile reads and returns file content
func (h *FileHandler) ReadFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Secure path validation (prevents symlink attacks)
	path, err := securePath(path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is a directory"})
		return
	}

	// Limit file size for reading (10MB)
	maxSize := int64(10 * 1024 * 1024)
	if info.Size() > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "file too large to read",
			"size":     info.Size(),
			"max_size": maxSize,
		})
		return
	}

	content, err := os.ReadFile(path) // #nosec G304 - path validated by securePath
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Detect if file is binary
	isBinary := false
	for _, b := range content[:min(512, len(content))] {
		if b == 0 {
			isBinary = true
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"path":      path,
		"name":      info.Name(),
		"size":      info.Size(),
		"is_binary": isBinary,
		"content":   string(content),
	})
}

// DownloadFile downloads a file
func (h *FileHandler) DownloadFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Secure path validation (prevents symlink attacks)
	path, err := securePath(path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot download directory"})
		return
	}

	// Log download action
	h.logFileAction(c, "download", path, map[string]interface{}{
		"file_name": info.Name(),
		"file_size": info.Size(),
	})

	// Set headers for download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", info.Name()))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))

	c.File(path)
}

// UploadFile handles file upload
func (h *FileHandler) UploadFile(c *gin.Context) {
	targetPath := c.PostForm("path")
	if targetPath == "" {
		targetPath = "/tmp"
	}

	// Secure path validation (prevents symlink attacks)
	targetPath, err := securePath(targetPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if target path exists and is a directory
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "target path not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target path is not a directory"})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Sanitize filename to prevent path traversal attacks
	safeFilename := sanitizeFilename(file.Filename)

	// Create destination path with sanitized filename
	destPath := filepath.Join(targetPath, safeFilename)

	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		// File exists, check if overwrite is allowed
		overwrite := c.PostForm("overwrite")
		if overwrite != "true" {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "file already exists",
				"path":    destPath,
				"message": "set overwrite=true to replace",
			})
			return
		}
	}

	// Save the file
	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get file info after upload
	uploadedInfo, _ := os.Stat(destPath)

	// Log upload action
	h.logFileAction(c, "upload", destPath, map[string]interface{}{
		"original_name": file.Filename,
		"safe_name":     safeFilename,
		"file_size":     uploadedInfo.Size(),
		"target_path":   targetPath,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "file uploaded successfully",
		"file": FileInfo{
			Name:        safeFilename,
			Path:        destPath,
			IsDir:       false,
			Size:        uploadedInfo.Size(),
			ModTime:     uploadedInfo.ModTime(),
			Permissions: uploadedInfo.Mode().String(),
		},
	})
}

// DeleteFile deletes a file or empty directory
func (h *FileHandler) DeleteFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	// Secure path validation (prevents symlink attacks and system path access)
	path, err := securePath(path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If it's a directory, only allow deleting empty directories unless recursive is set
	if info.IsDir() {
		recursive := c.Query("recursive")
		if recursive == "true" {
			if err := os.RemoveAll(path); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			if err := os.Remove(path); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "directory not empty",
					"message": "use recursive=true to delete non-empty directories",
				})
				return
			}
		}
	} else {
		if err := os.Remove(path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Log delete action
	h.logFileAction(c, "delete", path, map[string]interface{}{
		"is_dir":    info.IsDir(),
		"recursive": c.Query("recursive") == "true",
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "deleted successfully",
		"path":    path,
	})
}

// CreateDirectory creates a new directory
func (h *FileHandler) CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Secure path validation (prevents symlink attacks and system path access)
	path, err := securePath(req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "path already exists"})
		return
	}

	// Create directory with parents
	if err := os.MkdirAll(path, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, _ := os.Stat(path)

	// Log mkdir action
	h.logFileAction(c, "mkdir", path, nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "directory created successfully",
		"file": FileInfo{
			Name:        filepath.Base(path),
			Path:        path,
			IsDir:       true,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		},
	})
}

// SaveFile saves content to a file
func (h *FileHandler) SaveFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Secure path validation (prevents symlink attacks and system path access)
	path, err := securePath(req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write file
	if err := os.WriteFile(path, []byte(req.Content), 0600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, _ := os.Stat(path)

	// Log save action
	h.logFileAction(c, "save", path, map[string]interface{}{
		"file_size": info.Size(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "file saved successfully",
		"file": FileInfo{
			Name:        filepath.Base(path),
			Path:        path,
			IsDir:       false,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		},
	})
}

// RenameFile renames or moves a file/directory
func (h *FileHandler) RenameFile(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Secure path validation for source (prevents symlink attacks)
	oldPath, err := securePath(req.OldPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "source: " + err.Error()})
		return
	}

	// Secure path validation for destination
	newPath, err := securePath(req.NewPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "destination: " + err.Error()})
		return
	}

	// Check if source exists
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source path not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if destination already exists
	if _, err := os.Stat(newPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "destination path already exists"})
		return
	}

	// Create parent directories for destination if needed
	parentDir := filepath.Dir(newPath)
	if err := os.MkdirAll(parentDir, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Rename/move
	if err := os.Rename(oldPath, newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	info, _ := os.Stat(newPath)

	// Log rename action
	h.logFileAction(c, "rename", newPath, map[string]interface{}{
		"old_path": oldPath,
		"new_path": newPath,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "renamed successfully",
		"old_path": oldPath,
		"file": FileInfo{
			Name:        filepath.Base(newPath),
			Path:        newPath,
			IsDir:       info.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		},
	})
}

// CopyFile copies a file
func (h *FileHandler) CopyFile(c *gin.Context) {
	var req struct {
		SourcePath string `json:"source_path" binding:"required"`
		DestPath   string `json:"dest_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Secure path validation for source (prevents symlink attacks)
	sourcePath, err := securePath(req.SourcePath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "source: " + err.Error()})
		return
	}

	// Secure path validation for destination
	destPath, err := securePath(req.DestPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "destination: " + err.Error()})
		return
	}

	// Check if source exists and is a file
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if sourceInfo.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot copy directories"})
		return
	}

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "destination file already exists"})
		return
	}

	// Create parent directories for destination if needed
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Open source file
	source, err := os.Open(sourcePath) // #nosec G304 - path validated by securePath
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = source.Close() }()

	// Create destination file
	dest, err := os.Create(destPath) // #nosec G304 - path validated by securePath
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() { _ = dest.Close() }()

	// Copy content
	if _, err := io.Copy(dest, source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Preserve permissions (non-fatal if fails)
	_ = os.Chmod(destPath, sourceInfo.Mode())

	info, _ := os.Stat(destPath)

	// Log copy action
	h.logFileAction(c, "copy", destPath, map[string]interface{}{
		"source_path": sourcePath,
		"dest_path":   destPath,
		"file_size":   info.Size(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":     "file copied successfully",
		"source_path": sourcePath,
		"file": FileInfo{
			Name:        filepath.Base(destPath),
			Path:        destPath,
			IsDir:       false,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			Permissions: info.Mode().String(),
		},
	})
}
