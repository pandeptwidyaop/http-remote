package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Add proper origin validation
	},
}

// TerminalHandler handles interactive terminal sessions over WebSocket.
type TerminalHandler struct {
	cfg          *config.TerminalConfig
	auditService *services.AuditService
}

// NewTerminalHandler creates a new TerminalHandler instance.
func NewTerminalHandler(cfg *config.TerminalConfig, auditService *services.AuditService) *TerminalHandler {
	return &TerminalHandler{cfg: cfg, auditService: auditService}
}

// HandleWebSocket handles WebSocket terminal connections.
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	// Check if terminal is enabled
	if !h.cfg.IsEnabled() {
		c.JSON(403, gin.H{"error": "terminal is disabled"})
		return
	}

	// Verify authentication
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

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer func() { _ = ws.Close() }()

	log.Printf("[Terminal] User %s connected", user.Username)

	// Log terminal session start
	sessionStart := time.Now()
	h.auditService.Log(services.AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "terminal_connect",
		ResourceType: "terminal",
		ResourceID:   "session",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"shell": h.cfg.Shell,
		},
	})

	// Start shell with PTY using config
	cmd := exec.Command(h.cfg.Shell, h.cfg.Args...)

	// Build environment variables
	env := append(os.Environ(), "TERM=xterm-256color")
	env = append(env, h.cfg.Env...)
	cmd.Env = env

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte("Failed to start terminal\r\n"))
		return
	}
	defer func() { _ = ptmx.Close() }()


	// Channel to signal connection close
	done := make(chan struct{})

	// Read from PTY and send to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("PTY read error: %v", err)
					}
					close(done)
					return
				}
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					log.Printf("WebSocket write error: %v", err)
					close(done)
					return
				}
			}
		}
	}()

	// Read from WebSocket and write to PTY
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
				if _, err := ptmx.Write(msg); err != nil {
					log.Printf("PTY write error: %v", err)
					goto cleanup
				}
			}
		}
	}

cleanup:
	_ = cmd.Process.Kill()

	// Log terminal session end
	sessionDuration := time.Since(sessionStart)
	h.auditService.Log(services.AuditLog{
		UserID:       &user.ID,
		Username:     user.Username,
		Action:       "terminal_disconnect",
		ResourceType: "terminal",
		ResourceID:   "session",
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Details: map[string]interface{}{
			"duration_seconds": sessionDuration.Seconds(),
		},
	})

	log.Printf("[Terminal] User %s disconnected (duration: %v)", user.Username, sessionDuration)
}
