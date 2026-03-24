package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseWorkflowPlan(t *testing.T) {
	t.Setenv("AXLE_ALLOW_UNSAFE_BROWSER", "1")
	raw := "```json\n{\"steps\":[{\"id\":\"research\",\"name\":\"Research\",\"kind\":\"copilot\",\"prompt\":\"Analyze repo\"},{\"id\":\"browse\",\"name\":\"Browse docs\",\"kind\":\"browser\",\"script\":\"open https://1.1.1.1\\nwait 1s\\nextract body\",\"depends_on\":[\"research\"]}]}\n```"

	steps, err := parseWorkflowPlan(raw)
	if err != nil {
		t.Fatalf("parseWorkflowPlan: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[1].Kind != "browser" || len(steps[1].DependsOn) != 1 {
		t.Fatalf("unexpected step: %+v", steps[1])
	}
}

func TestWorkflowManager_StartPlannedSuccess(t *testing.T) {
	t.Setenv("AXLE_ALLOW_UNSAFE_BROWSER", "1")
	dir := t.TempDir()
	mem, _ := NewMemoryStore(dir)
	wm, err := NewWorkflowManager(dir, mem)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}

	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			return "copilot result for " + prompt, nil
		},
		func(ctx context.Context, workspace, script string) (string, error) {
			return "browser result for " + script, nil
		},
	)

	wf, err := wm.StartPlanned(123, "ship feature", "claude", "/tmp/project", "telegram", []WorkflowStep{
		{ID: "plan", Name: "Plan", Kind: "copilot", Prompt: "make a plan"},
		{ID: "browse", Name: "Browse", Kind: "browser", Script: "open https://1.1.1.1\nwait 1s\nextract body", DependsOn: []string{"plan"}},
	}, nil)
	if err != nil {
		t.Fatalf("StartPlanned: %v", err)
	}

	got := waitWorkflowStatus(t, wm, wf.ID, 3*time.Second)
	if got.Status != WorkflowCompleted {
		t.Fatalf("workflow status = %s, want completed", got.Status)
	}
	if got.Steps[0].Status != WorkflowStepCompleted || got.Steps[1].Status != WorkflowStepCompleted {
		t.Fatalf("unexpected step states: %+v", got.Steps)
	}
	if !strings.Contains(got.ResultSummary, "Plan") || !strings.Contains(got.ResultSummary, "Browse") {
		t.Fatalf("unexpected summary: %q", got.ResultSummary)
	}
	if mem.Count(123) == 0 {
		t.Fatal("workflow completion should be stored in memory")
	}
	recent := mem.Recent(123, 1)
	if len(recent) != 1 || recent[0].Role != "tool" || recent[0].Kind != "workflow" {
		t.Fatalf("workflow memory entry = %+v", recent)
	}
}

func TestWorkflowManager_Cancel(t *testing.T) {
	dir := t.TempDir()
	wm, err := NewWorkflowManager(dir, nil)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}

	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			select {
			case <-ctx.Done():
				return "", context.Canceled
			case <-time.After(2 * time.Second):
				return "done", nil
			}
		},
		func(ctx context.Context, workspace, script string) (string, error) {
			return "", errors.New("unexpected browser call")
		},
	)

	wf, err := wm.StartPlanned(456, "cancel me", "claude", "/tmp/project", "telegram", []WorkflowStep{
		{ID: "slow", Name: "Slow", Kind: "copilot", Prompt: "long running"},
	}, nil)
	if err != nil {
		t.Fatalf("StartPlanned: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if !wm.Cancel(wf.ID) {
		t.Fatal("Cancel should succeed")
	}

	got := waitWorkflowStatus(t, wm, wf.ID, 3*time.Second)
	if got.Status != WorkflowCancelled {
		t.Fatalf("workflow status = %s, want cancelled", got.Status)
	}
	if got.Steps[0].Status != WorkflowStepCancelled {
		t.Fatalf("step status = %s, want cancelled", got.Steps[0].Status)
	}
}

func TestWorkflowManager_CancelDuringPlanning(t *testing.T) {
	dir := t.TempDir()
	wm, err := NewWorkflowManager(dir, nil)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}

	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			select {
			case <-ctx.Done():
				return "", context.Canceled
			case <-time.After(2 * time.Second):
				return "", nil
			}
		},
		nil,
	)

	wf, err := wm.StartRequest(777, "plan slowly", "claude", "/tmp/project", "telegram", nil)
	if err != nil {
		t.Fatalf("StartRequest: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if !wm.Cancel(wf.ID) {
		t.Fatal("Cancel should succeed during planning")
	}

	got := waitWorkflowStatus(t, wm, wf.ID, 3*time.Second)
	if got.Status != WorkflowCancelled {
		t.Fatalf("workflow status = %s, want cancelled", got.Status)
	}
}

func TestWorkflowManager_CapacityLimit(t *testing.T) {
	dir := t.TempDir()
	wm, err := NewWorkflowManager(dir, nil)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}

	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			select {
			case <-ctx.Done():
				return "", context.Canceled
			case <-time.After(2 * time.Second):
				return "done", nil
			}
		},
		nil,
	)

	var ids []string
	for i := 0; i < maxActiveWorkflowsPerUser; i++ {
		wf, err := wm.StartPlanned(999, "capacity", "claude", "/tmp/project", "telegram", []WorkflowStep{
			{ID: "slow", Name: "Slow", Kind: "copilot", Prompt: "wait"},
		}, nil)
		if err != nil {
			t.Fatalf("StartPlanned %d: %v", i, err)
		}
		ids = append(ids, wf.ID)
	}

	if _, err := wm.StartPlanned(999, "too many", "claude", "/tmp/project", "telegram", []WorkflowStep{
		{ID: "slow", Name: "Slow", Kind: "copilot", Prompt: "wait"},
	}, nil); !errors.Is(err, ErrWorkflowCapacity) {
		t.Fatalf("expected capacity error, got %v", err)
	}

	for _, id := range ids {
		if !wm.Cancel(id) {
			t.Fatalf("expected cancel to succeed for %s", id)
		}
		waitWorkflowStatus(t, wm, id, 3*time.Second)
	}
}

func TestWorkflowManager_GetReturnsDeepCopy(t *testing.T) {
	dir := t.TempDir()
	wm, err := NewWorkflowManager(dir, nil)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}

	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			return "done", nil
		},
		nil,
	)

	wf, err := wm.StartPlanned(321, "copy", "claude", "/tmp/project", "telegram", []WorkflowStep{
		{ID: "plan", Name: "Plan", Kind: "copilot"},
		{ID: "run", Name: "Run", Kind: "copilot", DependsOn: []string{"plan"}},
	}, nil)
	if err != nil {
		t.Fatalf("StartPlanned: %v", err)
	}

	got, ok := wm.Get(wf.ID)
	if !ok {
		t.Fatalf("Get() returned not found")
	}
	got.Steps[1].DependsOn[0] = "tampered"

	gotAgain, ok := wm.Get(wf.ID)
	if !ok {
		t.Fatalf("Get() returned not found on second read")
	}
	if gotAgain.Steps[1].DependsOn[0] != "plan" {
		t.Fatalf("expected manager state to stay unchanged, got %+v", gotAgain.Steps[1].DependsOn)
	}

	wm.Cancel(wf.ID)
	waitWorkflowStatus(t, wm, wf.ID, 3*time.Second)
}

func waitWorkflowStatus(t *testing.T, wm *WorkflowManager, id string, timeout time.Duration) Workflow {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if got, ok := wm.Get(id); ok {
			if got.Status == WorkflowCompleted || got.Status == WorkflowFailed || got.Status == WorkflowCancelled {
				return got
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	got, _ := wm.Get(id)
	t.Fatalf("timeout waiting for workflow completion: %+v", got)
	return Workflow{}
}
