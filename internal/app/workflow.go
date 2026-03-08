package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/garyellow/axle/internal/bot/skill"
)

const (
	workflowFile              = "workflows.json"
	maxActiveWorkflowsPerUser = 3
	maxActiveWorkflowsTotal   = 8
)

var ErrWorkflowCapacity = errors.New("workflow capacity reached")

// WorkflowStatus is the lifecycle state of a workflow.
type WorkflowStatus string

const (
	WorkflowPlanning  WorkflowStatus = "planning"
	WorkflowRunning   WorkflowStatus = "running"
	WorkflowCompleted WorkflowStatus = "completed"
	WorkflowFailed    WorkflowStatus = "failed"
	WorkflowCancelled WorkflowStatus = "cancelled"
)

func (s WorkflowStatus) Label() string {
	switch s {
	case WorkflowPlanning:
		return "🧠 規劃中"
	case WorkflowRunning:
		return "🔄 執行中"
	case WorkflowCompleted:
		return "✅ 完成"
	case WorkflowFailed:
		return "❌ 失敗"
	case WorkflowCancelled:
		return "🛑 已取消"
	default:
		return string(s)
	}
}

// WorkflowStepStatus is the lifecycle state of an individual workflow step.
type WorkflowStepStatus string

const (
	WorkflowStepPending   WorkflowStepStatus = "pending"
	WorkflowStepRunning   WorkflowStepStatus = "running"
	WorkflowStepCompleted WorkflowStepStatus = "completed"
	WorkflowStepFailed    WorkflowStepStatus = "failed"
	WorkflowStepCancelled WorkflowStepStatus = "cancelled"
)

func (s WorkflowStepStatus) Label() string {
	switch s {
	case WorkflowStepPending:
		return "⏳ 等待中"
	case WorkflowStepRunning:
		return "🔄 執行中"
	case WorkflowStepCompleted:
		return "✅ 完成"
	case WorkflowStepFailed:
		return "❌ 失敗"
	case WorkflowStepCancelled:
		return "🛑 已取消"
	default:
		return string(s)
	}
}

// WorkflowStep is a single background task inside a workflow.
type WorkflowStep struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Kind      string             `json:"kind"` // "copilot" or "browser"
	Prompt    string             `json:"prompt,omitempty"`
	Script    string             `json:"script,omitempty"`
	DependsOn []string           `json:"depends_on,omitempty"`
	Status    WorkflowStepStatus `json:"status"`
	Result    string             `json:"result,omitempty"`
	Error     string             `json:"error,omitempty"`
	StartedAt time.Time          `json:"started_at,omitempty"`
	DoneAt    time.Time          `json:"done_at,omitempty"`
}

// Workflow is a persistent background orchestration request.
type Workflow struct {
	ID            string         `json:"id"`
	UserID        int64          `json:"user_id"`
	Source        string         `json:"source,omitempty"`
	Request       string         `json:"request"`
	Model         string         `json:"model"`
	Workspace     string         `json:"workspace"`
	Status        WorkflowStatus `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	DoneAt        time.Time      `json:"done_at,omitempty"`
	PlanText      string         `json:"plan_text,omitempty"`
	Steps         []WorkflowStep `json:"steps"`
	ResultSummary string         `json:"result_summary,omitempty"`
	Error         string         `json:"error,omitempty"`
	cancelFn      context.CancelFunc
}

// WorkflowNotice is emitted to adapters during workflow progress.
type WorkflowNotice struct {
	WorkflowID string         `json:"workflow_id"`
	Status     WorkflowStatus `json:"status"`
	Message    string         `json:"message"`
}

type workflowPlanEnvelope struct {
	Steps []WorkflowStep `json:"steps"`
}

// WorkflowManager manages persistent background workflows.
type WorkflowManager struct {
	mu            sync.RWMutex
	filePath      string
	workflows     map[string]*Workflow
	memory        *MemoryStore
	copilotRunner func(ctx context.Context, workspace, model, prompt string) (string, error)
	browserRunner func(ctx context.Context, workspace, script string) (string, error)
}

// NewWorkflowManager creates a workflow manager backed by ~/.axle/workflows.json.
func NewWorkflowManager(axleDir string, memory *MemoryStore) (*WorkflowManager, error) {
	wm := &WorkflowManager{
		filePath:      filepath.Join(axleDir, workflowFile),
		workflows:     make(map[string]*Workflow),
		memory:        memory,
		copilotRunner: defaultWorkflowCopilotRunner,
		browserRunner: defaultWorkflowBrowserRunner,
	}
	if err := wm.load(); err != nil {
		return nil, err
	}
	return wm, nil
}

// SetRunners overrides execution runners. Useful for tests.
func (wm *WorkflowManager) SetRunners(
	copilotFn func(ctx context.Context, workspace, model, prompt string) (string, error),
	browserFn func(ctx context.Context, workspace, script string) (string, error),
) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if copilotFn != nil {
		wm.copilotRunner = copilotFn
	}
	if browserFn != nil {
		wm.browserRunner = browserFn
	}
}

// StartRequest creates a new workflow, plans it with Copilot, and runs it in the background.
func (wm *WorkflowManager) StartRequest(userID int64, request, model, workspace, source string, notify func(WorkflowNotice)) (*Workflow, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return nil, fmt.Errorf("工作流需求不可為空")
	}
	if model == "" {
		model = skill.DefaultModel
	}
	if source == "" {
		source = "telegram"
	}

	ctx, cancel := context.WithCancel(context.Background())
	wf := &Workflow{
		ID:        fmt.Sprintf("wf-%d", time.Now().UnixNano()),
		UserID:    userID,
		Source:    source,
		Request:   request,
		Model:     model,
		Workspace: workspace,
		Status:    WorkflowPlanning,
		CreatedAt: time.Now(),
		cancelFn:  cancel,
	}

	wm.mu.Lock()
	if err := wm.ensureCapacityLocked(userID); err != nil {
		wm.mu.Unlock()
		return nil, err
	}
	wm.workflows[wf.ID] = wf
	if err := wm.persistLocked(); err != nil {
		delete(wm.workflows, wf.ID)
		wm.mu.Unlock()
		return nil, err
	}
	cp := copyWorkflow(wf)
	wm.mu.Unlock()

	go wm.runRequestSafely(ctx, wf.ID, request, model, workspace, source, notify)
	return &cp, nil
}

// StartPlanned creates and runs a workflow from precomputed steps.
func (wm *WorkflowManager) StartPlanned(userID int64, request, model, workspace, source string, steps []WorkflowStep, notify func(WorkflowNotice)) (*Workflow, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return nil, fmt.Errorf("工作流需求不可為空")
	}
	if model == "" {
		model = skill.DefaultModel
	}
	if source == "" {
		source = "telegram"
	}
	steps = normalizeWorkflowSteps(steps)
	if len(steps) == 0 {
		return nil, fmt.Errorf("工作流至少需要一個步驟")
	}

	ctx, cancel := context.WithCancel(context.Background())
	wf := &Workflow{
		ID:        fmt.Sprintf("wf-%d", time.Now().UnixNano()),
		UserID:    userID,
		Source:    source,
		Request:   request,
		Model:     model,
		Workspace: workspace,
		Status:    WorkflowRunning,
		CreatedAt: time.Now(),
		PlanText:  "manual",
		Steps:     steps,
		cancelFn:  cancel,
	}

	wm.mu.Lock()
	if err := wm.ensureCapacityLocked(userID); err != nil {
		wm.mu.Unlock()
		return nil, err
	}
	wm.workflows[wf.ID] = wf
	if err := wm.persistLocked(); err != nil {
		delete(wm.workflows, wf.ID)
		wm.mu.Unlock()
		return nil, err
	}
	cp := copyWorkflow(wf)
	wm.mu.Unlock()

	go wm.executeWorkflowSafely(ctx, wf.ID, source, notify)
	return &cp, nil
}

// List returns workflows for the given user, newest first.
func (wm *WorkflowManager) List(userID int64) []Workflow {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make([]Workflow, 0)
	for _, wf := range wm.workflows {
		if wf.UserID == userID {
			cp := copyWorkflow(wf)
			result = append(result, cp)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// Get returns a workflow by ID.
func (wm *WorkflowManager) Get(id string) (Workflow, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	wf, ok := wm.workflows[id]
	if !ok {
		return Workflow{}, false
	}
	cp := copyWorkflow(wf)
	return cp, true
}

// Cancel cancels a workflow if it is still active.
func (wm *WorkflowManager) Cancel(id string) bool {
	wm.mu.Lock()
	wf, ok := wm.workflows[id]
	if !ok || (wf.Status != WorkflowPlanning && wf.Status != WorkflowRunning) {
		wm.mu.Unlock()
		return false
	}
	cancel := wf.cancelFn
	wm.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return true
}

// RunningCount returns the number of active workflows for a user.
func (wm *WorkflowManager) RunningCount(userID int64) int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	count := 0
	for _, wf := range wm.workflows {
		if wf.UserID == userID && (wf.Status == WorkflowPlanning || wf.Status == WorkflowRunning) {
			count++
		}
	}
	return count
}

func (wm *WorkflowManager) runRequest(ctx context.Context, workflowID, request, model, workspace, source string, notify func(WorkflowNotice)) {
	steps, planText, err := wm.planWorkflow(ctx, request, model, workspace)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled {
			wm.finishWorkflow(workflowID, WorkflowCancelled, "", "", source)
			wm.notify(notify, WorkflowNotice{
				WorkflowID: workflowID,
				Status:     WorkflowCancelled,
				Message:    fmt.Sprintf("🛑 工作流 `%s` 已取消。", workflowID),
			})
			return
		}
		wm.finishWorkflow(workflowID, WorkflowFailed, "", "工作流規劃失敗: "+err.Error(), source)
		wm.notify(notify, WorkflowNotice{
			WorkflowID: workflowID,
			Status:     WorkflowFailed,
			Message:    fmt.Sprintf("❌ 工作流 `%s` 規劃失敗：%s", workflowID, err.Error()),
		})
		return
	}

	wm.mu.Lock()
	wf, ok := wm.workflows[workflowID]
	if !ok {
		wm.mu.Unlock()
		return
	}
	wf.PlanText = planText
	wf.Steps = normalizeWorkflowSteps(steps)
	wf.Status = WorkflowRunning
	_ = wm.persistLocked()
	wm.mu.Unlock()

	wm.notify(notify, WorkflowNotice{
		WorkflowID: workflowID,
		Status:     WorkflowRunning,
		Message:    fmt.Sprintf("🧭 工作流 `%s` 已完成規劃，共 %d 個步驟，開始背景執行。", workflowID, len(steps)),
	})

	wm.executeWorkflow(ctx, workflowID, notify)
}

func (wm *WorkflowManager) runRequestSafely(ctx context.Context, workflowID, request, model, workspace, source string, notify func(WorkflowNotice)) {
	defer wm.recoverWorkflowPanic(workflowID, source, notify)
	wm.runRequest(ctx, workflowID, request, model, workspace, source, notify)
}

func (wm *WorkflowManager) executeWorkflowSafely(ctx context.Context, workflowID, source string, notify func(WorkflowNotice)) {
	defer wm.recoverWorkflowPanic(workflowID, source, notify)
	wm.executeWorkflow(ctx, workflowID, notify)
}

func (wm *WorkflowManager) executeWorkflow(ctx context.Context, workflowID string, notify func(WorkflowNotice)) {
	for {
		wm.mu.Lock()
		wf, ok := wm.workflows[workflowID]
		if !ok {
			wm.mu.Unlock()
			return
		}

		stepIdx := findReadyStep(wf.Steps)
		if stepIdx < 0 {
			if hasPendingSteps(wf.Steps) {
				wm.mu.Unlock()
				wm.finishWorkflow(workflowID, WorkflowFailed, "", "工作流依賴無法解析或存在循環依賴", wf.Source)
				wm.notify(notify, WorkflowNotice{
					WorkflowID: workflowID,
					Status:     WorkflowFailed,
					Message:    fmt.Sprintf("❌ 工作流 `%s` 失敗：存在未完成依賴。", workflowID),
				})
				return
			}
			summary := summarizeWorkflow(wf)
			wm.mu.Unlock()
			wm.finishWorkflow(workflowID, WorkflowCompleted, summary, "", wf.Source)
			wm.notify(notify, WorkflowNotice{
				WorkflowID: workflowID,
				Status:     WorkflowCompleted,
				Message:    fmt.Sprintf("✅ 工作流 `%s` 完成\n\n%s", workflowID, summary),
			})
			return
		}

		step := wf.Steps[stepIdx]
		wf.Steps[stepIdx].Status = WorkflowStepRunning
		wf.Steps[stepIdx].StartedAt = time.Now()
		_ = wm.persistLocked()
		wm.mu.Unlock()

		result, err := wm.runStep(ctx, workflowID, step)
		if err != nil {
			if ctx.Err() == context.Canceled {
				wm.markStepCancelled(workflowID, step.ID)
				wm.finishWorkflow(workflowID, WorkflowCancelled, "", "", "")
				wm.notify(notify, WorkflowNotice{
					WorkflowID: workflowID,
					Status:     WorkflowCancelled,
					Message:    fmt.Sprintf("🛑 工作流 `%s` 已取消。", workflowID),
				})
				return
			}

			wm.markStepFailed(workflowID, step.ID, err.Error())
			wm.finishWorkflow(workflowID, WorkflowFailed, "", err.Error(), "")
			wm.notify(notify, WorkflowNotice{
				WorkflowID: workflowID,
				Status:     WorkflowFailed,
				Message:    fmt.Sprintf("❌ 工作流 `%s` 在步驟「%s」失敗：%s", workflowID, step.Name, err.Error()),
			})
			return
		}

		wm.markStepCompleted(workflowID, step.ID, result)
		wm.notify(notify, WorkflowNotice{
			WorkflowID: workflowID,
			Status:     WorkflowRunning,
			Message:    fmt.Sprintf("✅ 工作流 `%s` 步驟完成：%s", workflowID, step.Name),
		})
	}
}

func (wm *WorkflowManager) runStep(ctx context.Context, workflowID string, step WorkflowStep) (string, error) {
	wm.mu.RLock()
	wf, ok := wm.workflows[workflowID]
	if !ok {
		wm.mu.RUnlock()
		return "", fmt.Errorf("工作流不存在")
	}
	model := wf.Model
	workspace := wf.Workspace
	depSummary := collectDependencyResults(wf.Steps, step.DependsOn)
	wm.mu.RUnlock()

	switch strings.ToLower(step.Kind) {
	case "copilot", "":
		prompt := strings.TrimSpace(step.Prompt)
		if prompt == "" {
			prompt = strings.TrimSpace(wf.Request)
		}
		if depSummary != "" {
			prompt = depSummary + "\n\nCurrent step:\n" + prompt
		}
		return wm.copilotRunner(ctx, workspace, model, prompt)
	case "browser":
		script := strings.TrimSpace(step.Script)
		if script == "" {
			return "", fmt.Errorf("browser 步驟缺少 script")
		}
		return wm.browserRunner(ctx, workspace, script)
	default:
		return "", fmt.Errorf("不支援的工作流步驟類型: %s", step.Kind)
	}
}

func (wm *WorkflowManager) planWorkflow(ctx context.Context, request, model, workspace string) ([]WorkflowStep, string, error) {
	prompt := buildWorkflowPlannerPrompt(request)
	raw, err := wm.copilotRunner(ctx, workspace, model, prompt)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, "", context.Canceled
		}
		steps := fallbackWorkflowSteps(request)
		return steps, "fallback", nil
	}

	steps, err := parseWorkflowPlan(raw)
	if err != nil || len(steps) == 0 {
		return fallbackWorkflowSteps(request), raw, nil
	}
	return steps, raw, nil
}

func (wm *WorkflowManager) markStepCompleted(workflowID, stepID, result string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wf, ok := wm.workflows[workflowID]; ok {
		for i := range wf.Steps {
			if wf.Steps[i].ID == stepID {
				wf.Steps[i].Status = WorkflowStepCompleted
				wf.Steps[i].DoneAt = time.Now()
				wf.Steps[i].Result = truncateStr(strings.TrimSpace(result), 6000)
				break
			}
		}
		_ = wm.persistLocked()
	}
}

func (wm *WorkflowManager) markStepFailed(workflowID, stepID, errMsg string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wf, ok := wm.workflows[workflowID]; ok {
		for i := range wf.Steps {
			if wf.Steps[i].ID == stepID {
				wf.Steps[i].Status = WorkflowStepFailed
				wf.Steps[i].DoneAt = time.Now()
				wf.Steps[i].Error = truncateStr(strings.TrimSpace(errMsg), 1000)
				break
			}
		}
		_ = wm.persistLocked()
	}
}

func (wm *WorkflowManager) markStepCancelled(workflowID, stepID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wf, ok := wm.workflows[workflowID]; ok {
		for i := range wf.Steps {
			switch {
			case wf.Steps[i].ID == stepID:
				wf.Steps[i].Status = WorkflowStepCancelled
				wf.Steps[i].DoneAt = time.Now()
			case wf.Steps[i].Status == WorkflowStepPending:
				wf.Steps[i].Status = WorkflowStepCancelled
				wf.Steps[i].DoneAt = time.Now()
			}
		}
		_ = wm.persistLocked()
	}
}

func (wm *WorkflowManager) finishWorkflow(workflowID string, status WorkflowStatus, summary, errMsg, source string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wf, ok := wm.workflows[workflowID]
	if !ok {
		return
	}
	wf.Status = status
	wf.DoneAt = time.Now()
	wf.ResultSummary = truncateStr(strings.TrimSpace(summary), 8000)
	wf.Error = truncateStr(strings.TrimSpace(errMsg), 1000)
	wf.cancelFn = nil
	_ = wm.persistLocked()

	if wm.memory != nil {
		content := ""
		switch status {
		case WorkflowCompleted:
			content = fmt.Sprintf("Workflow %s completed for request: %s\n%s", workflowID, wf.Request, wf.ResultSummary)
		case WorkflowFailed:
			content = fmt.Sprintf("Workflow %s failed for request: %s\n%s", workflowID, wf.Request, wf.Error)
		case WorkflowCancelled:
			content = fmt.Sprintf("Workflow %s was cancelled for request: %s", workflowID, wf.Request)
		}
		if content != "" {
			entrySource := source
			if entrySource == "" {
				entrySource = wf.Source
			}
			if err := wm.memory.AddDetailed(wf.UserID, MemoryEntry{
				Role:      "system",
				Content:   content,
				Model:     wf.Model,
				Kind:      "workflow",
				Source:    entrySource,
				Workspace: wf.Workspace,
				Tags:      []string{"workflow", workflowID},
			}); err != nil {
				slog.Warn("⚠️ 工作流記憶寫入失敗", "workflow_id", workflowID, "user_id", wf.UserID, "error", err)
			}
		}
	}
}

func (wm *WorkflowManager) recoverWorkflowPanic(workflowID, source string, notify func(WorkflowNotice)) {
	if r := recover(); r != nil {
		slog.Error("workflow goroutine panic", "workflow_id", workflowID, "panic", r, "stack", string(debug.Stack()))
		errMsg := fmt.Sprintf("工作流內部異常：%v", r)
		wm.finishWorkflow(workflowID, WorkflowFailed, "", errMsg, source)
		wm.notify(notify, WorkflowNotice{
			WorkflowID: workflowID,
			Status:     WorkflowFailed,
			Message:    fmt.Sprintf("❌ 工作流 `%s` 因內部錯誤中止：%v", workflowID, r),
		})
	}
}

func (wm *WorkflowManager) load() error {
	data, err := os.ReadFile(wm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("讀取工作流失敗: %w", err)
	}

	var list []Workflow
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("解析工作流失敗: %w", err)
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()
	changed := false
	for i := range list {
		wf := list[i]
		wf.Steps = normalizeWorkflowSteps(wf.Steps)
		if wf.Status == WorkflowPlanning || wf.Status == WorkflowRunning {
			changed = true
			wf.Status = WorkflowFailed
			wf.Error = "工作流因 Axle 重啟而中斷"
			wf.DoneAt = time.Now()
			for idx := range wf.Steps {
				if wf.Steps[idx].Status == WorkflowStepPending || wf.Steps[idx].Status == WorkflowStepRunning {
					wf.Steps[idx].Status = WorkflowStepCancelled
					wf.Steps[idx].DoneAt = time.Now()
				}
			}
		}
		cp := wf
		wm.workflows[cp.ID] = &cp
	}
	if changed {
		if err := wm.persistLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (wm *WorkflowManager) persistLocked() error {
	list := make([]Workflow, 0, len(wm.workflows))
	for _, wf := range wm.workflows {
		list = append(list, copyWorkflow(wf))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化工作流失敗: %w", err)
	}
	return writeFileAtomic(wm.filePath, data, 0600)
}

func (wm *WorkflowManager) notify(notify func(WorkflowNotice), notice WorkflowNotice) {
	if notify != nil {
		notify(notice)
	}
}

func copyWorkflow(wf *Workflow) Workflow {
	cp := *wf
	cp.cancelFn = nil
	cp.Steps = make([]WorkflowStep, len(wf.Steps))
	copy(cp.Steps, wf.Steps)
	return cp
}

func (wm *WorkflowManager) ensureCapacityLocked(userID int64) error {
	userActive := 0
	totalActive := 0
	for _, wf := range wm.workflows {
		if wf.Status != WorkflowPlanning && wf.Status != WorkflowRunning {
			continue
		}
		totalActive++
		if wf.UserID == userID {
			userActive++
		}
	}

	switch {
	case userActive >= maxActiveWorkflowsPerUser:
		return fmt.Errorf("%w: 目前最多只能同時執行 %d 個背景工作流", ErrWorkflowCapacity, maxActiveWorkflowsPerUser)
	case totalActive >= maxActiveWorkflowsTotal:
		return fmt.Errorf("%w: 背景工作流佇列已滿，請稍後再試", ErrWorkflowCapacity)
	default:
		return nil
	}
}

func findReadyStep(steps []WorkflowStep) int {
	for i, step := range steps {
		if step.Status != WorkflowStepPending {
			continue
		}
		ready := true
		for _, dep := range step.DependsOn {
			if !isStepCompleted(steps, dep) {
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

func hasPendingSteps(steps []WorkflowStep) bool {
	for _, step := range steps {
		if step.Status == WorkflowStepPending {
			return true
		}
	}
	return false
}

func isStepCompleted(steps []WorkflowStep, stepID string) bool {
	for _, step := range steps {
		if step.ID == stepID {
			return step.Status == WorkflowStepCompleted
		}
	}
	return false
}

func collectDependencyResults(steps []WorkflowStep, deps []string) string {
	if len(deps) == 0 {
		return ""
	}

	var parts []string
	for _, dep := range deps {
		for _, step := range steps {
			if step.ID == dep && step.Result != "" {
				parts = append(parts, fmt.Sprintf("[%s]\n%s", step.Name, step.Result))
				break
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Previous workflow step outputs:\n" + strings.Join(parts, "\n\n")
}

func summarizeWorkflow(wf *Workflow) string {
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

func normalizeWorkflowSteps(steps []WorkflowStep) []WorkflowStep {
	result := make([]WorkflowStep, 0, len(steps))
	for i, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			step.Name = fmt.Sprintf("步驟 %d", i+1)
		}
		if strings.TrimSpace(step.ID) == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if step.Status == "" {
			step.Status = WorkflowStepPending
		}
		step.Kind = strings.ToLower(strings.TrimSpace(step.Kind))
		if step.Kind == "" {
			step.Kind = "copilot"
		}
		result = append(result, step)
	}
	return result
}

func buildWorkflowPlannerPrompt(request string) string {
	return `You are planning a lightweight background workflow for Axle.
Return ONLY valid JSON with this exact shape:
{"steps":[{"id":"step-1","name":"...","kind":"copilot","prompt":"...","depends_on":[]}]}

Rules:
- 2 to 4 steps only.
- Allowed kinds: "copilot" or "browser".
- Use "browser" only when web inspection is clearly needed.
- For browser steps, use the Axle browser DSL only:
  open https://example.com
  wait 2s
  extract body
  screenshot .axle/browser/example.png
- Keep prompts concise and actionable.
- Use depends_on only when necessary.

User request:
` + request
}

func fallbackWorkflowSteps(request string) []WorkflowStep {
	return normalizeWorkflowSteps([]WorkflowStep{
		{
			ID:     "analyze",
			Name:   "分析需求",
			Kind:   "copilot",
			Prompt: "Analyze the following request and produce a concrete implementation plan with key risks and files likely affected:\n" + request,
		},
		{
			ID:        "execute",
			Name:      "執行方案",
			Kind:      "copilot",
			Prompt:    "Execute the following request and provide the concrete result, decisions made, and any verification notes:\n" + request,
			DependsOn: []string{"analyze"},
		},
	})
}

func parseWorkflowPlan(raw string) ([]WorkflowStep, error) {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("找不到 JSON 物件")
	}
	clean = clean[start : end+1]

	var env workflowPlanEnvelope
	if err := json.Unmarshal([]byte(clean), &env); err != nil {
		return nil, err
	}
	return normalizeWorkflowSteps(env.Steps), nil
}

func defaultWorkflowCopilotRunner(ctx context.Context, workspace, model, prompt string) (string, error) {
	chunks, err := skill.RunCopilot(ctx, workspace, model, prompt)
	if err != nil {
		return "", err
	}
	return strings.Join(chunks, "\n"), nil
}

func defaultWorkflowBrowserRunner(ctx context.Context, workspace, script string) (string, error) {
	result, err := skill.RunBrowserScript(ctx, workspace, script)
	if err != nil {
		return "", err
	}
	return result.Summary(), nil
}
