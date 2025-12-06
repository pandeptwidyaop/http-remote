package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// TerminalHandler handles interactive terminal sessions over WebSocket.
type TerminalHandler struct {
	cfg            *config.TerminalConfig
	auditService   *services.AuditService
	sessionManager *services.TerminalSessionManager
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

// NewTerminalHandler creates a new TerminalHandler instance.
func NewTerminalHandler(cfg *config.TerminalConfig, auditService *services.AuditService, allowedOrigins []string) *TerminalHandler {
	h := &TerminalHandler{
		cfg:            cfg,
		auditService:   auditService,
		sessionManager: services.NewTerminalSessionManager(cfg),
		allowedOrigins: allowedOrigins,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	return h
}

// checkOrigin validates WebSocket connection origins to prevent CSRF attacks.
func (h *TerminalHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return len(h.allowedOrigins) == 0
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		log.Printf("[Terminal] Invalid origin URL: %s", origin)
		return false
	}

	if len(h.allowedOrigins) == 0 {
		host := r.Host
		if originURL.Host == host {
			return true
		}
		originHost := originURL.Hostname()
		requestHost := strings.Split(host, ":")[0]
		return originHost == requestHost
	}

	for _, allowed := range h.allowedOrigins {
		if strings.HasPrefix(allowed, "*.") {
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(originURL.Host, suffix) || originURL.Host == strings.TrimPrefix(suffix, ".") {
				return true
			}
		} else if allowed == origin || allowed == originURL.Host {
			return true
		}
	}

	log.Printf("[Terminal] Origin not allowed: %s (allowed: %v)", origin, h.allowedOrigins)
	return false
}

// ListSessions returns all terminal sessions for the current user.
func (h *TerminalHandler) ListSessions(c *gin.Context) {
	if !h.cfg.IsEnabled() {
		c.JSON(http.StatusForbidden, gin.H{"error": "terminal is disabled"})
		return
	}

	userObj, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	sessions := h.sessionManager.GetUserSessions(user.ID)
	sessionInfos := make([]services.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		if !s.IsClosed() {
			sessionInfos = append(sessionInfos, s.Info())
		}
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessionInfos})
}

// CreateSession creates a new persistent terminal session.
func (h *TerminalHandler) CreateSession(c *gin.Context) {
	if !h.cfg.IsEnabled() {
		c.JSON(http.StatusForbidden, gin.H{"error": "terminal is disabled"})
		return
	}

	userObj, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	session, err := h.sessionManager.CreateSession(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log session creation
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_session_create",
			ResourceType: "terminal",
			ResourceID:   session.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"shell": h.cfg.Shell,
			},
		})
	}

	c.JSON(http.StatusCreated, gin.H{"session": session.Info()})
}

// CloseSession closes a persistent terminal session.
func (h *TerminalHandler) CloseSession(c *gin.Context) {
	if !h.cfg.IsEnabled() {
		c.JSON(http.StatusForbidden, gin.H{"error": "terminal is disabled"})
		return
	}

	userObj, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	sessionID := c.Param("session_id")

	// Verify session belongs to user
	session, ok := h.sessionManager.GetSession(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.sessionManager.CloseSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log session close
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_session_close",
			ResourceType: "terminal",
			ResourceID:   sessionID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "session closed"})
}

// WebSocketMessage represents a message from the WebSocket client.
type WebSocketMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Data      string `json:"data,omitempty"`
}

// HandleWebSocket handles WebSocket terminal connections with session support.
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	if !h.cfg.IsEnabled() {
		c.JSON(403, gin.H{"error": "terminal is disabled"})
		return
	}

	userObj, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(500, gin.H{"error": "invalid user context"})
		return
	}

	// Check for session_id query parameter (for persistent sessions)
	sessionID := c.Query("session_id")
	persistent := c.Query("persistent") == "true"

	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer func() { _ = ws.Close() }()

	log.Printf("[Terminal] User %s connected (persistent: %v, session: %s)", user.Username, persistent, sessionID)

	if persistent {
		h.handlePersistentSession(ws, user, sessionID, c)
	} else {
		h.handleEphemeralSession(ws, user, c)
	}
}

// handlePersistentSession handles a WebSocket connection to a persistent session.
func (h *TerminalHandler) handlePersistentSession(ws *websocket.Conn, user *models.User, sessionID string, c *gin.Context) {
	var session *services.TerminalSession
	var ok bool

	// If sessionID provided, try to attach to existing session
	if sessionID != "" {
		session, ok = h.sessionManager.GetSession(sessionID)
		if !ok || session.UserID != user.ID {
			_ = ws.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[1;31mSession not found or access denied\x1b[0m\r\n"))
			return
		}
	} else {
		// Create new persistent session
		var err error
		session, err = h.sessionManager.CreateSession(user.ID, user.Username)
		if err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n\x1b[1;31mFailed to create session: %s\x1b[0m\r\n", err.Error())))
			return
		}

		// Log session creation
		if h.auditService != nil {
			_ = h.auditService.Log(services.AuditLog{
				UserID:       &user.ID,
				Username:     user.Username,
				Action:       "terminal_session_create",
				ResourceType: "terminal",
				ResourceID:   session.ID,
				IPAddress:    c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
			})
		}
	}

	// Send session info to client
	sessionInfo, _ := json.Marshal(map[string]interface{}{
		"type":    "session_info",
		"session": session.Info(),
	})
	_ = ws.WriteMessage(websocket.TextMessage, sessionInfo)

	// Generate unique client ID
	clientID := fmt.Sprintf("client-%d-%d", user.ID, time.Now().UnixNano())

	// Subscribe to session output
	outputCh, history := session.Subscribe(clientID)
	defer session.Unsubscribe(clientID)

	// Send history (replay buffer)
	if len(history) > 0 {
		_ = ws.WriteMessage(websocket.BinaryMessage, history)
	}

	// Log connection
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_connect",
			ResourceType: "terminal",
			ResourceID:   session.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

	sessionStart := time.Now()
	done := make(chan struct{})

	// Read from session and send to WebSocket
	go func() {
		for {
			select {
			case <-done:
				return
			case data, ok := <-outputCh:
				if !ok {
					close(done)
					return
				}
				if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}()

	// Read from WebSocket and send to session
	for {
		select {
		case <-done:
			goto cleanup
		default:
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				goto cleanup
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				if err := session.Write(msg); err != nil {
					log.Printf("Session write error: %v", err)
					goto cleanup
				}
			}
		}
	}

cleanup:
	close(done)

	// Log disconnect
	sessionDuration := time.Since(sessionStart)
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_disconnect",
			ResourceType: "terminal",
			ResourceID:   session.ID,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"duration_seconds": sessionDuration.Seconds(),
				"session_active":   !session.IsClosed(),
			},
		})
	}

	log.Printf("[Terminal] User %s disconnected from session %s (duration: %v, session still active: %v)",
		user.Username, session.ID, sessionDuration, !session.IsClosed())
}

// handleEphemeralSession handles a traditional one-off terminal session.
func (h *TerminalHandler) handleEphemeralSession(ws *websocket.Conn, user *models.User, c *gin.Context) {
	log.Printf("[Terminal] User %s connected (ephemeral)", user.Username)

	sessionStart := time.Now()
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_connect",
			ResourceType: "terminal",
			ResourceID:   "ephemeral",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"shell": h.cfg.Shell,
			},
		})
	}

	// Create ephemeral session using session manager
	session, err := h.sessionManager.CreateSession(user.ID, user.Username)
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n\x1b[1;31mFailed to start terminal: %s\x1b[0m\r\n", err.Error())))
		return
	}

	// Mark for cleanup when WebSocket closes
	defer func() { _ = h.sessionManager.CloseSession(session.ID) }()

	clientID := fmt.Sprintf("ephemeral-%d-%d", user.ID, time.Now().UnixNano())
	outputCh, _ := session.Subscribe(clientID)
	defer session.Unsubscribe(clientID)

	done := make(chan struct{})

	// Read from session and send to WebSocket
	go func() {
		for {
			select {
			case <-done:
				return
			case data, ok := <-outputCh:
				if !ok {
					return
				}
				if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
					return
				}
			}
		}
	}()

	// Read from WebSocket and write to session
	for {
		select {
		case <-done:
			goto cleanup
		default:
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				goto cleanup
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				if err := session.Write(msg); err != nil {
					log.Printf("Session write error: %v", err)
					goto cleanup
				}
			}
		}
	}

cleanup:
	close(done)

	sessionDuration := time.Since(sessionStart)
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_disconnect",
			ResourceType: "terminal",
			ResourceID:   "ephemeral",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"duration_seconds": sessionDuration.Seconds(),
			},
		})
	}

	log.Printf("[Terminal] User %s disconnected (ephemeral, duration: %v)", user.Username, sessionDuration)
}
