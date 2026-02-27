package app

import (
	"testing"
	"time"
)

func TestSubAgentManager_CreateAndList(t *testing.T) {
	m := NewSubAgentManager()
	agent, ctx := m.Create(123, "test-agent", "do something", "gpt-4", "/tmp")

	if agent.ID == "" {
		t.Error("agent ID should not be empty")
	}
	if agent.Status != SubAgentRunning {
		t.Error("new agent should be running")
	}
	if ctx == nil {
		t.Error("context should not be nil")
	}

	agents := m.List(123)
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "test-agent" {
		t.Errorf("name = %q", agents[0].Name)
	}

	// Different user should see empty list
	agents2 := m.List(456)
	if len(agents2) != 0 {
		t.Error("different user should see no agents")
	}
}

func TestSubAgentManager_Complete(t *testing.T) {
	m := NewSubAgentManager()
	agent, _ := m.Create(123, "completer", "task", "gpt-4", "/tmp")

	m.Complete(agent.ID, "done result")
	got, ok := m.Get(agent.ID)
	if !ok {
		t.Fatal("should find agent")
	}
	if got.Status != SubAgentCompleted {
		t.Errorf("status = %d, want completed", got.Status)
	}
	if got.Result != "done result" {
		t.Errorf("result = %q", got.Result)
	}
}

func TestSubAgentManager_Fail(t *testing.T) {
	m := NewSubAgentManager()
	agent, _ := m.Create(123, "failer", "task", "gpt-4", "/tmp")

	m.Fail(agent.ID, "oops")
	got, _ := m.Get(agent.ID)
	if got.Status != SubAgentFailed {
		t.Error("should be failed")
	}
	if got.Error != "oops" {
		t.Errorf("error = %q", got.Error)
	}
}

func TestSubAgentManager_Cancel(t *testing.T) {
	m := NewSubAgentManager()
	agent, ctx := m.Create(123, "cancellable", "task", "gpt-4", "/tmp")

	if !m.Cancel(agent.ID) {
		t.Error("cancel should succeed")
	}

	// Context should be cancelled
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Error("context should be cancelled")
	}

	got, _ := m.Get(agent.ID)
	if got.Status != SubAgentCancelled {
		t.Error("should be cancelled")
	}

	// Cancel again should fail
	if m.Cancel(agent.ID) {
		t.Error("second cancel should fail")
	}
}

func TestSubAgentManager_RunningCount(t *testing.T) {
	m := NewSubAgentManager()
	m.Create(123, "a1", "t1", "m", "/")
	m.Create(123, "a2", "t2", "m", "/")
	a3, _ := m.Create(123, "a3", "t3", "m", "/")

	if m.RunningCount(123) != 3 {
		t.Errorf("running count = %d, want 3", m.RunningCount(123))
	}

	m.Complete(a3.ID, "done")
	if m.RunningCount(123) != 2 {
		t.Errorf("running count after complete = %d, want 2", m.RunningCount(123))
	}
}

func TestSubAgentManager_Cleanup(t *testing.T) {
	m := NewSubAgentManager()
	a1, _ := m.Create(123, "old", "t", "m", "/")
	m.Complete(a1.ID, "done")

	// Cleanup with 0 duration should remove completed
	removed := m.Cleanup(0)
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	if len(m.List(123)) != 0 {
		t.Error("list should be empty after cleanup")
	}
}

func TestSubAgentStatus_String(t *testing.T) {
	tests := []struct {
		s    SubAgentStatus
		want string
	}{
		{SubAgentRunning, "🔄 執行中"},
		{SubAgentCompleted, "✅ 完成"},
		{SubAgentFailed, "❌ 失敗"},
		{SubAgentCancelled, "🛑 已取消"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("SubAgentStatus(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}
