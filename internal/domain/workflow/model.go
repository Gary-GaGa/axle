package workflow

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MaxActiveWorkflowsPerUser = 3
	MaxActiveWorkflowsTotal   = 8
)

var ErrCapacity = errors.New("workflow capacity reached")

type Status string

const (
	StatusPlanning  Status = "planning"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

func (s Status) Label() string {
	switch s {
	case StatusPlanning:
		return "🧠 規劃中"
	case StatusRunning:
		return "🔄 執行中"
	case StatusCompleted:
		return "✅ 完成"
	case StatusFailed:
		return "❌ 失敗"
	case StatusCancelled:
		return "🛑 已取消"
	default:
		return string(s)
	}
}

func (s Status) IsActive() bool {
	return s == StatusPlanning || s == StatusRunning
}

type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepCancelled StepStatus = "cancelled"
)

func (s StepStatus) Label() string {
	switch s {
	case StepPending:
		return "⏳ 等待中"
	case StepRunning:
		return "🔄 執行中"
	case StepCompleted:
		return "✅ 完成"
	case StepFailed:
		return "❌ 失敗"
	case StepCancelled:
		return "🛑 已取消"
	default:
		return string(s)
	}
}

type Step struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Kind      string     `json:"kind"`
	Prompt    string     `json:"prompt,omitempty"`
	Script    string     `json:"script,omitempty"`
	DependsOn []string   `json:"depends_on,omitempty"`
	Status    StepStatus `json:"status"`
	Result    string     `json:"result,omitempty"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	DoneAt    time.Time  `json:"done_at,omitempty"`
}

type Workflow struct {
	ID            string    `json:"id"`
	UserID        int64     `json:"user_id"`
	Source        string    `json:"source,omitempty"`
	Request       string    `json:"request"`
	Model         string    `json:"model"`
	Workspace     string    `json:"workspace"`
	Status        Status    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	DoneAt        time.Time `json:"done_at,omitempty"`
	PlanText      string    `json:"plan_text,omitempty"`
	Steps         []Step    `json:"steps"`
	ResultSummary string    `json:"result_summary,omitempty"`
	Error         string    `json:"error,omitempty"`
}

func NormalizeSteps(steps []Step) []Step {
	result := make([]Step, 0, len(steps))
	for i, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			step.Name = fmt.Sprintf("步驟 %d", i+1)
		}
		if strings.TrimSpace(step.ID) == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if step.Status == "" {
			step.Status = StepPending
		}
		step.Kind = strings.ToLower(strings.TrimSpace(step.Kind))
		if step.Kind == "" {
			step.Kind = "copilot"
		}
		result = append(result, step)
	}
	return result
}

func FindReadyStep(steps []Step) int {
	for i, step := range steps {
		if step.Status != StepPending {
			continue
		}
		ready := true
		for _, dep := range step.DependsOn {
			if !IsStepCompleted(steps, dep) {
				ready = false
				break
			}
		}
		if ready {
			return i
		}
	}
	return -1
}

func HasPendingSteps(steps []Step) bool {
	for _, step := range steps {
		if step.Status == StepPending {
			return true
		}
	}
	return false
}

func IsStepCompleted(steps []Step, stepID string) bool {
	for _, step := range steps {
		if step.ID == stepID {
			return step.Status == StepCompleted
		}
	}
	return false
}

func EnsureCapacity(workflows []Workflow, userID int64) error {
	userActive := 0
	totalActive := 0
	for _, wf := range workflows {
		if !wf.Status.IsActive() {
			continue
		}
		totalActive++
		if wf.UserID == userID {
			userActive++
		}
	}

	switch {
	case userActive >= MaxActiveWorkflowsPerUser:
		return fmt.Errorf("%w: 目前最多只能同時執行 %d 個背景工作流", ErrCapacity, MaxActiveWorkflowsPerUser)
	case totalActive >= MaxActiveWorkflowsTotal:
		return fmt.Errorf("%w: 背景工作流佇列已滿，請稍後再試", ErrCapacity)
	default:
		return nil
	}
}

func Summarize(wf Workflow) string {
	var sb strings.Builder
	sb.WriteString("📋 結果摘要\n")
	for _, step := range wf.Steps {
		sb.WriteString(fmt.Sprintf("• %s — %s\n", step.Name, step.Status.Label()))
		switch {
		case step.Result != "":
			sb.WriteString("  " + truncateStr(strings.ReplaceAll(step.Result, "\n", " "), 180) + "\n")
		case step.Error != "":
			sb.WriteString("  " + truncateStr(step.Error, 180) + "\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
