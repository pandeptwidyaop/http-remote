package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/pandeptwidyaop/http-remote/internal/config"
)

// TerminalSession represents a persistent terminal session.
type TerminalSession struct {
	ID           string
	UserID       int64
	Username     string
	Shell        string
	CreatedAt    time.Time
	LastActivity time.Time

	cmd       *exec.Cmd
	ptmx      *os.File
	mu        sync.Mutex
	buffer    *RingBuffer
	clients   map[string]chan []byte
	clientsMu sync.RWMutex
	done      chan struct{}
	closed    bool
}

// RingBuffer is a circular buffer for storing terminal output history.
type RingBuffer struct {
	data  []byte
	size  int
	start int
	end   int
	full  bool
	mu    sync.Mutex
}

// NewRingBuffer creates a new ring buffer with the given size.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the ring buffer.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, b := range p {
		r.data[r.end] = b
		r.end = (r.end + 1) % r.size
		if r.full {
			r.start = (r.start + 1) % r.size
		}
		if r.end == r.start {
			r.full = true
		}
	}
	return len(p), nil
}

// ReadAll returns all data currently in the buffer.
func (r *RingBuffer) ReadAll() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.start == r.end && !r.full {
		return nil
	}

	var result []byte
	if r.full || r.end < r.start {
		result = append(result, r.data[r.start:]...)
		result = append(result, r.data[:r.end]...)
	} else {
		result = append(result, r.data[r.start:r.end]...)
	}
	return result
}

// TerminalSessionManager manages persistent terminal sessions.
type TerminalSessionManager struct {
	sessions      map[string]*TerminalSession
	userSessions  map[int64][]string // userID -> sessionIDs
	mu            sync.RWMutex
	cfg           *config.TerminalConfig
	maxSessions   int
	bufferSize    int
	sessionTTL    time.Duration
	cleanupTicker *time.Ticker
	done          chan struct{}
}

// NewTerminalSessionManager creates a new session manager.
func NewTerminalSessionManager(cfg *config.TerminalConfig) *TerminalSessionManager {
	m := &TerminalSessionManager{
		sessions:     make(map[string]*TerminalSession),
		userSessions: make(map[int64][]string),
		cfg:          cfg,
		maxSessions:  10,             // Max sessions per user
		bufferSize:   64 * 1024,      // 64KB buffer per session
		sessionTTL:   24 * time.Hour, // Sessions expire after 24 hours of inactivity
		done:         make(chan struct{}),
	}

	// Start cleanup goroutine
	m.cleanupTicker = time.NewTicker(5 * time.Minute)
	go m.cleanupLoop()

	return m
}

// cleanupLoop periodically cleans up expired sessions.
func (m *TerminalSessionManager) cleanupLoop() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.cleanupExpiredSessions()
		case <-m.done:
			return
		}
	}
}

// cleanupExpiredSessions removes sessions that have been inactive too long.
func (m *TerminalSessionManager) cleanupExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.LastActivity) > m.sessionTTL {
			log.Printf("[TerminalSession] Cleaning up expired session %s for user %s", id, session.Username)
			session.Close()
			delete(m.sessions, id)
			m.removeUserSession(session.UserID, id)
		}
	}
}

// removeUserSession removes a session ID from the user's session list.
func (m *TerminalSessionManager) removeUserSession(userID int64, sessionID string) {
	sessions := m.userSessions[userID]
	for i, id := range sessions {
		if id == sessionID {
			m.userSessions[userID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
}

// CreateSession creates a new persistent terminal session.
func (m *TerminalSessionManager) CreateSession(userID int64, username string) (*TerminalSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check session limit per user
	if len(m.userSessions[userID]) >= m.maxSessions {
		return nil, fmt.Errorf("maximum sessions (%d) reached for user", m.maxSessions)
	}

	// Generate session ID
	sessionID := fmt.Sprintf("term-%d-%d", userID, time.Now().UnixNano())

	// Create the session
	session := &TerminalSession{
		ID:           sessionID,
		UserID:       userID,
		Username:     username,
		Shell:        m.cfg.Shell,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		buffer:       NewRingBuffer(m.bufferSize),
		clients:      make(map[string]chan []byte),
		done:         make(chan struct{}),
	}

	// Start shell with PTY
	// #nosec G204 - shell execution is expected behavior for terminal service
	cmd := exec.Command(m.cfg.Shell, m.cfg.Args...)
	env := append(os.Environ(), "TERM=xterm-256color")
	env = append(env, m.cfg.Env...)
	cmd.Env = env

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	session.cmd = cmd
	session.ptmx = ptmx

	// Start reading from PTY
	go session.readLoop()

	// Store session
	m.sessions[sessionID] = session
	m.userSessions[userID] = append(m.userSessions[userID], sessionID)

	log.Printf("[TerminalSession] Created session %s for user %s", sessionID, username)

	return session, nil
}

// GetSession returns a session by ID.
func (m *TerminalSessionManager) GetSession(sessionID string) (*TerminalSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	return session, ok
}

// GetUserSessions returns all sessions for a user.
func (m *TerminalSessionManager) GetUserSessions(userID int64) []*TerminalSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionIDs := m.userSessions[userID]
	sessions := make([]*TerminalSession, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if session, ok := m.sessions[id]; ok {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// CloseSession closes and removes a session.
func (m *TerminalSessionManager) CloseSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	session.Close()
	delete(m.sessions, sessionID)
	m.removeUserSession(session.UserID, sessionID)

	log.Printf("[TerminalSession] Closed session %s", sessionID)

	return nil
}

// Shutdown closes all sessions and stops the manager.
func (m *TerminalSessionManager) Shutdown() {
	close(m.done)
	m.cleanupTicker.Stop()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		session.Close()
	}
	m.sessions = make(map[string]*TerminalSession)
	m.userSessions = make(map[int64][]string)
}

// readLoop reads from PTY and broadcasts to all clients.
func (s *TerminalSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.done:
			return
		default:
			n, err := s.ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("[TerminalSession] Read error for session %s: %v", s.ID, err)
				}
				s.Close()
				return
			}

			data := make([]byte, n)
			copy(data, buf[:n])

			// Store in buffer for replay
			_, _ = s.buffer.Write(data)

			// Update activity
			s.mu.Lock()
			s.LastActivity = time.Now()
			s.mu.Unlock()

			// Broadcast to all clients
			s.broadcast(data)
		}
	}
}

// broadcast sends data to all connected clients.
func (s *TerminalSession) broadcast(data []byte) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, ch := range s.clients {
		select {
		case ch <- data:
		default:
			// Client channel is full, skip
		}
	}
}

// Subscribe adds a client to receive output from this session.
func (s *TerminalSession) Subscribe(clientID string) (<-chan []byte, []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	ch := make(chan []byte, 256)
	s.clients[clientID] = ch

	// Return channel and buffer history for replay
	history := s.buffer.ReadAll()

	log.Printf("[TerminalSession] Client %s subscribed to session %s", clientID, s.ID)

	return ch, history
}

// Unsubscribe removes a client from this session.
func (s *TerminalSession) Unsubscribe(clientID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if ch, ok := s.clients[clientID]; ok {
		close(ch)
		delete(s.clients, clientID)
		log.Printf("[TerminalSession] Client %s unsubscribed from session %s", clientID, s.ID)
	}
}

// Write writes data to the terminal.
func (s *TerminalSession) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	s.LastActivity = time.Now()
	_, err := s.ptmx.Write(data)
	return err
}

// Close closes the terminal session.
func (s *TerminalSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	close(s.done)

	// Close all client channels
	s.clientsMu.Lock()
	for id, ch := range s.clients {
		close(ch)
		delete(s.clients, id)
	}
	s.clientsMu.Unlock()

	// Kill the process
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}

	// Close PTY
	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
}

// IsClosed returns whether the session is closed.
func (s *TerminalSession) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// ClientCount returns the number of connected clients.
func (s *TerminalSession) ClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

// SessionInfo returns information about a session (for API responses).
type SessionInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	ClientCount  int       `json:"client_count"`
	IsActive     bool      `json:"is_active"`
}

// Info returns session information.
func (s *TerminalSession) Info() SessionInfo {
	return SessionInfo{
		ID:           s.ID,
		Name:         fmt.Sprintf("Session %s", s.ID[len(s.ID)-8:]),
		CreatedAt:    s.CreatedAt,
		LastActivity: s.LastActivity,
		ClientCount:  s.ClientCount(),
		IsActive:     !s.IsClosed(),
	}
}
