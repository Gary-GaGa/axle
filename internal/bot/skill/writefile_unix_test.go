//go:build unix

package skill

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestWriteFile_RejectsFIFOWIthoutBlocking(t *testing.T) {
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "pipe")
	if err := unix.Mkfifo(pipePath, 0600); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- WriteFile(dir, "pipe", "hello")
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected FIFO write to be rejected")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WriteFile blocked on FIFO")
	}
}
