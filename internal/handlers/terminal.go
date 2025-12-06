package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

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
	sessions sync.Map // map[string]*terminalSession
}

type terminalSession struct {
	ws   *websocket.Conn
	ptmx *os.File
	cmd  *exec.Cmd
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
	defer ws.Close()

	log.Printf("[Terminal] User %s connected", user.Username)

	// Start shell with PTY
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		ws.WriteMessage(websocket.TextMessage, []byte("Failed to start terminal\r\n"))
		return
	}
	defer ptmx.Close()

	_ = &terminalSession{
		ws:   ws,
		ptmx: ptmx,
		cmd:  cmd,
	}

	// Handle terminal I/O
	var wg sync.WaitGroup
	wg.Add(2)

	// Read from PTY and send to WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error: %v", err)
				}
				return
			}
			if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}()

	// Read from WebSocket and write to PTY
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				if _, err := ptmx.Write(msg); err != nil {
					log.Printf("PTY write error: %v", err)
					return
				}
			}
		}
	}()

	wg.Wait()
	cmd.Process.Kill()
	log.Printf("[Terminal] User %s disconnected", user.Username)
}
