package terminal

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestWindowChangeMsgMarshaling(t *testing.T) {
	msg := windowChangeMsg{
		Columns: 120,
		Rows:    35,
		Width:   0,
		Height:  0,
	}
	bytes := ssh.Marshal(msg)
	if len(bytes) != 16 {
		t.Fatalf("expected 16 bytes for RFC 4254 window-change message, got %d", len(bytes))
	}
}

func TestManagerSessionBookkeeping(t *testing.T) {
	mgr := NewManager(nil, nil)
	if err := mgr.Write("non-existent", "ls\n"); err == nil {
		t.Fatalf("expected error writing to non-existent session")
	}
	if err := mgr.Resize("non-existent", 80, 24); err != nil {
		t.Fatalf("expected nil for non-existent session resize, got %v", err)
	}
	if err := mgr.CloseSession("non-existent"); err != nil {
		t.Fatalf("expected nil closing non-existent session, got %v", err)
	}
	mgr.CloseAll()
}
