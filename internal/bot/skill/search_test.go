package skill

import (
	"context"
	"os"
	"path/filepath"
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
