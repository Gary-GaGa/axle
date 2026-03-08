package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/skill"
)

type gatewayChatRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
}

type gatewaySearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type gatewayBrowserRequest struct {
	Script string `json:"script"`
}

type gatewayWorkflowRequest struct {
	Request string `json:"request"`
	Model   string `json:"model,omitempty"`
}

type gatewayCancelRequest struct {
	ID string `json:"id"`
}

func (s *Server) requireGatewayAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.gatewayToken == "" || !constantTokenMatch(s.authToken(r), s.gatewayToken) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": "unauthorized",
			})
			return
		}
		next(w, r)
	}
}

func constantTokenMatch(got, expected string) bool {
	if got == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func (s *Server) recordMemory(entry app.MemoryEntry) {
	if s.memory == nil {
		return
	}
	if err := s.memory.AddDetailed(app.WebGatewayUserID, entry); err != nil {
		slog.Warn("⚠️ Web Gateway 記憶寫入失敗", "kind", entry.Kind, "source", entry.Source, "error", err)
	}
}

func (s *Server) authToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func (s *Server) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	running := false
	taskName := ""
	elapsedSecs := 0.0
	if s.tasks != nil {
		var elapsed time.Duration
		running, taskName, elapsed = s.tasks.Status()
		elapsedSecs = elapsed.Seconds()
	}

	resp := map[string]any{
		"workspace":         s.workspace,
		"default_model":     s.defaultModel,
		"task_running":      running,
		"task_name":         taskName,
		"task_elapsed_secs": elapsedSecs,
		"memory_count":      0,
		"workflow_count":    0,
	}
	if s.memory != nil {
		resp["memory_count"] = s.memory.Count(app.WebGatewayUserID)
	}
	if s.workflows != nil {
		resp["workflow_count"] = s.workflows.RunningCount(app.WebGatewayUserID)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req gatewayChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}

	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "prompt is required"})
		return
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = s.defaultModel
	}
	slog.Info("🌉 Web Chat", "model", model, "prompt_len", len(req.Prompt))

	ctx, done, ok := s.tryStartTask("WebChat")
	if !ok {
		running, name, elapsed := s.tasks.Status()
		_ = running
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "task busy",
			"task":    name,
			"elapsed": elapsed.Seconds(),
		})
		return
	}
	defer done()

	fullPrompt := req.Prompt
	if s.memory != nil {
		fullPrompt = s.memory.BuildContext(app.WebGatewayUserID, 8) + s.memory.BuildRAGContext(app.WebGatewayUserID, req.Prompt, 4) + req.Prompt
		s.recordMemory(app.MemoryEntry{
			Role:      "user",
			Content:   req.Prompt,
			Model:     model,
			Kind:      "chat",
			Source:    "web",
			Workspace: s.workspace,
		})
	}

	chunks, err := skill.RunCopilot(ctx, s.workspace, model, fullPrompt)
	if err != nil {
		if s.rpg != nil {
			s.rpg.EmitEvent("copilot_stream", "web chat failed", false)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	reply := strings.Join(chunks, "\n")
	if s.memory != nil {
		s.recordMemory(app.MemoryEntry{
			Role:      "assistant",
			Content:   reply,
			Model:     model,
			Kind:      "chat",
			Source:    "web",
			Workspace: s.workspace,
		})
	}
	if s.rpg != nil {
		s.rpg.EmitEvent("copilot_stream", "web chat", true)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reply": reply,
		"model": model,
	})
}

func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.memory == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "memory unavailable"})
		return
	}

	var req gatewaySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "query is required"})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 8
	}
	slog.Info("🌉 Web Memory Search", "query", req.Query, "limit", req.Limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"hits": s.memory.Search(app.WebGatewayUserID, req.Query, req.Limit),
	})
}

func (s *Server) handleMemoryRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.memory == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "memory unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": s.memory.Recent(app.WebGatewayUserID, 20),
	})
}

func (s *Server) handleMemoryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.memory == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "memory unavailable"})
		return
	}
	if err := s.memory.Clear(app.WebGatewayUserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleBrowserRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	var req gatewayBrowserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(req.Script) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "script is required"})
		return
	}
	slog.Info("🌉 Web Browser", "script_len", len(req.Script))

	ctx, done, ok := s.tryStartTask("WebBrowser")
	if !ok {
		running, name, elapsed := s.tasks.Status()
		_ = running
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "task busy",
			"task":    name,
			"elapsed": elapsed.Seconds(),
		})
		return
	}
	defer done()

	result, err := skill.RunBrowserScript(ctx, s.workspace, req.Script)
	if err != nil {
		if s.rpg != nil {
			s.rpg.EmitEvent("browser", "web browser failed", false)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	s.recordMemory(app.MemoryEntry{
		Role:      "tool",
		Content:   result.Summary(),
		Kind:      "browser",
		Source:    "web",
		Workspace: s.workspace,
		Tags:      append([]string{"browser"}, result.Screenshots...),
	})
	if s.rpg != nil {
		s.rpg.EmitEvent("browser", result.URL, true)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleWorkflowListOrCreate(w http.ResponseWriter, r *http.Request) {
	if s.workflows == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "workflow unavailable"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"workflows": s.workflows.List(app.WebGatewayUserID),
		})
	case http.MethodPost:
		var req gatewayWorkflowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		req.Request = strings.TrimSpace(req.Request)
		if req.Request == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request is required"})
			return
		}
		slog.Info("🌉 Web Workflow Create", "request_len", len(req.Request))
		wf, err := s.workflows.StartRequest(app.WebGatewayUserID, req.Request, req.Model, s.workspace, "web", nil)
		if err != nil {
			if errors.Is(err, app.ErrWorkflowCapacity) {
				writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		if s.rpg != nil {
			s.rpg.EmitEvent("workflow", req.Request, true)
		}
		writeJSON(w, http.StatusAccepted, wf)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleWorkflowCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	if s.workflows == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "workflow unavailable"})
		return
	}

	var req gatewayCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "workflow id is required"})
		return
	}
	wf, ok := s.workflows.Get(req.ID)
	if !ok || wf.UserID != app.WebGatewayUserID {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "workflow not found"})
		return
	}
	slog.Info("🌉 Web Workflow Cancel", "id", req.ID)
	if !s.workflows.Cancel(req.ID) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "workflow not found or not running"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) tryStartTask(name string) (context.Context, func(), bool) {
	if s.tasks == nil {
		return context.Background(), func() {}, true
	}
	ctx, ok := s.tasks.TryStart(name)
	if !ok {
		return nil, nil, false
	}
	return ctx, s.tasks.Done, true
}
