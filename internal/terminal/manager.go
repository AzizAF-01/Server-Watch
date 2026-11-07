package terminal

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
	"serverwatch/internal/storage"
)

type windowChangeMsg struct {
	Columns uint32
	Rows    uint32
	Width   uint32
	Height  uint32
}

type Session struct {
	ID        string
	ServerID  string
	Session   *ssh.Session
	Stdin     io.WriteCloser
	CreatedAt time.Time
	closed    bool
	mu        sync.Mutex
}

type Manager struct {
	pool     *sshpool.SSHPool
	storage  *storage.SQLiteStorage
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewManager(pool *sshpool.SSHPool, store *storage.SQLiteStorage) *Manager {
	return &Manager{
		pool:     pool,
		storage:  store,
		sessions: make(map[string]*Session),
	}
}

// StartSession initiates an interactive PTY shell session on the remote server
func (m *Manager) StartSession(ctx context.Context, srv *model.Server, cols int, rows int, onData func(sessionID, data string), onExit func(sessionID string)) (string, error) {
	if m.pool == nil {
		return "", fmt.Errorf("ssh pool not initialized")
	}

	client, err := m.pool.GetOrCreateClient(srv)
	if err != nil {
		return "", fmt.Errorf("gagal menghubungkan ke server SSH: %w", err)
	}

	sess, err := client.NewSession()
	if err != nil {
		m.pool.RemoveClient(srv.ID)
		return "", fmt.Errorf("gagal membuat SSH session: %w", err)
	}

	if cols <= 0 {
		cols = 100
	}
	if rows <= 0 {
		rows = 30
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = sess.Close()
		return "", fmt.Errorf("gagal mengalokasikan PTY remote: %w", err)
	}

	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		return "", fmt.Errorf("gagal membuka stdin pipe: %w", err)
	}

	stdoutPipe, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		return "", fmt.Errorf("gagal membuka stdout pipe: %w", err)
	}

	stderrPipe, err := sess.StderrPipe()
	if err != nil {
		_ = sess.Close()
		return "", fmt.Errorf("gagal membuka stderr pipe: %w", err)
	}

	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		return "", fmt.Errorf("gagal memulai remote interactive shell: %w", err)
	}

	sessionID := uuid.New().String()
	tSession := &Session{
		ID:        sessionID,
		ServerID:  srv.ID,
		Session:   sess,
		Stdin:     stdinPipe,
		CreatedAt: time.Now(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = tSession
	m.mu.Unlock()

	var exitOnce sync.Once
	triggerExit := func() {
		exitOnce.Do(func() {
			_ = m.CloseSession(sessionID)
			if onExit != nil {
				onExit(sessionID)
			}
		})
	}

	pump := func(r io.Reader) {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 && onData != nil {
				onData(sessionID, string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}

	go func() {
		pump(stdoutPipe)
		triggerExit()
	}()

	go func() {
		pump(stderrPipe)
	}()

	return sessionID, nil
}

// Write sends input keystrokes or data into the remote terminal stdin
func (m *Manager) Write(sessionID string, data string) error {
	m.mu.RLock()
	s, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("terminal session %s tidak ditemukan atau sudah ditutup", sessionID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.Stdin == nil {
		return fmt.Errorf("terminal session %s sudah tertutup", sessionID)
	}

	_, err := s.Stdin.Write([]byte(data))
	return err
}

// Resize updates terminal window dimensions on the remote PTY
func (m *Manager) Resize(sessionID string, cols int, rows int) error {
	m.mu.RLock()
	s, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists || s.Session == nil {
		return nil
	}

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	msg := windowChangeMsg{
		Columns: uint32(cols),
		Rows:    uint32(rows),
	}

	_, err := s.Session.SendRequest("window-change", false, ssh.Marshal(msg))
	return err
}

// CloseSession terminates an active interactive terminal session
func (m *Manager) CloseSession(sessionID string) error {
	m.mu.Lock()
	s, exists := m.sessions[sessionID]
	if exists {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if !exists {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.Stdin != nil {
		_ = s.Stdin.Close()
	}
	if s.Session != nil {
		_ = s.Session.Close()
	}

	return nil
}

// CloseAll terminates all active sessions upon application shutdown
func (m *Manager) CloseAll() {
	m.mu.Lock()
	active := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		active = append(active, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range active {
		s.mu.Lock()
		if !s.closed {
			s.closed = true
			if s.Stdin != nil {
				_ = s.Stdin.Close()
			}
			if s.Session != nil {
				_ = s.Session.Close()
			}
		}
		s.mu.Unlock()
	}
}
