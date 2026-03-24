package jsonmemory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	domainmemory "github.com/garyellow/axle/internal/domain/memory"
)

func TestRepositoryRoundTripAndClear(t *testing.T) {
	repo, err := NewMemoryRepository(t.TempDir())
	if err != nil {
		t.Fatalf("NewMemoryRepository() error = %v", err)
	}

	input := []domainmemory.Entry{
		{
			Timestamp: time.Now().UTC().Truncate(time.Second),
			Role:      "assistant",
			Content:   "stored memory",
			Kind:      "workflow",
			Source:    "web",
			Tags:      []string{"docs"},
		},
	}
	if err := repo.Save(context.Background(), 42, input); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.Load(context.Background(), 42)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 1 || got[0].Content != "stored memory" {
		t.Fatalf("Load() = %+v", got)
	}

	if err := repo.Clear(context.Background(), 42); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	loadedAfterClear, err := repo.Load(context.Background(), 42)
	if err != nil {
		t.Fatalf("Load() after clear error = %v", err)
	}
	if len(loadedAfterClear) != 0 {
		t.Fatalf("expected no entries after clear, got %+v", loadedAfterClear)
	}
}

func TestRepositoryCreatesExpectedLayout(t *testing.T) {
	root := t.TempDir()
	repo, err := NewMemoryRepository(root)
	if err != nil {
		t.Fatalf("NewMemoryRepository() error = %v", err)
	}

	if got := repo.userFile(9); got != filepath.Join(root, memoryDir, "9.json") {
		t.Fatalf("userFile() = %q", got)
	}
}
