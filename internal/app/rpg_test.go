package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestXPForLevel(t *testing.T) {
	if XPForLevel(1) != 0 {
		t.Errorf("level 1 should be 0 XP, got %d", XPForLevel(1))
	}
	prev := 0
	for l := 2; l <= 50; l++ {
		xp := XPForLevel(l)
		if xp <= prev {
			t.Errorf("XP should increase: level %d = %d, prev = %d", l, xp, prev)
		}
		prev = xp
	}
}

func TestLevelFromXP(t *testing.T) {
	tests := []struct {
		xp    int
		level int
	}{
		{0, 1},
		{10, 1},
		{100, 2},
		{1000, 13},
		{5000, 39},
	}
	for _, tt := range tests {
		got := LevelFromXP(tt.xp)
		if got != tt.level {
			t.Errorf("LevelFromXP(%d) = %d, want %d", tt.xp, got, tt.level)
		}
	}
}

func TestTierForLevel(t *testing.T) {
	tier := TierForLevel(1)
	if tier.TitleEN != "Apprentice" {
		t.Errorf("level 1 tier = %q", tier.TitleEN)
	}
	tier = TierForLevel(50)
	if tier.TitleEN != "Immortal Engine" {
		t.Errorf("level 50 tier = %q", tier.TitleEN)
	}
}

func TestRPGManager_EmitAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	m.EmitEvent("exec_shell", "echo hello", true)
	m.EmitEvent("copilot", "explain code", true)
	m.EmitEvent("read_code", "main.go", false) // failed, no XP

	snap := m.Snapshot()
	if snap.TotalXP != 40 { // 15 + 25 + 0
		t.Errorf("TotalXP = %d, want 40", snap.TotalXP)
	}
	if snap.TotalTasks != 3 {
		t.Errorf("TotalTasks = %d, want 3", snap.TotalTasks)
	}
	if snap.SkillUses["exec_shell"] != 1 {
		t.Error("exec_shell uses should be 1")
	}
	if len(snap.Events) != 3 {
		t.Errorf("Events = %d, want 3", len(snap.Events))
	}
}

func TestRPGManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	m1 := NewRPGManager(dir)
	m1.EmitEvent("copilot", "test", true)

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, rpgFile)); err != nil {
		t.Fatal("rpg_state.json not saved")
	}

	// Load in new manager
	m2 := NewRPGManager(dir)
	snap := m2.Snapshot()
	if snap.TotalXP != 25 {
		t.Errorf("loaded XP = %d, want 25", snap.TotalXP)
	}
}

func TestRPGManager_Subscribe(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	ch := m.Subscribe()
	defer m.Unsubscribe(ch)

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.EmitEvent("exec_shell", "ls", true)
	}()

	select {
	case evt := <-ch:
		if evt.SkillID != "exec_shell" {
			t.Errorf("event skill = %q", evt.SkillID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
	<-done
}

func TestRPGManager_Achievements(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	m.EmitEvent("exec_shell", "test", true)
	snap := m.Snapshot()

	found := false
	for _, a := range snap.Achievements {
		if a == "first_blood" {
			found = true
		}
		if a == "first_strike" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected first_blood or first_strike achievement, got %v", snap.Achievements)
	}
}

func TestRPGManager_LevelUp(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	ch := m.Subscribe()
	defer m.Unsubscribe(ch)

	// Emit enough events to trigger level up (need XP > XPForLevel(2) = ~56)
	for i := 0; i < 5; i++ {
		m.EmitEvent("copilot_stream", "task", true) // 30 XP each = 150 total
	}

	snap := m.Snapshot()
	if snap.Level < 2 {
		t.Errorf("expected level >= 2 after 150 XP, got level %d", snap.Level)
	}
}

func TestRPGManager_EventTrimming(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	for i := 0; i < 150; i++ {
		m.EmitEvent("read_code", "file", true)
	}

	snap := m.Snapshot()
	if len(snap.Events) > maxEventHistory {
		t.Errorf("events = %d, max = %d", len(snap.Events), maxEventHistory)
	}
}

func TestRPGManager_UpdateEquipment(t *testing.T) {
	dir := t.TempDir()
	m := NewRPGManager(dir)

	m.UpdateEquipment(RPGEquipment{
		Weapon:     "Claude Sonnet",
		Armor:      "Whitelist",
		Accessory:  "/project",
		Companions: 2,
	})

	snap := m.Snapshot()
	if snap.Equipment.Weapon != "Claude Sonnet" {
		t.Errorf("weapon = %q", snap.Equipment.Weapon)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"}, {1, "1"}, {42, "42"}, {100, "100"},
	}
	for _, tt := range tests {
		if got := itoa(tt.n); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
