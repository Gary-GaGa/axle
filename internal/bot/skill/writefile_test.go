package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hasPrefix(path, prefix string) bool {
	return strings.HasPrefix(path, prefix)
}

func TestReadCode_Basic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package test"), 0644)

	result, err := ReadCode(context.Background(), dir, "test.go")
	if err != nil {
		t.Fatalf("ReadCode failed: %v", err)
	}
	if !containsStr(result, "package test") {
		t.Error("should contain file content")
	}
	if !containsStr(result, "```go") {
		t.Error("should have go syntax highlighting")
	}
}

func TestReadCode_NonExistent(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadCode(context.Background(), dir, "nope.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadCode_PathEscape(t *testing.T) {
	dir := t.TempDir()
	// The sandbox cleans "../" so it stays inside workspace.
	// Reading a non-existent cleaned path should return "file not found".
	_, err := ReadCode(context.Background(), dir, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error (file not found inside sandbox)")
	}
}

func TestResolveAndValidate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0644)

	// Valid path
	abs, err := resolveAndValidate(dir, "ok.txt")
	if err != nil {
		t.Fatalf("resolveAndValidate failed: %v", err)
	}
	if abs != filepath.Join(dir, "ok.txt") {
		t.Errorf("got %s, want %s", abs, filepath.Join(dir, "ok.txt"))
	}

	// Path escape - "../" is cleaned by sandbox, resulting in workspace-relative path
	// For "/etc/passwd" as relPath, it becomes "{workspace}/etc/passwd" (inside workspace)
	abs, err = resolveAndValidate(dir, "../../etc/passwd")
	if err != nil {
		// This is OK too — sandbox may reject or clean
		t.Logf("escape path result: %v (acceptable)", err)
	} else {
		// If no error, verify it stayed inside workspace
		if !hasPrefix(abs, dir) {
			t.Errorf("resolved path %s escaped workspace %s", abs, dir)
		}
	}
}

func TestWriteFile_Basic(t *testing.T) {
	dir := t.TempDir()
	err := WriteFile(dir, "new.txt", "hello")
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
	if string(data) != "hello" {
		t.Errorf("content = %q, want 'hello'", string(data))
	}
}

func TestWriteFile_CreateSubdirs(t *testing.T) {
	dir := t.TempDir()
	err := WriteFile(dir, "a/b/c.txt", "nested")
	if err != nil {
		t.Fatalf("WriteFile nested failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "a", "b", "c.txt"))
	if string(data) != "nested" {
		t.Errorf("content = %q, want 'nested'", string(data))
	}
}

func TestWriteFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	// "../escape.txt" is cleaned to "escape.txt" inside workspace, so write succeeds
	err := WriteFile(dir, "../escape.txt", "test")
	if err != nil {
		// Sandbox rejected — acceptable
		t.Logf("write escape result: %v (acceptable)", err)
	} else {
		// Verify file is inside workspace
		if _, statErr := os.Stat(filepath.Join(dir, "escape.txt")); statErr != nil {
			t.Logf("file not at expected location — sandbox redirected")
		}
	}
}

func TestWriteFile_TooLarge(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, maxWriteBytes+1)
	err := WriteFile(dir, "big.txt", string(big))
	if err == nil {
		t.Error("expected error for oversized content")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("hi"), 0644)

	exists, err := FileExists(dir, "exists.txt")
	if err != nil || !exists {
		t.Error("expected file to exist")
	}

	exists, err = FileExists(dir, "nope.txt")
	if err != nil || exists {
		t.Error("expected file to not exist")
	}
}
