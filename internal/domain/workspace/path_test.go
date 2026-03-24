package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathStaysInsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	abs, err := ResolvePath(dir, "a/b.txt")
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}
	wantRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if abs != filepath.Join(wantRoot, "a", "b.txt") {
		t.Fatalf("ResolvePath = %s, want %s", abs, filepath.Join(wantRoot, "a", "b.txt"))
	}
}

func TestResolvePathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := ResolvePath(dir, "../../etc/passwd"); err == nil {
		t.Fatalf("expected traversal path to be rejected")
	}
}

func TestValidateSearchPattern(t *testing.T) {
	if _, err := ValidateSearchPattern("   "); err == nil {
		t.Fatalf("expected error for empty pattern")
	}
	pattern, err := ValidateSearchPattern("  hello ")
	if err != nil {
		t.Fatalf("ValidateSearchPattern failed: %v", err)
	}
	if pattern != "hello" {
		t.Fatalf("pattern = %q, want hello", pattern)
	}
}

func TestResolvePathRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if _, err := ResolvePath(dir, "link.txt"); err == nil {
		t.Fatalf("expected symlink path to be rejected")
	}
}
