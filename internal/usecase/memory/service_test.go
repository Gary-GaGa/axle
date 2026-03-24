package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	domainmemory "github.com/garyellow/axle/internal/domain/memory"
	"github.com/garyellow/axle/internal/usecase/dto"
)

type stubRepository struct {
	loadEntries []domainmemory.Entry
	saveErr     error
	clearErr    error
	saved       []domainmemory.Entry
	clearedUser int64
}

func (s *stubRepository) Load(_ context.Context, _ int64) ([]domainmemory.Entry, error) {
	return append([]domainmemory.Entry(nil), s.loadEntries...), nil
}

func (s *stubRepository) Save(_ context.Context, _ int64, entries []domainmemory.Entry) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = append([]domainmemory.Entry(nil), entries...)
	return nil
}

func (s *stubRepository) Clear(_ context.Context, userID int64) error {
	if s.clearErr != nil {
		return s.clearErr
	}
	s.clearedUser = userID
	return nil
}

func TestServiceLoadAndSearch(t *testing.T) {
	repo := &stubRepository{
		loadEntries: []domainmemory.Entry{
			{
				Timestamp: time.Now().Add(-time.Hour),
				Role:      "assistant",
				Content:   "Axle already used browser capture for release docs.",
				Tags:      []string{"browser", "docs"},
			},
		},
	}
	service := NewService(repo)

	if err := service.Load(context.Background(), dto.LoadMemoryInput{UserID: 7}); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := service.Count(context.Background(), dto.CountMemoryInput{UserID: 7}); got.Count != 1 {
		t.Fatalf("Count() = %d, want 1", got.Count)
	}

	hits := service.Search(context.Background(), dto.SearchMemoryInput{UserID: 7, Query: "browser docs", Limit: 5}).Hits
	if len(hits) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(hits))
	}
	if hits[0].Score <= 0 {
		t.Fatalf("Search() score = %d, want > 0", hits[0].Score)
	}
}

func TestServiceAddDetailedRollbackOnSaveError(t *testing.T) {
	repo := &stubRepository{
		loadEntries: []domainmemory.Entry{
			{Role: "user", Content: "existing"},
		},
		saveErr: errors.New("boom"),
	}
	service := NewService(repo)
	if err := service.Load(context.Background(), dto.LoadMemoryInput{UserID: 11}); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	err := service.AddDetailed(context.Background(), dto.AddDetailedMemoryInput{
		UserID: 11,
		Entry:  dto.MemoryEntry{Role: "assistant", Content: "new"},
	})
	if err == nil {
		t.Fatal("expected AddDetailed() error")
	}

	if got := service.Count(context.Background(), dto.CountMemoryInput{UserID: 11}); got.Count != 1 {
		t.Fatalf("Count() after rollback = %d, want 1", got.Count)
	}
}

func TestServiceClear(t *testing.T) {
	repo := &stubRepository{}
	service := NewService(repo)

	if err := service.Add(context.Background(), dto.AddMemoryInput{UserID: 3, Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := service.Clear(context.Background(), dto.ClearMemoryInput{UserID: 3}); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if repo.clearedUser != 3 {
		t.Fatalf("cleared user = %d, want 3", repo.clearedUser)
	}
	if got := service.Count(context.Background(), dto.CountMemoryInput{UserID: 3}); got.Count != 0 {
		t.Fatalf("Count() after clear = %d, want 0", got.Count)
	}
}

func TestServiceClearPreservesCacheOnRepositoryError(t *testing.T) {
	repo := &stubRepository{clearErr: errors.New("boom")}
	service := NewService(repo)

	if err := service.Add(context.Background(), dto.AddMemoryInput{UserID: 5, Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := service.Clear(context.Background(), dto.ClearMemoryInput{UserID: 5}); err == nil {
		t.Fatal("expected Clear() error")
	}
	if got := service.Count(context.Background(), dto.CountMemoryInput{UserID: 5}); got.Count != 1 {
		t.Fatalf("Count() after failed clear = %d, want 1", got.Count)
	}
}

func TestServiceConcurrentReadAndWrite(t *testing.T) {
	repo := &stubRepository{}
	service := NewService(repo)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := service.Add(context.Background(), dto.AddMemoryInput{
				UserID:  88,
				Role:    "user",
				Content: fmt.Sprintf("entry-%d browser docs", i),
			}); err != nil {
				t.Errorf("Add() error = %v", err)
			}
		}(i)
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.Count(context.Background(), dto.CountMemoryInput{UserID: 88})
			_ = service.Search(context.Background(), dto.SearchMemoryInput{UserID: 88, Query: "browser docs", Limit: 5})
			_ = service.BuildContext(context.Background(), dto.BuildContextInput{UserID: 88, Limit: 5})
			_ = service.BuildRAGContext(context.Background(), dto.BuildRAGContextInput{UserID: 88, Query: "browser docs", Limit: 5})
		}()
	}

	wg.Wait()

	if got := service.Count(context.Background(), dto.CountMemoryInput{UserID: 88}); got.Count != 20 {
		t.Fatalf("Count() = %d, want 20", got.Count)
	}
}
