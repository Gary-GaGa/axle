package dto

import "time"

type MemoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Source    string    `json:"source,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}

type MemorySearchHit struct {
	Entry   MemoryEntry `json:"entry"`
	Score   int         `json:"score"`
	Snippet string      `json:"snippet"`
}

type LoadMemoryInput struct {
	UserID int64
}

type AddMemoryInput struct {
	UserID  int64
	Role    string
	Content string
	Model   string
}

type AddDetailedMemoryInput struct {
	UserID int64
	Entry  MemoryEntry
}

type RecentMemoryInput struct {
	UserID int64
	Limit  int
}

type CountMemoryInput struct {
	UserID int64
}

type SearchMemoryInput struct {
	UserID int64
	Query  string
	Limit  int
}

type BuildContextInput struct {
	UserID int64
	Limit  int
}

type BuildRAGContextInput struct {
	UserID int64
	Query  string
	Limit  int
}

type ClearMemoryInput struct {
	UserID int64
}

type MemoryEntriesOutput struct {
	Entries []MemoryEntry
}

type MemorySearchOutput struct {
	Hits []MemorySearchHit
}

type MemoryCountOutput struct {
	Count int
}

type MemoryTextOutput struct {
	Text string
}
