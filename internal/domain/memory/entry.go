package memory

import (
	"strings"
	"time"
)

const MaxEntryContent = 4000

// Entry represents a single stored memory item.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	Kind      string    `json:"kind,omitempty"`
	Source    string    `json:"source,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}

// SearchHit is a ranked memory search result.
type SearchHit struct {
	Entry   Entry  `json:"entry"`
	Score   int    `json:"score"`
	Snippet string `json:"snippet"`
}

// NormalizeEntry fills defaults and trims unsupported content.
func NormalizeEntry(entry Entry) Entry {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.Role == "" {
		entry.Role = "system"
	}
	if entry.Kind == "" {
		entry.Kind = "chat"
	}
	if entry.Source == "" {
		entry.Source = "telegram"
	}
	entry.Content = truncateStr(strings.TrimSpace(entry.Content), MaxEntryContent)
	if len(entry.Tags) > 0 {
		tags := make([]string, 0, len(entry.Tags))
		for _, tag := range entry.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		entry.Tags = tags
	}
	return entry
}
