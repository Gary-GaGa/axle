package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMemoryStore_AddAndRecent(t *testing.T) {
	dir := t.TempDir()
	ms, err := NewMemoryStore(dir)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}

	_ = ms.Add(123, "user", "hello", "gpt-4")
	_ = ms.AddDetailed(123, MemoryEntry{
		Role:      "assistant",
		Content:   "hi there",
		Model:     "gpt-4",
		Kind:      "chat",
		Source:    "web",
		Workspace: "/tmp/project",
		Tags:      []string{"reply"},
	})

	entries := ms.Recent(123, 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Role != "user" || entries[0].Content != "hello" {
		t.Errorf("first entry = %+v", entries[0])
	}
	if entries[1].Source != "web" || entries[1].Workspace != "/tmp/project" {
		t.Errorf("second entry metadata = %+v", entries[1])
	}
}

func TestMemoryStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)
	_ = ms.AddDetailed(456, MemoryEntry{
		Role:    "tool",
		Content: "persistent message",
		Model:   "claude",
		Kind:    "browser",
		Source:  "web",
		Tags:    []string{"docs", "screenshot"},
	})

	ms2, _ := NewMemoryStore(dir)
	_ = ms2.Load(456)

	entries := ms2.Recent(456, 10)
	if len(entries) != 1 || entries[0].Content != "persistent message" {
		t.Error("memory should persist across store instances")
	}
	if entries[0].Kind != "browser" || entries[0].Source != "web" {
		t.Errorf("loaded metadata = %+v", entries[0])
	}
}

func TestMemoryStore_Search(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)
	now := time.Now()

	_ = ms.AddDetailed(789, MemoryEntry{
		Timestamp: now.Add(-2 * time.Hour),
		Role:      "assistant",
		Content:   "We used wttr.in to fetch Taipei weather details and return a concise summary.",
		Kind:      "workflow",
		Source:    "telegram",
		Tags:      []string{"weather", "taipei"},
	})
	_ = ms.AddDetailed(789, MemoryEntry{
		Timestamp: now.Add(-time.Hour),
		Role:      "tool",
		Content:   "Browser extracted release notes from the dashboard successfully.",
		Kind:      "browser",
		Source:    "web",
		Tags:      []string{"dashboard", "release-notes"},
	})

	hits := ms.Search(789, "Taipei weather", 5)
	if len(hits) == 0 {
		t.Fatal("expected search hits")
	}
	if !strings.Contains(strings.ToLower(hits[0].Entry.Content), "weather") {
		t.Fatalf("top hit = %+v", hits[0])
	}

	misses := ms.Search(789, "quantum banana", 5)
	if len(misses) != 0 {
		t.Fatalf("expected no irrelevant hits, got %+v", misses)
	}
}

func TestMemoryStore_BuildContextAndRAG(t *testing.T) {
	dir := t.TempDir()
	ms, _ := NewMemoryStore(dir)

	_ = ms.Add(999, "user", "question 1", "")
	_ = ms.Add(999, "assistant", "answer 1", "")
	_ = ms.AddDetailed(999, MemoryEntry{
		Role:    "tool",
		Content: "Saved screenshot for docs at .axle/browser/run-1/page.png",
		Kind:    "browser",
		Source:  "web",
		Tags:    []string{"docs", "screenshot"},
	})

	ctx := ms.BuildContext(999, 5)
	if ctx == "" || !strings.Contains(ctx, "question 1") || !strings.Contains(ctx, "answer 1") {
		t.Fatalf("unexpected recent context: %q", ctx)
	}

	rag := ms.BuildRAGContext(999, "screenshot docs", 5)
	if rag == "" || !strings.Contains(strings.ToLower(rag), "screenshot") {
		t.Fatalf("unexpected rag context: %q", rag)
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

	_, err := os.Stat(filepath.Join(dir, "memory", "100.json"))
	if !os.IsNotExist(err) {
		t.Error("file should be deleted after clear")
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
