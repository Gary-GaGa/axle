package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SubAgentStatus tracks the state of a delegated sub-agent task.
type SubAgentStatus int

const (
	SubAgentRunning SubAgentStatus = iota
	SubAgentCompleted
	SubAgentFailed
	SubAgentCancelled
)

func (s SubAgentStatus) String() string {
	switch s {
	case SubAgentRunning:
		return "🔄 執行中"
	case SubAgentCompleted:
		return "✅ 完成"
	case SubAgentFailed:
		return "❌ 失敗"
	case SubAgentCancelled:
		return "🛑 已取消"
	default:
		return "❓ 未知"
	}
}

// SubAgent represents a delegated background task.
type SubAgent struct {
	ID        string
	Name      string
	Task      string
	Model     string
	Workspace string
	UserID    int64
	Status    SubAgentStatus
	Result    string
	Error     string
	CreatedAt time.Time
	DoneAt    time.Time
	cancelFn  context.CancelFunc
}

// SubAgentManager manages multiple sub-agent tasks.
type SubAgentManager struct {
	mu     sync.RWMutex
	agents map[string]*SubAgent
	nextID int
}

// NewSubAgentManager creates a ready-to-use SubAgentManager.
func NewSubAgentManager() *SubAgentManager {
	return &SubAgentManager{agents: make(map[string]*SubAgent)}
}

// Create registers a new sub-agent and returns its context.
func (m *SubAgentManager) Create(userID int64, name, task, model, workspace string) (*SubAgent, context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	id := fmt.Sprintf("agent-%d", m.nextID)

	ctx, cancel := context.WithCancel(context.Background())
	agent := &SubAgent{
		ID:        id,
		Name:      name,
		Task:      task,
		Model:     model,
		Workspace: workspace,
		UserID:    userID,
		Status:    SubAgentRunning,
		CreatedAt: time.Now(),
		cancelFn:  cancel,
	}
	m.agents[id] = agent
	slog.Info("🤖 子代理已建立", "id", id, "name", name, "user_id", userID)
	return agent, ctx
}

// Complete marks a sub-agent as completed with result.
func (m *SubAgentManager) Complete(id, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.agents[id]; ok {
		a.Status = SubAgentCompleted
		a.Result = truncateStr(result, 4000)
		a.DoneAt = time.Now()
	}
}

// Fail marks a sub-agent as failed with error.
func (m *SubAgentManager) Fail(id, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.agents[id]; ok {
		a.Status = SubAgentFailed
		a.Error = errMsg
		a.DoneAt = time.Now()
	}
}

// Cancel cancels a running sub-agent.
func (m *SubAgentManager) Cancel(id string) bool {
	m.mu.Lock()
	a, ok := m.agents[id]
	if !ok || a.Status != SubAgentRunning {
		m.mu.Unlock()
		return false
	}
	cancel := a.cancelFn
	a.Status = SubAgentCancelled
	a.DoneAt = time.Now()
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	slog.Info("🛑 子代理已取消", "id", id)
	return true
}

// List returns all sub-agents for a user, ordered by creation time (newest first).
func (m *SubAgentManager) List(userID int64) []*SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*SubAgent
	for _, a := range m.agents {
		if a.UserID == userID {
			cp := *a
			result = append(result, &cp)
		}
	}
	// Sort newest first
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// RunningCount returns the number of running sub-agents for a user.
func (m *SubAgentManager) RunningCount(userID int64) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, a := range m.agents {
		if a.UserID == userID && a.Status == SubAgentRunning {
			count++
		}
	}
	return count
}

// Get returns a copy of a sub-agent by ID.
func (m *SubAgentManager) Get(id string) (*SubAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[id]
	if !ok {
		return nil, false
	}
	cp := *a
	return &cp, true
}

// Cleanup removes completed/failed/cancelled agents older than the given duration.
func (m *SubAgentManager) Cleanup(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, a := range m.agents {
		if a.Status != SubAgentRunning && a.DoneAt.Before(cutoff) {
			delete(m.agents, id)
			removed++
		}
	}
	return removed
}
