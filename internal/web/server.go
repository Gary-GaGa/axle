package web

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/garyellow/axle/internal/bot/skill"
	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: allowWebSocketOrigin,
}

// Server is the Axle web server: RPG dashboard + authenticated web gateway.
type Server struct {
	rpg          *app.RPGManager
	tasks        *app.TaskManager
	memory       *app.MemoryStore
	workflows    *app.WorkflowManager
	workspace    string
	defaultModel string
	gatewayToken string
	addr         string
	mux          *http.ServeMux
	server       *http.Server
	static       http.Handler
}

// NewServer creates a new Axle web server.
func NewServer(
	addr string,
	rpg *app.RPGManager,
	tasks *app.TaskManager,
	memory *app.MemoryStore,
	workflows *app.WorkflowManager,
	workspace, gatewayToken string,
) *Server {
	s := &Server{
		rpg:          rpg,
		tasks:        tasks,
		memory:       memory,
		workflows:    workflows,
		workspace:    workspace,
		defaultModel: skill.DefaultModel,
		gatewayToken: gatewayToken,
		addr:         addr,
	}

	s.mux = http.NewServeMux()
	s.mux.Handle("/api/state", s.requireLocalOnly(http.HandlerFunc(s.handleState)))
	s.mux.Handle("/api/skills", s.requireLocalOnly(http.HandlerFunc(s.handleSkills)))
	s.mux.Handle("/ws", s.requireLocalOnly(http.HandlerFunc(s.handleWS)))

	s.mux.Handle("/chat", s.requireLocalOnly(http.HandlerFunc(s.serveChat)))
	s.mux.Handle("/api/chat/status", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleGatewayStatus))))
	s.mux.Handle("/api/chat/send", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleChatSend))))
	s.mux.Handle("/api/memory/search", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleMemorySearch))))
	s.mux.Handle("/api/memory/recent", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleMemoryRecent))))
	s.mux.Handle("/api/memory/clear", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleMemoryClear))))
	s.mux.Handle("/api/browser/run", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleBrowserRun))))
	s.mux.Handle("/api/workflows", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleWorkflowListOrCreate))))
	s.mux.Handle("/api/workflows/cancel", s.requireLocalOnly(http.HandlerFunc(s.requireGatewayAuth(s.handleWorkflowCancel))))

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		slog.Error("⚠️ static FS 錯誤", "error", err)
	} else {
		s.static = http.FileServer(http.FS(sub))
		s.mux.Handle("/", s.requireLocalOnly(s.static))
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s
}

// Start begins listening in the background.
func (s *Server) Start() {
	go func() {
		baseURL := webBaseURL(s.addr)
		slog.Info("🎮 Axle Web 啟動", "dashboard", baseURL, "chat", baseURL+"/chat")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("⚠️ Web server 錯誤", "error", err)
		}
	}()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	_ = s.server.Shutdown(ctx)
}

func webBaseURL(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}

func (s *Server) serveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := staticFS.ReadFile("static/chat.html")
	if err != nil {
		http.Error(w, "chat UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// handleState returns the full RPG state as JSON.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if s.rpg == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}

	snap := s.rpg.Snapshot()
	tier := app.TierForLevel(snap.Level)

	resp := map[string]any{
		"level":         snap.Level,
		"total_xp":      snap.TotalXP,
		"next_level_xp": app.XPForLevel(snap.Level + 1),
		"tier":          tier,
		"skill_uses":    snap.SkillUses,
		"achievements":  snap.Achievements,
		"events":        snap.Events,
		"total_tasks":   snap.TotalTasks,
		"equipment":     snap.Equipment,
		"started_at":    snap.StartedAt,
		"uptime_secs":   int(time.Since(snap.StartedAt).Seconds()),
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleSkills returns all skill definitions.
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, app.SkillDefs)
}

// handleWS upgrades to WebSocket and streams RPG events.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.rpg == nil {
		http.Error(w, "rpg disabled", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("⚠️ WebSocket 升級失敗", "error", err)
		return
	}
	defer conn.Close()

	ch := s.rpg.Subscribe()
	defer s.rpg.Unsubscribe(ch)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	snap := s.rpg.Snapshot()
	_ = conn.WriteJSON(map[string]any{"type": "init", "state": snap})

	for {
		select {
		case <-readDone:
			return
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case evt := <-ch:
			snap := s.rpg.Snapshot()
			msg := map[string]any{
				"type":  "event",
				"event": evt,
				"state": map[string]any{
					"level":         snap.Level,
					"total_xp":      snap.TotalXP,
					"next_level_xp": app.XPForLevel(snap.Level + 1),
					"tier":          app.TierForLevel(snap.Level),
				},
			}
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) requireLocalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackRemote(r.RemoteAddr) {
			http.Error(w, "local access only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	return isLoopbackHost(host)
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
