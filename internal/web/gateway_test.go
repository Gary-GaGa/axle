package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/garyellow/axle/internal/app"
)

func TestGatewayAuthAndStatus(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/chat/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/chat/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/chat/status?token=secret-token", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec = httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected query token to be rejected, got %d", rec.Code)
	}
}

func TestGatewayMemorySearch(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.memory.AddDetailed(app.WebGatewayUserID, app.MemoryEntry{
		Role:    "tool",
		Content: "Browser extracted dashboard release notes",
		Kind:    "browser",
		Source:  "web",
	}); err != nil {
		t.Fatalf("seed memory: %v", err)
	}

	body := bytes.NewBufferString(`{"query":"release notes","limit":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/memory/search", body)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Hits []app.MemorySearchHit `json:"hits"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Hits) == 0 {
		t.Fatal("expected search hits")
	}
}

func TestGatewayWorkflowCreateAndList(t *testing.T) {
	srv := newTestServer(t)
	srv.workflows.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) {
			if bytes.Contains([]byte(prompt), []byte(`"steps"`)) {
				return `{"steps":[{"id":"step-1","name":"Analyze","kind":"copilot","prompt":"Analyze request"}]}`, nil
			}
			return "done", nil
		},
		func(ctx context.Context, workspace, script string) (string, error) {
			return "browser done", nil
		},
	)

	createBody := bytes.NewBufferString(`{"request":"review deployment risk"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/workflows", createBody)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Workflows []app.Workflow `json:"workflows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode workflows: %v", err)
	}
	if len(resp.Workflows) == 0 {
		t.Fatal("expected at least one workflow")
	}
	waitForWorkflowTerminal(t, srv.workflows, resp.Workflows[0].ID)
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()
	mem, err := app.NewMemoryStore(dir)
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	wm, err := app.NewWorkflowManager(dir, mem)
	if err != nil {
		t.Fatalf("NewWorkflowManager: %v", err)
	}
	wm.SetRunners(
		func(ctx context.Context, workspace, model, prompt string) (string, error) { return "done", nil },
		func(ctx context.Context, workspace, script string) (string, error) { return "browser done", nil },
	)

	return NewServer(
		"127.0.0.1:0",
		nil,
		&app.TaskManager{},
		mem,
		wm,
		filepath.Join(dir, "workspace"),
		"secret-token",
	)
}

func waitForWorkflowTerminal(t *testing.T, wm *app.WorkflowManager, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		wf, ok := wm.Get(id)
		if ok && wf.Status != app.WorkflowPlanning && wf.Status != app.WorkflowRunning {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("workflow %s did not reach terminal state", id)
}
