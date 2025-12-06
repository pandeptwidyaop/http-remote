package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
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
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

// NewTerminalHandler creates a new TerminalHandler instance.
func NewTerminalHandler(cfg *config.TerminalConfig, auditService *services.AuditService, allowedOrigins []string) *TerminalHandler {
	h := &TerminalHandler{
		cfg:            cfg,
		auditService:   auditService,
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
		// No origin header - could be same-origin request or non-browser client
		// For security, we'll allow it only if there are no allowed origins configured
		// (backward compatibility for local development)
		return len(h.allowedOrigins) == 0
	}

	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		log.Printf("[Terminal] Invalid origin URL: %s", origin)
		return false
	}

	// If no allowed origins configured, allow same-origin only
	if len(h.allowedOrigins) == 0 {
		host := r.Host
		// Compare origin host with request host
		if originURL.Host == host {
			return true
		}
		// Also check without port for flexibility
		originHost := originURL.Hostname()
		requestHost := strings.Split(host, ":")[0]
		return originHost == requestHost
	}

	// Check against allowed origins list
	for _, allowed := range h.allowedOrigins {
		// Support wildcard matching (e.g., "*.example.com")
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
	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer func() { _ = ws.Close() }()

	log.Printf("[Terminal] User %s connected", user.Username)

	// Audit log terminal connection
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_connect",
			ResourceType: "terminal",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

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
	log.Printf("[Terminal] User %s disconnected", user.Username)

	// Audit log terminal disconnect
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "terminal_disconnect",
			ResourceType: "terminal",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}
}
