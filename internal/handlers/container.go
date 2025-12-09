// Package handlers provides HTTP request handlers for the web UI and API.
package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// ContainerHandler handles HTTP requests for container management.
type ContainerHandler struct {
	service  *services.ContainerService
	upgrader websocket.Upgrader
}

// NewContainerHandler creates a new ContainerHandler instance.
func NewContainerHandler(service *services.ContainerService) *ContainerHandler {
	return &ContainerHandler{
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// List returns all containers.
// GET /api/containers?all=true
func (h *ContainerHandler) List(c *gin.Context) {
	showAll := c.DefaultQuery("all", "false") == "true"

	containers, err := h.service.List(c.Request.Context(), showAll)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, containers)
}

// Get returns detailed information about a container.
// GET /api/containers/:id
func (h *ContainerHandler) Get(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	detail, err := h.service.Get(c.Request.Context(), containerID)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Start starts a container.
// POST /api/containers/:id/start
func (h *ContainerHandler) Start(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	// Get user info from session
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	uid, _ := userID.(int64)
	uname, _ := username.(string)

	if err := h.service.Start(c.Request.Context(), containerID, uid, uname); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		if strings.Contains(err.Error(), "already started") {
			c.JSON(http.StatusConflict, gin.H{"error": "container is already running"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "container started"})
}

// Stop stops a container.
// POST /api/containers/:id/stop?timeout=10
func (h *ContainerHandler) Stop(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	var timeout *int
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		var t int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &t); err == nil {
			timeout = &t
		}
	}

	// Get user info from session
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	uid, _ := userID.(int64)
	uname, _ := username.(string)

	if err := h.service.Stop(c.Request.Context(), containerID, timeout, uid, uname); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		if strings.Contains(err.Error(), "already stopped") || strings.Contains(err.Error(), "not running") {
			c.JSON(http.StatusConflict, gin.H{"error": "container is not running"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "container stopped"})
}

// Restart restarts a container.
// POST /api/containers/:id/restart?timeout=10
func (h *ContainerHandler) Restart(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	var timeout *int
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		var t int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &t); err == nil {
			timeout = &t
		}
	}

	// Get user info from session
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	uid, _ := userID.(int64)
	uname, _ := username.(string)

	if err := h.service.Restart(c.Request.Context(), containerID, timeout, uid, uname); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "container restarted"})
}

// Remove removes a container.
// DELETE /api/containers/:id?force=true
func (h *ContainerHandler) Remove(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	force := c.DefaultQuery("force", "false") == "true"

	// Get user info from session
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	uid, _ := userID.(int64)
	uname, _ := username.(string)

	if err := h.service.Remove(c.Request.Context(), containerID, force, uid, uname); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		if strings.Contains(err.Error(), "is running") {
			c.JSON(http.StatusConflict, gin.H{"error": "container is running, use force=true to remove"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "container removed"})
}

// StreamLogs streams container logs via Server-Sent Events.
// GET /api/containers/:id/logs?follow=true&tail=100&timestamps=true
func (h *ContainerHandler) StreamLogs(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	follow := c.DefaultQuery("follow", "true") == "true"
	tail := c.DefaultQuery("tail", "100")
	timestamps := c.DefaultQuery("timestamps", "false") == "true"
	since := c.DefaultQuery("since", "")

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()

	logReader, err := h.service.Logs(ctx, containerID, services.LogOptions{
		Follow:     follow,
		Tail:       tail,
		Timestamps: timestamps,
		Since:      since,
	})
	if err != nil {
		// Send error as SSE event
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err.Error())
		c.Writer.Flush()
		return
	}
	defer logReader.Close()

	// Stream logs line by line
	reader := bufio.NewReader(logReader)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Docker log stream has 8-byte header for multiplexed streams
			// Read header first
			header := make([]byte, 8)
			_, err := io.ReadFull(reader, header)
			if err != nil {
				if err == io.EOF {
					// Send end event
					_, _ = fmt.Fprintf(c.Writer, "event: end\ndata: stream ended\n\n")
					c.Writer.Flush()
					return
				}
				return
			}

			// Parse size from header (last 4 bytes, big endian)
			size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
			if size == 0 {
				continue
			}

			// Read the actual log line
			line := make([]byte, size)
			_, err = io.ReadFull(reader, line)
			if err != nil {
				return
			}

			// Send as SSE event
			lineStr := strings.TrimRight(string(line), "\n\r")
			data, _ := json.Marshal(map[string]interface{}{
				"stream": header[0], // 1=stdout, 2=stderr
				"line":   lineStr,
			})
			_, _ = fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", data)
			c.Writer.Flush()
		}
	}
}

// ExecCommand executes a command in a container (non-interactive).
// POST /api/containers/:id/exec
func (h *ContainerHandler) ExecCommand(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	var req struct {
		Cmd        []string `json:"cmd" binding:"required"`
		WorkingDir string   `json:"working_dir,omitempty"`
		User       string   `json:"user,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cmd is required"})
		return
	}

	// Get user info from session
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	uid, _ := userID.(int64)
	uname, _ := username.(string)

	result, err := h.service.Exec(c.Request.Context(), containerID, services.ExecConfig{
		Cmd:          req.Cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		WorkingDir:   req.WorkingDir,
		User:         req.User,
	}, uid, uname)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecInteractive handles WebSocket connections for interactive terminal.
// GET /api/containers/:id/terminal (WebSocket upgrade)
func (h *ContainerHandler) ExecInteractive(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	// Get shell and user from query params
	shell := c.DefaultQuery("shell", "/bin/sh")
	user := c.Query("user") // Empty means use container's default user

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	// Create exec with TTY
	execID, err := h.service.CreateExec(c.Request.Context(), containerID, services.ExecConfig{
		Cmd:          []string{shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		User:         user,
	})
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %s", err.Error())))
		return
	}

	// Attach to exec
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hijacked, err := h.service.AttachExec(ctx, execID)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %s", err.Error())))
		return
	}
	defer hijacked.Close()

	// Set up ping/pong for keep-alive
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var wg sync.WaitGroup
	wg.Add(3)

	// Ping ticker for keep-alive
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					cancel()
					return
				}
			}
		}
	}()

	// Docker -> WebSocket
	go func() {
		defer wg.Done()
		defer cancel()

		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := hijacked.Reader.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						return
					}
				}
			}
		}
	}()

	// WebSocket -> Docker
	go func() {
		defer wg.Done()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				_, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if _, err := hijacked.Conn.Write(data); err != nil {
					return
				}
			}
		}
	}()

	wg.Wait()
}

// CheckDocker checks if Docker is available.
// GET /api/containers/status
func (h *ContainerHandler) CheckDocker(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	available := h.service.IsDockerAvailable(ctx)

	c.JSON(http.StatusOK, gin.H{
		"available": available,
	})
}
