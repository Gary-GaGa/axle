package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxMemoryEntries = 50 // per user
	memoryDir        = "memory"
)

// MemoryEntry represents a single conversation exchange.
type MemoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
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

	ms.mu.Lock()
	ms.entries[userID] = entries
	ms.mu.Unlock()
	return nil
}

// Add appends a memory entry for a user and persists to disk.
func (ms *MemoryStore) Add(userID int64, role, content, model string) error {
	entry := MemoryEntry{
		Timestamp: time.Now(),
		Role:      role,
		Content:   truncateStr(content, 2000),
		Model:     model,
	}

	ms.mu.Lock()
	entries := ms.entries[userID]
	entries = append(entries, entry)
	// Trim old entries
	if len(entries) > maxMemoryEntries {
		entries = entries[len(entries)-maxMemoryEntries:]
	}
	ms.entries[userID] = entries
	ms.mu.Unlock()

	return ms.persist(userID)
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

// BuildContext formats recent memory entries as context for Copilot prompts.
func (ms *MemoryStore) BuildContext(userID int64, maxEntries int) string {
	entries := ms.Recent(userID, maxEntries)
	if len(entries) == 0 {
		return ""
	}

	var sb []string
	sb = append(sb, "[Previous conversation context]")
	for _, e := range entries {
		prefix := "User"
		if e.Role == "assistant" {
			prefix = "Assistant"
		}
		sb = append(sb, fmt.Sprintf("%s: %s", prefix, e.Content))
	}
	sb = append(sb, "[End of context]\n")
	return joinStrings(sb, "\n")
}

// Clear removes all memory for a user.
func (ms *MemoryStore) Clear(userID int64) error {
	ms.mu.Lock()
	delete(ms.entries, userID)
	ms.mu.Unlock()

	path := ms.userFile(userID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (ms *MemoryStore) persist(userID int64) error {
	ms.mu.RLock()
	entries := ms.entries[userID]
	data, err := json.MarshalIndent(entries, "", "  ")
	ms.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("序列化記憶失敗: %w", err)
	}
	return os.WriteFile(ms.userFile(userID), data, 0600)
}

func (ms *MemoryStore) userFile(userID int64) string {
	return filepath.Join(ms.baseDir, fmt.Sprintf("%d.json", userID))
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
