package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryStore_AddAndRecent(t *testing.T) {
	dir := t.TempDir()
	ms, err := NewMemoryStore(dir)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	_ = ms.Add(123, "user", "hello", "gpt-4")
	_ = ms.Add(123, "assistant", "hi there", "gpt-4")

	entries := ms.Recent(123, 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Role != "user" || entries[0].Content != "hello" {
		t.Errorf("first entry = %+v", entries[0])
	}
	if entries[1].Role != "assistant" {
		t.Errorf("second entry role = %s", entries[1].Role)
	}
}

func TestMemoryStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)
	_ = ms.Add(456, "user", "persistent message", "claude")

	// Create new store from same dir
	ms2, _ := NewMemoryStore(dir)
	_ = ms2.Load(456)

	entries := ms2.Recent(456, 10)
	if len(entries) != 1 || entries[0].Content != "persistent message" {
		t.Error("memory should persist across store instances")
	}
}

func TestMemoryStore_MaxEntries(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)

	for i := 0; i < maxMemoryEntries+10; i++ {
		_ = ms.Add(789, "user", "msg", "")
	}

	if ms.Count(789) > maxMemoryEntries {
		t.Errorf("count %d exceeds max %d", ms.Count(789), maxMemoryEntries)
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)
	_ = ms.Add(100, "user", "to be cleared", "")

	if err := ms.Clear(100); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if ms.Count(100) != 0 {
		t.Error("should be empty after clear")
	}

	// File should be deleted
	_, err := os.Stat(filepath.Join(dir, "memory", "100.json"))
	if !os.IsNotExist(err) {
		t.Error("file should be deleted after clear")
	}
}

func TestMemoryStore_BuildContext(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)

	// Empty context
	ctx := ms.BuildContext(999, 5)
	if ctx != "" {
		t.Error("empty user should return empty context")
	}

	_ = ms.Add(999, "user", "question 1", "")
	_ = ms.Add(999, "assistant", "answer 1", "")

	ctx = ms.BuildContext(999, 5)
	if ctx == "" {
		t.Error("should return non-empty context")
	}
	if !containsStr(ctx, "question 1") || !containsStr(ctx, "answer 1") {
		t.Error("context should contain conversation entries")
	}
}

func TestTruncateStr(t *testing.T) {
	short := truncateStr("hello", 10)
	if short != "hello" {
		t.Errorf("short = %q", short)
	}

	long := truncateStr("hello world", 5)
	if long != "hello..." {
		t.Errorf("long = %q", long)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
