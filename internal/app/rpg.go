package app

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── RPG Skill definitions ────────────────────────────────────────────────────

// RPGSkillDef maps an agent skill to its RPG representation.
type RPGSkillDef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Type   string `json:"type"`
	XP     int    `json:"xp"`
	Sprite string `json:"sprite"`
}

// SkillDefs is the complete skill → RPG mapping.
var SkillDefs = map[string]RPGSkillDef{
	"read_code":    {ID: "read_code", Name: "Code Scan", Icon: "👁", Type: "偵查", XP: 5, Sprite: "scan"},
	"write_file":   {ID: "write_file", Name: "Rune Carving", Icon: "✏️", Type: "創造", XP: 10, Sprite: "carve"},
	"exec_shell":   {ID: "exec_shell", Name: "Shell Strike", Icon: "🗡", Type: "攻擊", XP: 15, Sprite: "strike"},
	"copilot":      {ID: "copilot", Name: "AI Summon", Icon: "🔮", Type: "魔法", XP: 25, Sprite: "summon"},
	"copilot_stream": {ID: "copilot_stream", Name: "Thunder Stream", Icon: "⚡", Type: "持續魔法", XP: 30, Sprite: "thunder"},
	"web_search":   {ID: "web_search", Name: "Oracle Vision", Icon: "🔍", Type: "偵查", XP: 10, Sprite: "oracle"},
	"web_fetch":    {ID: "web_fetch", Name: "Portal Fetch", Icon: "🌐", Type: "傳送", XP: 10, Sprite: "portal"},
	"git_status":   {ID: "git_status", Name: "Compass Check", Icon: "🧭", Type: "導航", XP: 5, Sprite: "compass"},
	"git_diff":     {ID: "git_diff", Name: "Change Sight", Icon: "📊", Type: "偵查", XP: 5, Sprite: "sight"},
	"git_push":     {ID: "git_push", Name: "Warp Push", Icon: "🚀", Type: "傳送", XP: 20, Sprite: "warp"},
	"search_code":  {ID: "search_code", Name: "Deep Scan", Icon: "🔎", Type: "偵查", XP: 10, Sprite: "deepscan"},
	"list_dir":     {ID: "list_dir", Name: "Map Reveal", Icon: "🗺", Type: "探索", XP: 5, Sprite: "map"},
	"send_email":   {ID: "send_email", Name: "Messenger Hawk", Icon: "📨", Type: "通訊", XP: 15, Sprite: "hawk"},
	"read_email":   {ID: "read_email", Name: "Mail Check", Icon: "📬", Type: "通訊", XP: 5, Sprite: "mail"},
	"calendar":     {ID: "calendar", Name: "Time Sight", Icon: "📅", Type: "預知", XP: 5, Sprite: "hourglass"},
	"briefing":     {ID: "briefing", Name: "Morning Report", Icon: "📜", Type: "情報", XP: 20, Sprite: "scroll"},
	"sub_agent":    {ID: "sub_agent", Name: "Summon Companion", Icon: "👥", Type: "召喚", XP: 30, Sprite: "companion"},
	"plugin":       {ID: "plugin", Name: "Equip Artifact", Icon: "🧩", Type: "裝備", XP: 15, Sprite: "artifact"},
	"safety_block": {ID: "safety_block", Name: "Safety Ward", Icon: "🛡", Type: "防禦", XP: 0, Sprite: "shield"},
	"pdf":          {ID: "pdf", Name: "Scroll Decode", Icon: "📄", Type: "解讀", XP: 10, Sprite: "decode"},
	"image":        {ID: "image", Name: "Vision Lens", Icon: "📸", Type: "偵查", XP: 10, Sprite: "lens"},
	"github":       {ID: "github", Name: "Guild Board", Icon: "📋", Type: "情報", XP: 10, Sprite: "guild"},
	"self_upgrade":  {ID: "self_upgrade", Name: "Self Evolution", Icon: "🔧", Type: "進化", XP: 50, Sprite: "evolve"},
}

// ── Level system ─────────────────────────────────────────────────────────────

// LevelInfo describes a level tier.
type LevelInfo struct {
	Level    int    `json:"level"`
	Title    string `json:"title"`
	TitleEN  string `json:"title_en"`
	Icon     string `json:"icon"`
	MinXP    int    `json:"min_xp"`
}

// LevelTiers defines all level boundaries.
var LevelTiers = []LevelInfo{
	{Level: 1, Title: "見習編碼師", TitleEN: "Apprentice", Icon: "🟤", MinXP: 0},
	{Level: 6, Title: "程式碼遊俠", TitleEN: "Code Ranger", Icon: "🟢", MinXP: 101},
	{Level: 11, Title: "數據法師", TitleEN: "Data Mage", Icon: "🔵", MinXP: 501},
	{Level: 21, Title: "架構術士", TitleEN: "Arch Sorcerer", Icon: "🟣", MinXP: 2001},
	{Level: 36, Title: "傳奇工匠", TitleEN: "Legendary Crafter", Icon: "🟡", MinXP: 5001},
	{Level: 50, Title: "不朽引擎", TitleEN: "Immortal Engine", Icon: "🔴", MinXP: 10001},
}

// XPForLevel returns the cumulative XP required to reach the given level.
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	// XP curve: 20 * level^1.5
	return int(20 * math.Pow(float64(level), 1.5))
}

// LevelFromXP returns the current level for a given XP total.
func LevelFromXP(xp int) int {
	level := 1
	for {
		next := XPForLevel(level + 1)
		if xp < next {
			return level
		}
		level++
		if level > 999 {
			return level
		}
	}
}

// TierForLevel returns the LevelInfo for the given level.
func TierForLevel(level int) LevelInfo {
	result := LevelTiers[0]
	for _, t := range LevelTiers {
		if level >= t.Level {
			result = t
		}
	}
	return result
}

// ── RPG Event ────────────────────────────────────────────────────────────────

// RPGEvent is a single event in the activity feed.
type RPGEvent struct {
	Time      time.Time `json:"time"`
	SkillID   string    `json:"skill_id"`
	SkillName string    `json:"skill_name"`
	Icon      string    `json:"icon"`
	Detail    string    `json:"detail"`
	XP        int       `json:"xp"`
	Success   bool      `json:"success"`
}

// ── RPG State ────────────────────────────────────────────────────────────────

const (
	rpgFile         = "rpg_state.json"
	maxEventHistory = 100
)

// RPGState holds the persistent RPG progression data.
type RPGState struct {
	TotalXP       int            `json:"total_xp"`
	Level         int            `json:"level"`
	SkillUses     map[string]int `json:"skill_uses"`
	Achievements  []string       `json:"achievements"`
	Events        []RPGEvent     `json:"events"`
	TotalTasks    int            `json:"total_tasks"`
	StartedAt     time.Time      `json:"started_at"`
	Equipment     RPGEquipment   `json:"equipment"`
}

// RPGEquipment represents the agent's current "gear".
type RPGEquipment struct {
	Weapon     string `json:"weapon"`      // current AI model
	Armor      string `json:"armor"`       // auth mode
	Accessory  string `json:"accessory"`   // workspace
	Companions int    `json:"companions"`  // sub-agent count
}

// ── RPG Manager ──────────────────────────────────────────────────────────────

// RPGManager manages the RPG state and event broadcasting.
type RPGManager struct {
	mu        sync.RWMutex
	state     RPGState
	baseDir   string
	listeners []chan RPGEvent
	listenerMu sync.Mutex
}

// NewRPGManager creates a new RPG manager, loading state from disk.
func NewRPGManager(baseDir string) *RPGManager {
	m := &RPGManager{
		baseDir: baseDir,
		state: RPGState{
			Level:     1,
			SkillUses: make(map[string]int),
			StartedAt: time.Now(),
		},
	}
	m.load()
	return m
}

// EmitEvent records a skill usage, awards XP, and broadcasts to listeners.
func (m *RPGManager) EmitEvent(skillID, detail string, success bool) {
	def, ok := SkillDefs[skillID]
	if !ok {
		def = RPGSkillDef{ID: skillID, Name: skillID, Icon: "❓", XP: 5}
	}

	xp := def.XP
	if !success {
		xp = 0
	}

	evt := RPGEvent{
		Time:      time.Now(),
		SkillID:   skillID,
		SkillName: def.Name,
		Icon:      def.Icon,
		Detail:    detail,
		XP:        xp,
		Success:   success,
	}

	m.mu.Lock()
	m.state.TotalXP += xp
	m.state.TotalTasks++
	m.state.SkillUses[skillID]++
	oldLevel := m.state.Level
	m.state.Level = LevelFromXP(m.state.TotalXP)

	// Append event, trim to max
	m.state.Events = append(m.state.Events, evt)
	if len(m.state.Events) > maxEventHistory {
		m.state.Events = m.state.Events[len(m.state.Events)-maxEventHistory:]
	}

	// Check achievements
	m.checkAchievements()
	newLevel := m.state.Level
	m.mu.Unlock()

	if newLevel > oldLevel {
		tier := TierForLevel(newLevel)
		lvlEvt := RPGEvent{
			Time:      time.Now(),
			SkillID:   "level_up",
			SkillName: "Level Up!",
			Icon:      "🎉",
			Detail:    tier.Icon + " " + tier.Title + " (Lv." + itoa(newLevel) + ")",
			XP:        0,
			Success:   true,
		}
		m.broadcast(lvlEvt)
	}

	m.broadcast(evt)
	m.save()
}

// UpdateEquipment sets current equipment state.
func (m *RPGManager) UpdateEquipment(eq RPGEquipment) {
	m.mu.Lock()
	m.state.Equipment = eq
	m.mu.Unlock()
}

// Snapshot returns a copy of the current RPG state.
func (m *RPGManager) Snapshot() RPGState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.state
	// Deep copy maps/slices
	s.SkillUses = make(map[string]int, len(m.state.SkillUses))
	for k, v := range m.state.SkillUses {
		s.SkillUses[k] = v
	}
	s.Events = make([]RPGEvent, len(m.state.Events))
	copy(s.Events, m.state.Events)
	s.Achievements = make([]string, len(m.state.Achievements))
	copy(s.Achievements, m.state.Achievements)
	return s
}

// Subscribe returns a channel that receives RPG events.
func (m *RPGManager) Subscribe() chan RPGEvent {
	ch := make(chan RPGEvent, 32)
	m.listenerMu.Lock()
	m.listeners = append(m.listeners, ch)
	m.listenerMu.Unlock()
	return ch
}

// Unsubscribe removes a listener channel.
func (m *RPGManager) Unsubscribe(ch chan RPGEvent) {
	m.listenerMu.Lock()
	defer m.listenerMu.Unlock()
	for i, l := range m.listeners {
		if l == ch {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

func (m *RPGManager) broadcast(evt RPGEvent) {
	m.listenerMu.Lock()
	defer m.listenerMu.Unlock()
	for _, ch := range m.listeners {
		select {
		case ch <- evt:
		default: // drop if full
		}
	}
}

func (m *RPGManager) checkAchievements() {
	checks := map[string]bool{
		"first_blood":     m.state.TotalTasks >= 1,
		"10_tasks":        m.state.TotalTasks >= 10,
		"100_tasks":       m.state.TotalTasks >= 100,
		"500_tasks":       m.state.TotalTasks >= 500,
		"first_summon":    m.state.SkillUses["copilot"] >= 1,
		"10_summons":      m.state.SkillUses["copilot"] >= 10,
		"first_strike":    m.state.SkillUses["exec_shell"] >= 1,
		"code_scanner":    m.state.SkillUses["read_code"] >= 20,
		"git_master":      m.state.SkillUses["git_push"] >= 10,
		"multi_skilled":   len(m.state.SkillUses) >= 10,
		"level_10":        m.state.Level >= 10,
		"level_25":        m.state.Level >= 25,
		"level_50":        m.state.Level >= 50,
	}

	have := make(map[string]bool, len(m.state.Achievements))
	for _, a := range m.state.Achievements {
		have[a] = true
	}
	for id, met := range checks {
		if met && !have[id] {
			m.state.Achievements = append(m.state.Achievements, id)
		}
	}
}

// ── Persistence ──────────────────────────────────────────────────────────────

func (m *RPGManager) load() {
	path := filepath.Join(m.baseDir, rpgFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return // first run
	}
	var s RPGState
	if err := json.Unmarshal(data, &s); err != nil {
		slog.Warn("⚠️ RPG state 載入失敗", "error", err)
		return
	}
	if s.SkillUses == nil {
		s.SkillUses = make(map[string]int)
	}
	m.state = s
	slog.Info("🎮 RPG state 已載入", "level", s.Level, "xp", s.TotalXP)
}

func (m *RPGManager) save() {
	m.mu.RLock()
	data, err := json.MarshalIndent(m.state, "", "  ")
	m.mu.RUnlock()
	if err != nil {
		slog.Warn("⚠️ RPG state 序列化失敗", "error", err)
		return
	}
	path := filepath.Join(m.baseDir, rpgFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		slog.Warn("⚠️ RPG state 儲存失敗", "error", err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
