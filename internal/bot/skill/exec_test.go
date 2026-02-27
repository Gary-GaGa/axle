package skill

import (
	"context"
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
	// Both stdout and stderr should be captured
	if len(out) == 0 {
		t.Error("expected some output")
	}
}
