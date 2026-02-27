package app

import (
	"testing"
	"time"
)

func TestSessionManager_GetCopy_Empty(t *testing.T) {
	sm := NewSessionManager()
	sess := sm.GetCopy(123)
	if sess.Mode != ModeIdle {
		t.Errorf("empty session mode = %d, want ModeIdle", sess.Mode)
	}
}

func TestSessionManager_Update(t *testing.T) {
	sm := NewSessionManager()
	sm.Update(123, func(s *UserSession) {
		s.Mode = ModeAwaitExecCmd
		s.SelectedModel = "gpt-4"
	})

	sess := sm.GetCopy(123)
	if sess.Mode != ModeAwaitExecCmd {
		t.Errorf("mode = %d, want ModeAwaitExecCmd", sess.Mode)
	}
	if sess.SelectedModel != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", sess.SelectedModel)
	}
}

func TestSessionManager_Reset_PreservesModel(t *testing.T) {
	sm := NewSessionManager()
	sm.Update(123, func(s *UserSession) {
		s.Mode = ModeAwaitCopilotPrompt
		s.SelectedModel = "claude-opus"
		s.ActiveWorkspace = "/tmp/test"
		s.PendingCmd = "rm -rf"
	})

	sm.Reset(123)
	sess := sm.GetCopy(123)

	if sess.Mode != ModeIdle {
		t.Error("mode should be idle after reset")
	}
	if sess.SelectedModel != "claude-opus" {
		t.Error("model should be preserved after reset")
	}
	if sess.ActiveWorkspace != "/tmp/test" {
		t.Error("workspace should be preserved after reset")
	}
	if sess.PendingCmd != "" {
		t.Error("pending cmd should be cleared after reset")
	}
}

func TestTaskManager_TryStart(t *testing.T) {
	tm := &TaskManager{}
	ctx, ok := tm.TryStart("test-task")
	if !ok || ctx == nil {
		t.Fatal("first TryStart should succeed")
	}

	// Second should fail
	_, ok2 := tm.TryStart("second-task")
	if ok2 {
		t.Error("second TryStart should fail while first is running")
	}

	// Release
	tm.Done()

	// Third should succeed
	_, ok3 := tm.TryStart("third-task")
	if !ok3 {
		t.Error("TryStart after Done should succeed")
	}
	tm.Done()
}

func TestTaskManager_Cancel(t *testing.T) {
	tm := &TaskManager{}

	// Cancel with nothing running
	if tm.Cancel() {
		t.Error("cancel with no task should return false")
	}

	ctx, _ := tm.TryStart("cancellable")

	if !tm.Cancel() {
		t.Error("cancel with running task should return true")
	}

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Error("context should be cancelled")
	}

	tm.Done()
}

func TestTaskManager_Status(t *testing.T) {
	tm := &TaskManager{}

	running, name, elapsed := tm.Status()
	if running || name != "" || elapsed != 0 {
		t.Error("empty status should show not running")
	}

	tm.TryStart("status-test")
	time.Sleep(10 * time.Millisecond)

	running, name, elapsed = tm.Status()
	if !running {
		t.Error("should be running")
	}
	if name != "status-test" {
		t.Errorf("name = %q, want status-test", name)
	}
	if elapsed < 10*time.Millisecond {
		t.Error("elapsed should be at least 10ms")
	}

	tm.Done()
}
