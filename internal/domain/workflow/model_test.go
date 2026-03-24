package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeStepsAndFindReadyStep(t *testing.T) {
	steps := NormalizeSteps([]Step{
		{Name: "", Kind: "", Prompt: "analyze"},
		{ID: "run", Name: "Run", Kind: "copilot", DependsOn: []string{"step-1"}},
	})

	if steps[0].ID != "step-1" || steps[0].Name != "步驟 1" || steps[0].Kind != "copilot" || steps[0].Status != StepPending {
		t.Fatalf("unexpected normalized step: %+v", steps[0])
	}

	if got := FindReadyStep(steps); got != 0 {
		t.Fatalf("FindReadyStep() = %d, want 0", got)
	}

	steps[0].Status = StepCompleted
	if got := FindReadyStep(steps); got != 1 {
		t.Fatalf("FindReadyStep() after dep = %d, want 1", got)
	}
}

func TestEnsureCapacity(t *testing.T) {
	workflows := []Workflow{
		{UserID: 7, Status: StatusPlanning},
		{UserID: 7, Status: StatusRunning},
		{UserID: 7, Status: StatusRunning},
	}

	err := EnsureCapacity(workflows, 7)
	if !errors.Is(err, ErrCapacity) {
		t.Fatalf("EnsureCapacity() error = %v, want capacity", err)
	}

	if err := EnsureCapacity(workflows, 8); err != nil {
		t.Fatalf("EnsureCapacity() other user error = %v", err)
	}
}

func TestSummarize(t *testing.T) {
	summary := Summarize(Workflow{
		Steps: []Step{
			{Name: "Plan", Status: StepCompleted, Result: "first line\nsecond line"},
			{Name: "Run", Status: StepFailed, Error: strings.Repeat("x", 220)},
		},
	})

	if !strings.Contains(summary, "Plan") || !strings.Contains(summary, "Run") {
		t.Fatalf("unexpected summary = %q", summary)
	}
	if !strings.Contains(summary, StepCompleted.Label()) || !strings.Contains(summary, StepFailed.Label()) {
		t.Fatalf("missing labels in summary = %q", summary)
	}
}
