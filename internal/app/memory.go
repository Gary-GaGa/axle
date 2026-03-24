package app

import (
	"context"

	domainmemory "github.com/garyellow/axle/internal/domain/memory"
	jsonmemory "github.com/garyellow/axle/internal/interface/out/persistence/json"
	"github.com/garyellow/axle/internal/usecase/dto"
	usecasememory "github.com/garyellow/axle/internal/usecase/memory"
	portin "github.com/garyellow/axle/internal/usecase/port/in"
)

const maxMemoryEntryContent = domainmemory.MaxEntryContent

type MemoryEntry = dto.MemoryEntry

type MemorySearchHit = dto.MemorySearchHit

// MemoryStore manages persistent conversation memory per user.
type MemoryStore struct {
	service portin.MemoryUsecase
}

// NewMemoryStore creates a MemoryStore backed by the given directory.
func NewMemoryStore(axleDir string) (*MemoryStore, error) {
	repo, err := jsonmemory.NewMemoryRepository(axleDir)
	if err != nil {
		return nil, err
	}
	return &MemoryStore{
		service: usecasememory.NewService(repo),
	}, nil
}

// Load reads a user's memory from disk into the cache.
func (ms *MemoryStore) Load(userID int64) error {
	return ms.service.Load(context.Background(), dto.LoadMemoryInput{UserID: userID})
}

// Add appends a standard chat memory entry for a user and persists to disk.
func (ms *MemoryStore) Add(userID int64, role, content, model string) error {
	return ms.service.Add(context.Background(), dto.AddMemoryInput{
		UserID:  userID,
		Role:    role,
		Content: content,
		Model:   model,
	})
}

// AddDetailed appends a full memory entry for a user and persists to disk.
func (ms *MemoryStore) AddDetailed(userID int64, entry MemoryEntry) error {
	return ms.service.AddDetailed(context.Background(), dto.AddDetailedMemoryInput{
		UserID: userID,
		Entry:  entry,
	})
}

// Recent returns the last N memory entries for a user.
func (ms *MemoryStore) Recent(userID int64, n int) []MemoryEntry {
	result := ms.service.Recent(context.Background(), dto.RecentMemoryInput{UserID: userID, Limit: n})
	return append([]MemoryEntry(nil), result.Entries...)
}

// Count returns the number of stored entries for a user.
func (ms *MemoryStore) Count(userID int64) int {
	result := ms.service.Count(context.Background(), dto.CountMemoryInput{UserID: userID})
	return result.Count
}

// Search finds relevant history entries for a query using lexical scoring.
func (ms *MemoryStore) Search(userID int64, query string, limit int) []MemorySearchHit {
	result := ms.service.Search(context.Background(), dto.SearchMemoryInput{
		UserID: userID,
		Query:  query,
		Limit:  limit,
	})
	return append([]MemorySearchHit(nil), result.Hits...)
}

// SearchRelevant returns the top matching entries without search metadata.
func (ms *MemoryStore) SearchRelevant(userID int64, query string, limit int) []MemoryEntry {
	result := ms.service.SearchRelevant(context.Background(), dto.SearchMemoryInput{
		UserID: userID,
		Query:  query,
		Limit:  limit,
	})
	return append([]MemoryEntry(nil), result.Entries...)
}

// BuildContext formats recent memory entries as context for Copilot prompts.
func (ms *MemoryStore) BuildContext(userID int64, maxEntries int) string {
	result := ms.service.BuildContext(context.Background(), dto.BuildContextInput{
		UserID: userID,
		Limit:  maxEntries,
	})
	return result.Text
}

// BuildRAGContext formats relevant long-term memory as retrieval context.
func (ms *MemoryStore) BuildRAGContext(userID int64, query string, maxEntries int) string {
	result := ms.service.BuildRAGContext(context.Background(), dto.BuildRAGContextInput{
		UserID: userID,
		Query:  query,
		Limit:  maxEntries,
	})
	return result.Text
}

// Clear removes all memory for a user.
func (ms *MemoryStore) Clear(userID int64) error {
	return ms.service.Clear(context.Background(), dto.ClearMemoryInput{UserID: userID})
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
