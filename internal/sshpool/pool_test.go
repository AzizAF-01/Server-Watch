package sshpool

import (
	"strings"
	"testing"
)

func TestBoundedBuffer_Limit(t *testing.T) {
	buf := newBoundedBuffer(100)

	// Write 60 bytes
	payload1 := []byte(strings.Repeat("A", 60))
	n, err := buf.Write(payload1)
	if err != nil || n != 60 {
		t.Fatalf("expected write 60 bytes, got n=%d, err=%v", n, err)
	}

	// Write 60 more bytes (total 120, but cap is 100)
	payload2 := []byte(strings.Repeat("B", 60))
	n, err = buf.Write(payload2)
	if err != nil || n != 60 {
		t.Fatalf("expected write to accept all bytes, got n=%d, err=%v", n, err)
	}

	res := buf.String()
	if len(res) != 100 {
		t.Fatalf("expected buffer length 100, got %d", len(res))
	}

	if !strings.HasPrefix(res, strings.Repeat("A", 60)) {
		t.Errorf("expected buffer to start with 60 As")
	}
	if !strings.HasSuffix(res, strings.Repeat("B", 40)) {
		t.Errorf("expected buffer to end with 40 Bs")
	}
}
