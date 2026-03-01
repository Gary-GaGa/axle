package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/garyellow/axle/internal/app"
	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // local only
}

// Server is the RPG dashboard HTTP + WebSocket server.
type Server struct {
	rpg    *app.RPGManager
	addr   string
	mux    *http.ServeMux
	server *http.Server
}

// NewServer creates a new web dashboard server.
func NewServer(addr string, rpg *app.RPGManager) *Server {
	s := &Server{rpg: rpg, addr: addr}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/skills", s.handleSkills)
	s.mux.HandleFunc("/ws", s.handleWS)

	// Serve embedded static files
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		slog.Error("⚠️ static FS 錯誤", "error", err)
	} else {
		s.mux.Handle("/", http.FileServer(http.FS(sub)))
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

// Start begins listening in the background.
func (s *Server) Start() {
	go func() {
		slog.Info("🎮 RPG Dashboard 啟動", "addr", "http://"+s.addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("⚠️ Web server 錯誤", "error", err)
		}
	}()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	s.server.Close()
}

// handleState returns the full RPG state as JSON.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	snap := s.rpg.Snapshot()
	tier := app.TierForLevel(snap.Level)

	resp := map[string]any{
		"level":        snap.Level,
		"total_xp":     snap.TotalXP,
		"next_level_xp": app.XPForLevel(snap.Level + 1),
		"tier":         tier,
		"skill_uses":   snap.SkillUses,
		"achievements": snap.Achievements,
		"events":       snap.Events,
		"total_tasks":  snap.TotalTasks,
		"equipment":    snap.Equipment,
		"started_at":   snap.StartedAt,
		"uptime_secs":  int(time.Since(snap.StartedAt).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleSkills returns all skill definitions.
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app.SkillDefs)
}

// handleWS upgrades to WebSocket and streams RPG events.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("⚠️ WebSocket 升級失敗", "error", err)
		return
	}
	defer conn.Close()

	ch := s.rpg.Subscribe()
	defer s.rpg.Unsubscribe(ch)

	// Send initial state
	snap := s.rpg.Snapshot()
	conn.WriteJSON(map[string]any{"type": "init", "state": snap})

	// Stream events
	for evt := range ch {
		snap := s.rpg.Snapshot()
		msg := map[string]any{
			"type":  "event",
			"event": evt,
			"state": map[string]any{
				"level":    snap.Level,
				"total_xp": snap.TotalXP,
				"next_level_xp": app.XPForLevel(snap.Level + 1),
				"tier":     app.TierForLevel(snap.Level),
			},
		}
		if err := conn.WriteJSON(msg); err != nil {
			break
		}
	}
}
