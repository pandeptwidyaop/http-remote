package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/pandeptwidyaop/http-remote/internal/middleware"
	"github.com/pandeptwidyaop/http-remote/internal/models"
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
}

// NewTerminalHandler creates a new TerminalHandler instance.
func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{}
}

// HandleWebSocket handles WebSocket terminal connections.
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
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

	// Start shell with PTY
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

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
}
