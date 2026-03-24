package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
	_, err := ReadCode(context.Background(), dir, "../../../etc/passwd")
	if err == nil {
		t.Error("expected traversal path to be rejected")
	}
}

func TestResolveAndValidate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0644)
	canonicalDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}

	// Valid path
	abs, err := resolveAndValidate(dir, "ok.txt")
	if err != nil {
		t.Fatalf("resolveAndValidate failed: %v", err)
	}
	if abs != filepath.Join(canonicalDir, "ok.txt") {
		t.Errorf("got %s, want %s", abs, filepath.Join(canonicalDir, "ok.txt"))
	}

	abs, err = resolveAndValidate(dir, "../../etc/passwd")
	if err == nil {
		t.Fatalf("expected traversal path to be rejected, got %s", abs)
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
	err := WriteFile(dir, "../escape.txt", "test")
	if err == nil {
		t.Fatal("expected traversal path to be rejected")
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

func TestFileExists_Directory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "notes"), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	exists, err := FileExists(dir, "notes")
	if err != nil || !exists {
		t.Fatalf("expected directory path to exist, got exists=%v err=%v", exists, err)
	}
}

func TestFileExistsDetailed_RejectsTraversalPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "escape.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := FileExistsDetailed(context.Background(), dir, "../escape.txt"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestReadCode_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	os.WriteFile(outside, []byte("secret"), 0644)
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	if _, err := ReadCode(context.Background(), dir, "link.txt"); err == nil {
		t.Fatal("expected symlink read to be rejected")
	}
}

func TestWriteFile_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	os.WriteFile(outside, []byte("secret"), 0644)
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	if err := WriteFile(dir, "link.txt", "nope"); err == nil {
		t.Fatal("expected symlink write to be rejected")
	}
}

func TestWriteFile_RejectsHardLink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Link(outside, filepath.Join(dir, "hard.txt")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}

	if err := WriteFile(dir, "hard.txt", "nope"); err == nil {
		t.Fatal("expected hard link write to be rejected")
	}
}

func TestWriteFileDetailed_RejectsTraversalPath(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteFileDetailed(context.Background(), dir, "../escape.txt", "hello"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestFileExists_RejectsHardLink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Link(outside, filepath.Join(dir, "hard.txt")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}

	if _, err := FileExists(dir, "hard.txt"); err == nil {
		t.Fatal("expected hard link existence check to be rejected")
	}
}
