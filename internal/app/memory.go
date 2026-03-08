package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	memoryDir             = "memory"
	maxMemoryEntryContent = 4000
)

// MemoryEntry represents a single stored memory item.
type MemoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"` // "user", "assistant", "tool", "system"
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	Kind      string    `json:"kind,omitempty"`      // "chat", "browser", "workflow", "subagent", "exec"
	Source    string    `json:"source,omitempty"`    // "telegram", "web", "scheduler", etc.
	Workspace string    `json:"workspace,omitempty"` // active workspace for the memory
	Tags      []string  `json:"tags,omitempty"`
}

// MemorySearchHit is a ranked memory search result.
type MemorySearchHit struct {
	Entry   MemoryEntry `json:"entry"`
	Score   int         `json:"score"`
	Snippet string      `json:"snippet"`
}

// MemoryStore manages persistent conversation memory per user.
type MemoryStore struct {
	mu      sync.RWMutex
	baseDir string
	entries map[int64][]MemoryEntry
}

// NewMemoryStore creates a MemoryStore backed by the given directory.
func NewMemoryStore(axleDir string) (*MemoryStore, error) {
	dir := filepath.Join(axleDir, memoryDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("建立記憶目錄失敗: %w", err)
	}
	return &MemoryStore{
		baseDir: dir,
		entries: make(map[int64][]MemoryEntry),
	}, nil
}

// Load reads a user's memory from disk into the cache.
func (ms *MemoryStore) Load(userID int64) error {
	path := ms.userFile(userID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("讀取記憶失敗: %w", err)
	}

	var entries []MemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析記憶失敗: %w", err)
	}

	for i := range entries {
		entries[i] = normalizeMemoryEntry(entries[i])
	}

	ms.mu.Lock()
	ms.entries[userID] = entries
	ms.mu.Unlock()
	return nil
}

// Add appends a standard chat memory entry for a user and persists to disk.
func (ms *MemoryStore) Add(userID int64, role, content, model string) error {
	return ms.AddDetailed(userID, MemoryEntry{
		Role:    role,
		Content: content,
		Model:   model,
		Kind:    "chat",
		Source:  "telegram",
	})
}

// AddDetailed appends a full memory entry for a user and persists to disk.
func (ms *MemoryStore) AddDetailed(userID int64, entry MemoryEntry) error {
	entry = normalizeMemoryEntry(entry)

	ms.mu.Lock()
	prevLen := len(ms.entries[userID])
	ms.entries[userID] = append(ms.entries[userID], entry)
	if err := ms.persistLocked(userID); err != nil {
		ms.entries[userID] = ms.entries[userID][:prevLen]
		ms.mu.Unlock()
		return err
	}
	ms.mu.Unlock()
	return nil
}

// Recent returns the last N memory entries for a user.
func (ms *MemoryStore) Recent(userID int64, n int) []MemoryEntry {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entries := ms.entries[userID]
	if n <= 0 || n > len(entries) {
		n = len(entries)
	}
	if n == 0 {
		return nil
	}

	result := make([]MemoryEntry, n)
	copy(result, entries[len(entries)-n:])
	return result
}

// Count returns the number of stored entries for a user.
func (ms *MemoryStore) Count(userID int64) int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.entries[userID])
}

// Search finds relevant history entries for a query using lexical scoring.
func (ms *MemoryStore) Search(userID int64, query string, limit int) []MemorySearchHit {
	ms.mu.RLock()
	entries := append([]MemoryEntry(nil), ms.entries[userID]...)
	ms.mu.RUnlock()

	query = strings.TrimSpace(query)
	if query == "" || len(entries) == 0 {
		return nil
	}

	queryTokens := tokenizeText(query)
	if len(queryTokens) == 0 {
		return nil
	}

	hits := make([]MemorySearchHit, 0, len(entries))
	for _, entry := range entries {
		score := scoreMemoryEntry(entry, query, queryTokens)
		if score <= 0 {
			continue
		}
		hits = append(hits, MemorySearchHit{
			Entry:   entry,
			Score:   score,
			Snippet: buildSnippet(entry.Content, query),
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Entry.Timestamp.After(hits[j].Entry.Timestamp)
		}
		return hits[i].Score > hits[j].Score
	})

	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// SearchRelevant returns the top matching entries without search metadata.
func (ms *MemoryStore) SearchRelevant(userID int64, query string, limit int) []MemoryEntry {
	hits := ms.Search(userID, query, limit)
	if len(hits) == 0 {
		return nil
	}

	result := make([]MemoryEntry, 0, len(hits))
	for _, hit := range hits {
		result = append(result, hit.Entry)
	}
	return result
}

// BuildContext formats recent memory entries as context for Copilot prompts.
func (ms *MemoryStore) BuildContext(userID int64, maxEntries int) string {
	entries := ms.Recent(userID, maxEntries)
	if len(entries) == 0 {
		return ""
	}

	var sb []string
	sb = append(sb, "[Recent conversation context]")
	for _, e := range entries {
		sb = append(sb, formatMemoryLine(e))
	}
	sb = append(sb, "[End of recent context]\n")
	return joinStrings(sb, "\n")
}

// BuildRAGContext formats relevant long-term memory as retrieval context.
func (ms *MemoryStore) BuildRAGContext(userID int64, query string, maxEntries int) string {
	entries := ms.SearchRelevant(userID, query, maxEntries)
	if len(entries) == 0 {
		return ""
	}

	var sb []string
	sb = append(sb, "[Relevant long-term memory]")
	for _, e := range entries {
		line := formatMemoryLine(e)
		if !e.Timestamp.IsZero() {
			line = fmt.Sprintf("%s [%s]", line, e.Timestamp.Format("2006-01-02 15:04"))
		}
		sb = append(sb, line)
	}
	sb = append(sb, "[End of long-term memory]\n")
	return joinStrings(sb, "\n")
}

// Clear removes all memory for a user.
func (ms *MemoryStore) Clear(userID int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.entries, userID)
	path := ms.userFile(userID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (ms *MemoryStore) persistLocked(userID int64) error {
	entries := append([]MemoryEntry(nil), ms.entries[userID]...)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化記憶失敗: %w", err)
	}
	return writeFileAtomic(ms.userFile(userID), data, 0600)
}

func (ms *MemoryStore) userFile(userID int64) string {
	return filepath.Join(ms.baseDir, fmt.Sprintf("%d.json", userID))
}

func normalizeMemoryEntry(entry MemoryEntry) MemoryEntry {
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
	entry.Content = truncateStr(strings.TrimSpace(entry.Content), maxMemoryEntryContent)
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

func scoreMemoryEntry(entry MemoryEntry, query string, queryTokens []string) int {
	contentLower := strings.ToLower(entry.Content)
	queryLower := strings.ToLower(query)

	score := 0
	matched := false
	if strings.Contains(contentLower, queryLower) {
		score += 40
		matched = true
	}

	tokenSet := make(map[string]struct{}, len(queryTokens))
	for _, token := range queryTokens {
		tokenSet[token] = struct{}{}
	}

	for token := range tokenSet {
		if len(token) < 2 {
			continue
		}
		if strings.Contains(contentLower, token) {
			score += 12
			matched = true
		}
		if strings.Contains(strings.ToLower(entry.Kind), token) {
			score += 6
			matched = true
		}
		if strings.Contains(strings.ToLower(entry.Source), token) {
			score += 4
			matched = true
		}
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), token) {
				score += 5
				matched = true
			}
		}
	}

	if !matched {
		return 0
	}

	switch entry.Role {
	case "assistant":
		score += 3
	case "tool":
		score += 2
	}

	if !entry.Timestamp.IsZero() {
		age := time.Since(entry.Timestamp)
		switch {
		case age < 24*time.Hour:
			score += 10
		case age < 7*24*time.Hour:
			score += 6
		case age < 30*24*time.Hour:
			score += 3
		}
	}

	return score
}

func formatMemoryLine(entry MemoryEntry) string {
	prefix := roleLabel(entry.Role)
	if entry.Source != "" && entry.Source != "telegram" {
		prefix += "/" + entry.Source
	}
	if entry.Kind != "" && entry.Kind != "chat" {
		prefix += "/" + entry.Kind
	}
	return fmt.Sprintf("%s: %s", prefix, entry.Content)
}

func roleLabel(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	case "tool":
		return "Tool"
	case "system":
		return "System"
	default:
		if role == "" {
			return "System"
		}
		return role
	}
}

func buildSnippet(content, query string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if len(content) <= 180 {
		return content
	}

	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery != "" {
		if idx := strings.Index(lowerContent, lowerQuery); idx >= 0 {
			start := idx - 60
			if start < 0 {
				start = 0
			}
			end := idx + len(query) + 80
			if end > len(content) {
				end = len(content)
			}
			snippet := content[start:end]
			if start > 0 {
				snippet = "..." + snippet
			}
			if end < len(content) {
				snippet += "..."
			}
			return snippet
		}
	}
	return truncateStr(content, 180)
}

func tokenizeText(text string) []string {
	text = strings.ToLower(text)
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Fields(b.String())
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
