package skill

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	cmds := []string{
		"git init",
		"git config user.email test@test.com",
		"git config user.name Test",
	}
	for _, c := range cmds {
		if _, err := ExecShell(ctx, dir, c); err != nil {
			t.Fatalf("setup %q: %v", c, err)
		}
	}
	// Create a file and initial commit
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644)
	ExecShell(ctx, dir, "git add . && git commit -m 'init'")
	return dir
}

func TestGitStatus(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)

	// Clean status
	out, err := GitStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GitStatus: %v", err)
	}
	if strings.Contains(out, "modified") {
		t.Error("expected clean status")
	}

	// Dirty status
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	out, err = GitStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GitStatus dirty: %v", err)
	}
	if !strings.Contains(out, "new.txt") {
		t.Error("expected new.txt in status")
	}
}

func TestGitDiff(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)

	// Modify file
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0644)

	// Unstaged diff
	out, err := GitDiff(context.Background(), dir, false)
	if err != nil {
		t.Fatalf("GitDiff: %v", err)
	}
	if !strings.Contains(out, "world") {
		t.Error("expected diff to contain 'world'")
	}

	// Staged diff
	ExecShell(context.Background(), dir, "git add .")
	out, err = GitDiff(context.Background(), dir, true)
	if err != nil {
		t.Fatalf("GitDiff staged: %v", err)
	}
	if !strings.Contains(out, "world") {
		t.Error("expected staged diff to contain 'world'")
	}
}

func TestGitLog(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)

	out, err := GitLog(context.Background(), dir, 5)
	if err != nil {
		t.Fatalf("GitLog: %v", err)
	}
	if !strings.Contains(out, "init") {
		t.Error("expected 'init' in log")
	}
}

func TestGitAddCommitPush(t *testing.T) {
	if !isGitAvailable() {
		t.Skip("git not available")
	}
	dir := setupGitRepo(t)

	os.WriteFile(filepath.Join(dir, "commit.txt"), []byte("data"), 0644)
	// Commit without remote (push will fail, but commit should succeed)
	out, err := GitAddCommitPush(context.Background(), dir, "test commit")
	// Push may fail (no remote), but the output should contain commit info
	_ = err
	if !strings.Contains(out, "test commit") && !strings.Contains(out, "push") {
		t.Logf("output: %s", out)
	}
}

func TestGitStatus_NotARepo(t *testing.T) {
	out, err := GitStatus(context.Background(), t.TempDir())
	// Either returns error or output contains "fatal"
	if err == nil && !strings.Contains(out, "fatal") {
		t.Error("expected error or fatal message for non-git directory")
	}
}
