package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListDir_Basic(t *testing.T) {
	dir := t.TempDir()
	// Create some files and dirs
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "file1.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "subdir", "file2.go"), []byte("package sub"), 0644)

	result, err := ListDir(context.Background(), dir, ".", 2)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	if result == "" {
		t.Fatal("ListDir returned empty")
	}

	// Should contain both files
	if !contains(result, "file1.go") {
		t.Error("result should contain file1.go")
	}
	if !contains(result, "subdir") {
		t.Error("result should contain subdir")
	}
	if !contains(result, "file2.go") {
		t.Error("result should contain file2.go")
	}
}

func TestListDir_HiddenFilesExcluded(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("hello"), 0644)

	result, err := ListDir(context.Background(), dir, ".", 1)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	if contains(result, ".hidden") {
		t.Error("hidden files should be excluded")
	}
	if !contains(result, "visible.txt") {
		t.Error("visible files should be included")
	}
}

func TestListDir_NonExistent(t *testing.T) {
	_, err := ListDir(context.Background(), t.TempDir(), "nonexistent", 1)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestListDir_NotADir(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	os.WriteFile(f, []byte("hello"), 0644)
	_, err := ListDir(context.Background(), dir, "file.txt", 1)
	if err == nil {
		t.Error("expected error for non-directory")
	}
}

func TestListDir_ContextCancel(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", "b", "c"), 0755)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, _ := ListDir(ctx, dir, ".", 5)
	// Should still return something (the root header at minimum)
	_ = result
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
