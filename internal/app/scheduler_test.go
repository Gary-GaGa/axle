package app

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduleManager_AddAndList(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewScheduleManager(dir)
	if err != nil {
		t.Fatalf("NewScheduleManager: %v", err)
	}

	sched, err := sm.Add("test", "echo hello", 60)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if sched.ID == "" {
		t.Error("schedule ID should not be empty")
	}
	if sched.Name != "test" {
		t.Errorf("name = %q", sched.Name)
	}
	if !sched.Enabled {
		t.Error("should be enabled by default")
	}

	list := sm.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(list))
	}
}

func TestScheduleManager_Delete(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)
	sched, _ := sm.Add("deleteme", "echo", 60)

	if !sm.Delete(sched.ID) {
		t.Error("delete should succeed")
	}
	if len(sm.List()) != 0 {
		t.Error("list should be empty after delete")
	}

	if sm.Delete("nonexistent") {
		t.Error("delete nonexistent should fail")
	}
}

func TestScheduleManager_Toggle(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)
	sched, _ := sm.Add("toggleable", "echo", 60)

	enabled, ok := sm.Toggle(sched.ID)
	if !ok {
		t.Error("toggle should succeed")
	}
	if enabled {
		t.Error("should be disabled after toggle")
	}

	enabled, ok = sm.Toggle(sched.ID)
	if !ok {
		t.Error("toggle should succeed")
	}
	if !enabled {
		t.Error("should be enabled after second toggle")
	}

	_, ok = sm.Toggle("nonexistent")
	if ok {
		t.Error("toggle nonexistent should fail")
	}
}

func TestScheduleManager_Persistence(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)
	sm.Add("persistent", "echo hello", 30)
	sm.StopAll()

	// Verify file exists
	_, err := os.Stat(filepath.Join(dir, "schedules.json"))
	if err != nil {
		t.Error("schedules.json should exist")
	}

	// Load from same dir
	sm2, _ := NewScheduleManager(dir)
	list := sm2.List()
	if len(list) != 1 || list[0].Name != "persistent" {
		t.Error("schedule should persist")
	}
	sm2.StopAll()
}

func TestScheduleManager_Execution(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)

	var executed int32
	sm.SetExecFunc(func(id, cmd string) {
		atomic.AddInt32(&executed, 1)
	})

	// Use very short interval - but schedules use minutes, so we test startLocked directly
	// Instead, just verify SetExecFunc works by manually checking
	sm.Add("exec-test", "echo test", 1)
	sm.StopAll()

	// The exec func was set correctly
	if sm.execFn == nil {
		t.Error("execFn should be set")
	}
}

func TestScheduleManager_InvalidAdd(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)

	_, err := sm.Add("", "echo", 5)
	if err == nil {
		t.Error("expected error for empty name")
	}

	_, err = sm.Add("test", "", 5)
	if err == nil {
		t.Error("expected error for empty command")
	}

	_, err = sm.Add("test", "echo", 0)
	if err == nil {
		t.Error("expected error for zero interval")
	}
}

func TestScheduleManager_StopAll(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)
	sm.Add("s1", "echo 1", 60)
	sm.Add("s2", "echo 2", 60)

	// StopAll should not panic
	sm.StopAll()

	// Can restart
	sm.StartAll()
	sm.StopAll()
}

func TestScheduleManager_StartAllIdempotent(t *testing.T) {
	dir := t.TempDir()
	sm, _ := NewScheduleManager(dir)
	sm.Add("idem", "echo", 60)

	// Multiple StartAll calls should not create duplicate goroutines
	sm.StartAll()
	sm.StartAll()
	time.Sleep(10 * time.Millisecond)
	sm.StopAll()
}
