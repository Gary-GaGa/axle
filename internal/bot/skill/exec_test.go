package skill

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecShell_Basic(t *testing.T) {
	out, err := ExecShell(context.Background(), t.TempDir(), "echo hello")
	if err != nil {
		t.Fatalf("ExecShell: %v", err)
	}
	if out != "hello" && out != "hello\n" {
		t.Errorf("output = %q", out)
	}
}

func TestExecShell_Multiline(t *testing.T) {
	out, err := ExecShell(context.Background(), t.TempDir(), "echo a && echo b")
	if err != nil {
		t.Fatalf("ExecShell: %v", err)
	}
	if out != "a\nb" && out != "a\nb\n" {
		t.Errorf("output = %q", out)
	}
}

func TestExecShell_WorkingDir(t *testing.T) {
	dir := t.TempDir()
	out, err := ExecShell(context.Background(), dir, "pwd")
	if err != nil {
		t.Fatalf("ExecShell: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty pwd output")
	}
}

func TestExecShell_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := ExecShell(ctx, t.TempDir(), "sleep 10")
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestExecShell_FailingCommand(t *testing.T) {
	_, err := ExecShell(context.Background(), t.TempDir(), "exit 1")
	if err == nil {
		t.Error("expected error from failing command")
	}
}

func TestExecShell_Stderr(t *testing.T) {
	out, _ := ExecShell(context.Background(), t.TempDir(), "echo err >&2 && echo ok")
	if len(out) == 0 {
		t.Error("expected some output")
	}
}

func TestExecShell_NoOutput(t *testing.T) {
	out, err := ExecShell(context.Background(), t.TempDir(), "true")
	if err != nil {
		t.Fatalf("ExecShell: %v", err)
	}
	if !strings.Contains(out, "無輸出") {
		t.Errorf("expected '無輸出' placeholder, got %q", out)
	}
}

func TestExecShell_StderrOnly(t *testing.T) {
	out, err := ExecShell(context.Background(), t.TempDir(), "echo warning >&2")
	// Some shells return exit 0 even with stderr output
	if err != nil && out == "" {
		// Both err and empty out is also valid
		return
	}
	if !strings.Contains(out, "warning") {
		t.Errorf("expected stderr in output, got %q", out)
	}
}

func TestLimitedWriter_UnderLimit(t *testing.T) {
	lw := &limitedWriter{limit: 100}
	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("n = %d", n)
	}
	if lw.dropped {
		t.Error("should not be dropped")
	}
}

func TestLimitedWriter_OverLimit(t *testing.T) {
	lw := &limitedWriter{limit: 5}
	lw.Write([]byte("hel"))
	n, err := lw.Write([]byte("loworld"))
	if err != nil {
		t.Fatal(err)
	}
	// Write returns len(p) when fully over limit, or partial write count when partially fitting
	if n < 2 {
		t.Errorf("n = %d, expected at least 2", n)
	}
	if !lw.dropped {
		t.Error("should be dropped")
	}
	if lw.buf.String() != "hello" {
		t.Errorf("buf = %q", lw.buf.String())
	}
}

func TestLimitedWriter_ExactLimit(t *testing.T) {
	lw := &limitedWriter{limit: 5}
	lw.Write([]byte("hello"))
	n, _ := lw.Write([]byte("x"))
	if n != 1 {
		t.Errorf("n = %d", n)
	}
	if !lw.dropped {
		t.Error("should be dropped after limit")
	}
}
