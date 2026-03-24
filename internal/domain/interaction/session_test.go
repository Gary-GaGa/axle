package interaction

import "testing"

func TestSessionManagerGetCopyEmpty(t *testing.T) {
	sm := NewSessionManager()
	sess := sm.GetCopy(123)
	if sess.Mode != ModeIdle {
		t.Fatalf("empty session mode = %d, want %d", sess.Mode, ModeIdle)
	}
}

func TestSessionManagerUpdateAndReset(t *testing.T) {
	sm := NewSessionManager()
	sm.Update(123, func(s *UserSession) {
		s.Mode = ModeAwaitExecCmd
		s.SelectedModel = "gpt-4"
		s.ActiveWorkspace = "/tmp/project"
		s.PendingCmd = "rm -rf /"
		s.EnabledExtras = map[string]bool{"memory": true}
	})

	sess := sm.GetCopy(123)
	if sess.Mode != ModeAwaitExecCmd {
		t.Fatalf("mode = %d, want %d", sess.Mode, ModeAwaitExecCmd)
	}
	if sess.SelectedModel != "gpt-4" {
		t.Fatalf("model = %q, want gpt-4", sess.SelectedModel)
	}

	sm.Reset(123)
	reset := sm.GetCopy(123)
	if reset.Mode != ModeIdle {
		t.Fatalf("reset mode = %d, want %d", reset.Mode, ModeIdle)
	}
	if reset.SelectedModel != "gpt-4" {
		t.Fatalf("reset should preserve model, got %q", reset.SelectedModel)
	}
	if reset.ActiveWorkspace != "/tmp/project" {
		t.Fatalf("reset should preserve workspace, got %q", reset.ActiveWorkspace)
	}
	if reset.PendingCmd != "" {
		t.Fatalf("reset should clear pending command, got %q", reset.PendingCmd)
	}
	if !reset.EnabledExtras["memory"] {
		t.Fatalf("reset should preserve enabled extras")
	}
}

func TestSessionManagerGetCopyClonesExtras(t *testing.T) {
	sm := NewSessionManager()
	sm.Update(123, func(s *UserSession) {
		s.EnabledExtras = map[string]bool{"memory": true}
	})

	cp := sm.GetCopy(123)
	cp.EnabledExtras["memory"] = false
	cp.EnabledExtras["browser"] = true

	again := sm.GetCopy(123)
	if !again.EnabledExtras["memory"] {
		t.Fatalf("stored extras should not be mutated through copy")
	}
	if again.EnabledExtras["browser"] {
		t.Fatalf("new keys in copied extras should not leak back")
	}
}
