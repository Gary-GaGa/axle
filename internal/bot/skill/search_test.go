package skill

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSearchCode_Basic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n\nfunc helper() {\n\treturn\n}"), 0644)

	results, err := SearchCode(context.Background(), dir, "func")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}
}

func TestSearchCode_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("Hello World\nhello world\nHELLO WORLD"), 0644)

	results, err := SearchCode(context.Background(), dir, "hello")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestSearchCode_NoMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("some content"), 0644)

	results, err := SearchCode(context.Background(), dir, "nonexistent_pattern_xyz")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchCode_SkipsBinary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "image.png"), []byte("func fake"), 0644)
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("func real"), 0644)

	results, err := SearchCode(context.Background(), dir, "func")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	// Should only find in code.go, not image.png
	for _, r := range results {
		if r.File == "image.png" {
			t.Error("should not search binary files")
		}
	}
}

func TestSearchCode_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	hidden := filepath.Join(dir, ".git")
	os.MkdirAll(hidden, 0755)
	os.WriteFile(filepath.Join(hidden, "config"), []byte("pattern match"), 0644)
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("pattern match"), 0644)

	results, err := SearchCode(context.Background(), dir, "pattern")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	for _, r := range results {
		if r.File == ".git/config" {
			t.Error("should skip hidden directories")
		}
	}
}

func TestSearchCode_SkipsSymlinkedFiles(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	os.WriteFile(outside, []byte("pattern match"), 0644)
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	results, err := SearchCode(context.Background(), dir, "pattern")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected symlinked file to be skipped, got %+v", results)
	}
}

func TestSearchCode_SymlinkWorkspaceRoot(t *testing.T) {
	realWorkspace := t.TempDir()
	os.WriteFile(filepath.Join(realWorkspace, "main.go"), []byte("pattern match"), 0644)
	linkParent := t.TempDir()
	linkWorkspace := filepath.Join(linkParent, "workspace-link")
	if err := os.Symlink(realWorkspace, linkWorkspace); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	results, err := SearchCode(context.Background(), linkWorkspace, "pattern")
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(results) != 1 || results[0].File != "main.go" {
		t.Fatalf("expected search to follow symlinked workspace root, got %+v", results)
	}
}

func TestSearchCode_RejectsHardLink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("pattern match"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Link(outside, filepath.Join(dir, "hard.txt")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}

	if _, err := SearchCode(context.Background(), dir, "pattern"); err == nil {
		t.Fatalf("expected hard link to be rejected")
	}
}

func TestSearchCode_SkipsUnreadableSubtree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on Windows")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("pattern match"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	secret := filepath.Join(dir, "secret")
	if err := os.MkdirAll(secret, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secret, "hidden.go"), []byte("pattern match"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Chmod(secret, 0000); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	defer os.Chmod(secret, 0755)

	result, err := SearchCodeDetailed(context.Background(), dir, "pattern")
	if err != nil {
		t.Fatalf("SearchCodeDetailed failed: %v", err)
	}
	if len(result.Matches) == 0 || result.Matches[0].File != "main.go" {
		t.Fatalf("expected readable matches to remain available, got %+v", result)
	}
	if !result.Truncated {
		t.Fatalf("expected skipped unreadable subtree to mark result incomplete")
	}
}

func TestSearchCode_EmptyPattern(t *testing.T) {
	dir := t.TempDir()
	_, err := SearchCode(context.Background(), dir, "")
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestFormatSearchResults_Empty(t *testing.T) {
	result := FormatSearchResults("test", nil)
	if result == "" {
		t.Error("should return non-empty string")
	}
}

func TestFormatSearchResults_EmptyButIncomplete(t *testing.T) {
	result := FormatSearchResults("test", nil, true)
	if !containsStr(result, "未找到匹配") {
		t.Fatalf("expected empty-result message, got %q", result)
	}
	if !containsStr(result, "結果可能不完整") {
		t.Fatalf("expected incomplete warning, got %q", result)
	}
}

func TestFormatSearchResults_WithResults(t *testing.T) {
	results := []SearchResult{
		{File: "main.go", Line: 1, Content: "package main"},
		{File: "main.go", Line: 3, Content: "func main()"},
	}
	formatted := FormatSearchResults("main", results)
	if !containsStr(formatted, "main.go") {
		t.Error("should contain file name")
	}
	if !containsStr(formatted, "2 筆") {
		t.Error("should contain result count")
	}
}
