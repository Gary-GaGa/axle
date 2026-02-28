package skill

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCalendarCheckInstalled(t *testing.T) {
	// Just ensure it returns a bool without panicking
	_ = CalendarCheckInstalled()
}

func TestCalendarTodayTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // ensure timeout

	_, err := CalendarToday(ctx)
	// On macOS with icalBuddy or osascript, it might succeed or fail—either is OK
	// We're mainly testing that it doesn't panic with an expired context
	_ = err
}

func TestCalendarFunctions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tests := []struct {
		name string
		fn   func(context.Context) (string, error)
	}{
		{"Today", CalendarToday},
		{"Tomorrow", CalendarTomorrow},
		{"Week", CalendarWeek},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.fn(ctx)
			// On CI/non-macOS, icalBuddy and osascript may not work
			if err != nil {
				t.Logf("Calendar %s returned error (OK on non-macOS): %v", tt.name, err)
				return
			}
			if result == "" {
				t.Errorf("Calendar %s returned empty string", tt.name)
			}
		})
	}
}

func TestGenerateBriefing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := GenerateBriefing(ctx, t.TempDir())

	if result == "" {
		t.Fatal("briefing returned empty")
	}
	if !strings.Contains(result, "每日簡報") {
		t.Error("missing header")
	}
	if !strings.Contains(result, "系統狀態") {
		t.Error("missing system status")
	}
	if !strings.Contains(result, runtime.Version()) {
		t.Error("missing Go version")
	}
}

func TestGenerateBriefingWithGitRepo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Use the actual axle workspace which is a git repo
	result := GenerateBriefing(ctx, ".")

	if !strings.Contains(result, "每日簡報") {
		t.Error("missing header")
	}
	// Git section may or may not appear depending on workspace
	t.Log("Briefing length:", len(result))
}

func TestRunCopilotStreamNotInstalled(t *testing.T) {
	// Test with a non-existent workspace
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	called := 0
	_, err := RunCopilotStream(ctx, "/nonexistent/path", "test-model", "hello", func(s string) {
		called++
	})

	// Should fail since copilot won't find the workspace
	if err == nil {
		t.Logf("RunCopilotStream succeeded, callback called %d times", called)
	} else {
		t.Log("RunCopilotStream failed as expected:", err)
	}
}
