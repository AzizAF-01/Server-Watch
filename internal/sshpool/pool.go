package sshpool

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"serverwatch/internal/model"
	"serverwatch/internal/storage"
)

// SSHPool manages persistent SSH client connections and execution
type SSHPool struct {
	mu       sync.RWMutex
	clients  map[string]*ssh.Client
	dialMu   sync.Map // serverID -> *sync.Mutex (per-server dial lock)
	storage  *storage.SQLiteStorage
	keychain *storage.KeychainManager
}

func NewSSHPool(store *storage.SQLiteStorage, km *storage.KeychainManager) *SSHPool {
	return &SSHPool{
		clients:  make(map[string]*ssh.Client),
		storage:  store,
		keychain: km,
	}
}

func (p *SSHPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, client := range p.clients {
		_ = client.Close()
		delete(p.clients, id)
	}
}

// RemoveClient closes and evicts cached client
func (p *SSHPool) RemoveClient(serverID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.clients[serverID]; ok {
		_ = c.Close()
		delete(p.clients, serverID)
	}
}

// buildAuthMethods prepares SSH authentication mechanics
func (p *SSHPool) buildAuthMethods(cfg *model.ServerConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	switch cfg.AuthType {
	case model.AuthTypePassword:
		pwd := cfg.Password
		if pwd == "" && cfg.ID != "" {
			pwd, _ = p.keychain.GetPassword(cfg.ID)
		}
		if pwd == "" {
			return nil, errors.New("password authentication selected but password is empty")
		}
		methods = append(methods, ssh.Password(pwd))

	case model.AuthTypeKey:
		if cfg.KeyPath == "" {
			return nil, errors.New("private key path is required")
		}
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		passphrase := cfg.Passphrase
		if passphrase == "" && cfg.ID != "" {
			passphrase, _ = p.keychain.GetPassphrase(cfg.ID)
		}

		var signer ssh.Signer
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))

	case model.AuthTypeAgent:
		agentConn, err := connectSSHAgent()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
		}
		agentClient := agent.NewClient(agentConn)
		signers, err := agentClient.Signers()
		if err != nil || len(signers) == 0 {
			return nil, errors.New("no identities found in SSH agent")
		}
		methods = append(methods, ssh.PublicKeys(signers...))

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}

	return methods, nil
}

// connectSSHAgent detects agent on Windows pipe or Unix socket
func connectSSHAgent() (net.Conn, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock != "" {
		return net.Dial("unix", sock)
	}
	// Windows named pipe for OpenSSH agent
	return net.Dial("unix", `\\.\pipe\openssh-ssh-agent`)
}

// TestConnection dials host, validates host key, and runs light diagnostics
func (p *SSHPool) TestConnection(cfg *model.ServerConfig) (*model.ConnectionTestResult, error) {
	start := time.Now()
	res := &model.ConnectionTestResult{
		Success: false,
	}

	auths, err := p.buildAuthMethods(cfg)
	if err != nil {
		res.Error = err.Error()
		return res, nil
	}

	var capturedKey ssh.PublicKey
	clientConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			capturedKey = key
			return nil
		},
		Timeout: 7 * time.Second,
	}

	targetAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client, err := p.dialWithJumpHost(clientConfig, targetAddr, cfg.JumpHostID)
	if err != nil {
		res.Error = fmt.Sprintf("SSH connection failed: %v", err)
		return res, nil
	}
	defer client.Close()

	res.LatencyMs = time.Since(start).Milliseconds()

	if capturedKey != nil {
		res.HostKeyType = capturedKey.Type()
		res.HostKeyFingerprint = ssh.FingerprintSHA256(capturedKey)
	}

	// Run light diagnostic command
	sess, err := client.NewSession()
	if err == nil {
		defer sess.Close()
		var out bytes.Buffer
		sess.Stdout = &out
		cmd := "uname -srm; which systemctl 2>/dev/null; cat /etc/os-release 2>/dev/null | grep -E '^(PRETTY_NAME)=' | head -1"
		if err := sess.Run(cmd); err == nil {
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					res.OSInfo = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				} else if strings.Contains(line, "Linux") {
					res.KernelInfo = line
				} else if strings.Contains(line, "systemctl") {
					res.HasSystemd = true
				}
			}
		}
	}

	res.Success = true
	return res, nil
}

// dialWithJumpHost handles direct connection or proxy jump
func (p *SSHPool) dialWithJumpHost(targetConfig *ssh.ClientConfig, targetAddr string, jumpHostID string) (*ssh.Client, error) {
	if jumpHostID == "" {
		return ssh.Dial("tcp", targetAddr, targetConfig)
	}

	// Fetch Jump Host config from storage
	jumpServer, err := p.storage.GetServer(jumpHostID)
	if err != nil {
		return nil, fmt.Errorf("jump host %s not found: %w", jumpHostID, err)
	}

	jumpCfg := &model.ServerConfig{
		ID:       jumpServer.ID,
		Host:     jumpServer.Host,
		Port:     jumpServer.Port,
		User:     jumpServer.User,
		AuthType: jumpServer.AuthType,
		KeyPath:  jumpServer.KeyPath,
	}

	jumpAuths, err := p.buildAuthMethods(jumpCfg)
	if err != nil {
		return nil, fmt.Errorf("jump host authentication failed: %w", err)
	}

	jumpClientConfig := &ssh.ClientConfig{
		User: jumpServer.User,
		Auth: jumpAuths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			fp := ssh.FingerprintSHA256(key)
			if jumpServer.HostKeyFingerprint != "" && jumpServer.TrustedHostKey {
				if subtle.ConstantTimeCompare([]byte(fp), []byte(jumpServer.HostKeyFingerprint)) != 1 {
					return fmt.Errorf("JUMP HOST KEY MISMATCH! Expected %s, got %s", jumpServer.HostKeyFingerprint, fp)
				}
				return nil
			}
			if jumpServer.HostKeyFingerprint == "" {
				jumpServer.HostKeyFingerprint = fp
				jumpServer.TrustedHostKey = true
				if p.storage != nil {
					_ = p.storage.SaveServer(jumpServer)
				}
				return nil
			}
			if !jumpServer.TrustedHostKey {
				return fmt.Errorf("jump host key belum diverifikasi/dipercaya (fingerprint: %s)", fp)
			}
			return nil
		},
		Timeout: 7 * time.Second,
	}

	jumpAddr := fmt.Sprintf("%s:%d", jumpServer.Host, jumpServer.Port)
	jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial jump host %s: %w", jumpAddr, err)
	}

	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("failed to dial target via jump host: %w", err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetConfig)
	if err != nil {
		conn.Close()
		jumpClient.Close()
		return nil, fmt.Errorf("failed to create client conn via jump host: %w", err)
	}

	return ssh.NewClient(ncc, chans, reqs), nil
}

// GetOrCreateClient returns active persistent client or creates one
func (p *SSHPool) GetOrCreateClient(srv *model.Server) (*ssh.Client, error) {
	p.mu.RLock()
	client, exists := p.clients[srv.ID]
	p.mu.RUnlock()

	if exists {
		// Test if client is still alive with a strict 3-second timeout
		keepaliveDone := make(chan error, 1)
		go func(c *ssh.Client) {
			_, _, err := c.SendRequest("keepalive@serverwatch", true, nil)
			keepaliveDone <- err
		}(client)

		select {
		case err := <-keepaliveDone:
			if err == nil {
				return client, nil
			}
		case <-time.After(3 * time.Second):
		}
		// If dead or timed out, remove and re-dial
		p.RemoveClient(srv.ID)
	}

	// Use per-server dial lock so an offline/slow server cannot stall other servers
	val, _ := p.dialMu.LoadOrStore(srv.ID, &sync.Mutex{})
	srvMu := val.(*sync.Mutex)
	srvMu.Lock()
	defer srvMu.Unlock()

	// Double check after acquiring per-server lock
	p.mu.RLock()
	client, exists = p.clients[srv.ID]
	p.mu.RUnlock()
	if exists {
		return client, nil
	}

	cfg := &model.ServerConfig{
		ID:         srv.ID,
		Host:       srv.Host,
		Port:       srv.Port,
		User:       srv.User,
		AuthType:   srv.AuthType,
		KeyPath:    srv.KeyPath,
		JumpHostID: srv.JumpHostID,
	}

	auths, err := p.buildAuthMethods(cfg)
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User: srv.User,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			fp := ssh.FingerprintSHA256(key)
			if srv.HostKeyFingerprint != "" && srv.TrustedHostKey {
				if subtle.ConstantTimeCompare([]byte(fp), []byte(srv.HostKeyFingerprint)) != 1 {
					return fmt.Errorf("HOST KEY MISMATCH! Expected %s, got %s", srv.HostKeyFingerprint, fp)
				}
				return nil
			}
			if srv.HostKeyFingerprint == "" {
				srv.HostKeyFingerprint = fp
				srv.TrustedHostKey = true
				if p.storage != nil {
					_ = p.storage.SaveServer(srv)
				}
				return nil
			}
			if !srv.TrustedHostKey {
				return fmt.Errorf("host key belum diverifikasi/dipercaya (fingerprint: %s)", fp)
			}
			return nil
		},
		Timeout: 10 * time.Second,
	}

	targetAddr := fmt.Sprintf("%s:%d", srv.Host, srv.Port)
	newClient, err := p.dialWithJumpHost(clientConfig, targetAddr, srv.JumpHostID)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.clients[srv.ID] = newClient
	p.mu.Unlock()

	return newClient, nil
}

// boundedBuffer limits the maximum bytes stored in memory to prevent memory exhaustion (SEC-02).
type boundedBuffer struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	maxCap int
}

func newBoundedBuffer(maxCap int) *boundedBuffer {
	return &boundedBuffer{maxCap: maxCap}
}

func (b *boundedBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	currLen := b.buf.Len()
	if currLen >= b.maxCap {
		return len(p), nil
	}
	remaining := b.maxCap - currLen
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// ExecuteCommand executes a single command safely with timeout and output limit (SEC-02)
func (p *SSHPool) ExecuteCommand(ctx context.Context, srv *model.Server, cmd string) (string, error) {
	client, err := p.GetOrCreateClient(srv)
	if err != nil {
		return "", err
	}

	sess, err := client.NewSession()
	if err != nil {
		p.RemoveClient(srv.ID)
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer sess.Close()

	// SEC-02: Limit maximum stdout/stderr to 1MB/512KB per command to prevent memory exhaustion
	stdoutBuf := newBoundedBuffer(1024 * 1024)
	stderrBuf := newBoundedBuffer(512 * 1024)
	sess.Stdout = stdoutBuf
	sess.Stderr = stderrBuf

	done := make(chan error, 1)
	go func() {
		done <- sess.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		p.RemoveClient(srv.ID)
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("command execution error: %v, stderr: %s", err, stderrBuf.String())
		}
		return stdoutBuf.String(), nil
	}
}

// StreamRemoteFile streams remote file content directly to dst io.Writer (SEC-02, SEC-15).
// Unlike ExecuteCommand, this bypasses the 1MB memory buffer limit to safely transfer
// large backup archives without truncation or out-of-memory crashes.
func (p *SSHPool) StreamRemoteFile(ctx context.Context, srv *model.Server, remotePath string, dst io.Writer) error {
	client, err := p.GetOrCreateClient(srv)
	if err != nil {
		return err
	}

	sess, err := client.NewSession()
	if err != nil {
		p.RemoveClient(srv.ID)
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer sess.Close()

	escapedPath := strings.ReplaceAll(remotePath, "'", "'\\''")
	cmd := fmt.Sprintf("cat '%s'", escapedPath)

	sess.Stdout = dst
	stderrBuf := newBoundedBuffer(512 * 1024)
	sess.Stderr = stderrBuf

	done := make(chan error, 1)
	go func() {
		done <- sess.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		p.RemoveClient(srv.ID)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("streaming error: %v (stderr: %s)", err, strings.TrimSpace(stderrBuf.String()))
		}
		return nil
	}
}

