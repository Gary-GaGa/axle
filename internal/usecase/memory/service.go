package memory

import (
	"context"
	"fmt"
	"sync"

	domainmemory "github.com/garyellow/axle/internal/domain/memory"
	"github.com/garyellow/axle/internal/usecase/dto"
	portin "github.com/garyellow/axle/internal/usecase/port/in"
)

var _ portin.MemoryUsecase = (*Service)(nil)

type Service struct {
	repo        domainmemory.Repository
	mu          sync.RWMutex
	timelines   map[int64]*domainmemory.Timeline
	userLocks   map[int64]*sync.Mutex
	userLocksMu sync.Mutex
}

func NewService(repo domainmemory.Repository) *Service {
	return &Service{
		repo:      repo,
		timelines: make(map[int64]*domainmemory.Timeline),
		userLocks: make(map[int64]*sync.Mutex),
	}
}

func (s *Service) Load(ctx context.Context, input dto.LoadMemoryInput) error {
	lock := s.userLock(input.UserID)
	lock.Lock()
	defer lock.Unlock()

	entries, err := s.repo.Load(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("load memory: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.timelines[input.UserID] = domainmemory.NewTimeline(entries)
	return nil
}

func (s *Service) Add(ctx context.Context, input dto.AddMemoryInput) error {
	return s.AddDetailed(ctx, dto.AddDetailedMemoryInput{
		UserID: input.UserID,
		Entry: dto.MemoryEntry{
			Role:    input.Role,
			Content: input.Content,
			Model:   input.Model,
			Kind:    "chat",
			Source:  "telegram",
		},
	})
}

func (s *Service) AddDetailed(ctx context.Context, input dto.AddDetailedMemoryInput) error {
	entry := dtoToDomainEntry(input.Entry)
	lock := s.userLock(input.UserID)
	lock.Lock()
	defer lock.Unlock()

	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	s.mu.RUnlock()

	if timeline == nil {
		entries, err := s.repo.Load(ctx, input.UserID)
		if err != nil {
			return fmt.Errorf("load memory before save: %w", err)
		}
		timeline = domainmemory.NewTimeline(entries)
	}

	next := timeline.Snapshot()
	next.Add(entry)
	entries := next.Entries()
	if err := s.repo.Save(ctx, input.UserID, entries); err != nil {
		return fmt.Errorf("save memory: %w", err)
	}

	s.mu.Lock()
	s.timelines[input.UserID] = next
	s.mu.Unlock()
	return nil
}

func (s *Service) Recent(_ context.Context, input dto.RecentMemoryInput) *dto.MemoryEntriesOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemoryEntriesOutput{}
	}
	entries := timeline.Recent(input.Limit)
	s.mu.RUnlock()
	return &dto.MemoryEntriesOutput{Entries: entriesToDTO(entries)}
}

func (s *Service) Count(_ context.Context, input dto.CountMemoryInput) *dto.MemoryCountOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemoryCountOutput{}
	}
	count := timeline.Count()
	s.mu.RUnlock()
	return &dto.MemoryCountOutput{Count: count}
}

func (s *Service) Search(_ context.Context, input dto.SearchMemoryInput) *dto.MemorySearchOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemorySearchOutput{}
	}
	snapshot := timeline.Snapshot()
	s.mu.RUnlock()
	hits := snapshot.Search(input.Query, input.Limit)
	return &dto.MemorySearchOutput{Hits: searchHitsToDTO(hits)}
}

func (s *Service) SearchRelevant(_ context.Context, input dto.SearchMemoryInput) *dto.MemoryEntriesOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemoryEntriesOutput{}
	}
	snapshot := timeline.Snapshot()
	s.mu.RUnlock()
	entries := snapshot.SearchRelevant(input.Query, input.Limit)
	return &dto.MemoryEntriesOutput{Entries: entriesToDTO(entries)}
}

func (s *Service) BuildContext(_ context.Context, input dto.BuildContextInput) *dto.MemoryTextOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemoryTextOutput{}
	}
	snapshot := timeline.Snapshot()
	s.mu.RUnlock()
	text := snapshot.BuildContext(input.Limit)
	return &dto.MemoryTextOutput{Text: text}
}

func (s *Service) BuildRAGContext(_ context.Context, input dto.BuildRAGContextInput) *dto.MemoryTextOutput {
	s.mu.RLock()
	timeline := s.timelines[input.UserID]
	if timeline == nil {
		s.mu.RUnlock()
		return &dto.MemoryTextOutput{}
	}
	snapshot := timeline.Snapshot()
	s.mu.RUnlock()
	text := snapshot.BuildRAGContext(input.Query, input.Limit)
	return &dto.MemoryTextOutput{Text: text}
}

func (s *Service) Clear(ctx context.Context, input dto.ClearMemoryInput) error {
	lock := s.userLock(input.UserID)
	lock.Lock()
	defer lock.Unlock()

	if err := s.repo.Clear(ctx, input.UserID); err != nil {
		return fmt.Errorf("clear memory: %w", err)
	}

	s.mu.Lock()
	delete(s.timelines, input.UserID)
	s.mu.Unlock()
	return nil
}

func (s *Service) userLock(userID int64) *sync.Mutex {
	s.userLocksMu.Lock()
	defer s.userLocksMu.Unlock()

	lock := s.userLocks[userID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.userLocks[userID] = lock
	}
	return lock
}

func dtoToDomainEntry(entry dto.MemoryEntry) domainmemory.Entry {
	return domainmemory.Entry{
		Timestamp: entry.Timestamp,
		Role:      entry.Role,
		Content:   entry.Content,
		Model:     entry.Model,
		Kind:      entry.Kind,
		Source:    entry.Source,
		Workspace: entry.Workspace,
		Tags:      append([]string(nil), entry.Tags...),
	}
}

func entriesToDTO(entries []domainmemory.Entry) []dto.MemoryEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]dto.MemoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.MemoryEntry{
			Timestamp: entry.Timestamp,
			Role:      entry.Role,
			Content:   entry.Content,
			Model:     entry.Model,
			Kind:      entry.Kind,
			Source:    entry.Source,
			Workspace: entry.Workspace,
			Tags:      append([]string(nil), entry.Tags...),
		})
	}
	return result
}

func searchHitsToDTO(hits []domainmemory.SearchHit) []dto.MemorySearchHit {
	if len(hits) == 0 {
		return nil
	}
	result := make([]dto.MemorySearchHit, 0, len(hits))
	for _, hit := range hits {
		result = append(result, dto.MemorySearchHit{
			Entry: dto.MemoryEntry{
				Timestamp: hit.Entry.Timestamp,
				Role:      hit.Entry.Role,
				Content:   hit.Entry.Content,
				Model:     hit.Entry.Model,
				Kind:      hit.Entry.Kind,
				Source:    hit.Entry.Source,
				Workspace: hit.Entry.Workspace,
				Tags:      append([]string(nil), hit.Entry.Tags...),
			},
			Score:   hit.Score,
			Snippet: hit.Snippet,
		})
	}
	return result
}
