package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Schedule represents a periodic task.
type Schedule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Interval int    `json:"interval_minutes"` // interval in minutes
	Enabled  bool   `json:"enabled"`
}

// ScheduleManager manages periodic scheduled tasks.
type ScheduleManager struct {
	mu        sync.RWMutex
	schedules []Schedule
	stopChs   map[string]chan struct{}
	filePath  string
	execFn    func(schedID, command string) // callback to execute command
}

// NewScheduleManager creates a ScheduleManager with persistence.
func NewScheduleManager(axleDir string) (*ScheduleManager, error) {
	filePath := filepath.Join(axleDir, "schedules.json")
	sm := &ScheduleManager{
		filePath: filePath,
		stopChs:  make(map[string]chan struct{}),
	}

	if err := sm.load(); err != nil {
		slog.Warn("載入排程時有警告", "error", err)
	}
	return sm, nil
}

// SetExecFunc sets the callback used to execute scheduled commands.
func (sm *ScheduleManager) SetExecFunc(fn func(schedID, command string)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.execFn = fn
}

// Add creates a new schedule and starts it if enabled.
func (sm *ScheduleManager) Add(name, command string, intervalMin int) (Schedule, error) {
	if name == "" || command == "" || intervalMin <= 0 {
		return Schedule{}, fmt.Errorf("名稱、指令和間隔（分鐘）都不能為空/零")
	}
	if intervalMin < 1 {
		return Schedule{}, fmt.Errorf("間隔最少 1 分鐘")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := fmt.Sprintf("sched-%d", time.Now().UnixNano())
	sched := Schedule{
		ID:       id,
		Name:     name,
		Command:  command,
		Interval: intervalMin,
		Enabled:  true,
	}
	sm.schedules = append(sm.schedules, sched)

	if err := sm.persistLocked(); err != nil {
		return sched, err
	}

	sm.startLocked(sched)
	slog.Info("⏰ 排程已建立", "id", id, "name", name, "interval", intervalMin)
	return sched, nil
}

// List returns all schedules.
func (sm *ScheduleManager) List() []Schedule {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make([]Schedule, len(sm.schedules))
	copy(result, sm.schedules)
	return result
}

// Delete removes a schedule by ID and stops it.
func (sm *ScheduleManager) Delete(id string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Stop if running
	if ch, ok := sm.stopChs[id]; ok {
		close(ch)
		delete(sm.stopChs, id)
	}

	for i, s := range sm.schedules {
		if s.ID == id {
			sm.schedules = append(sm.schedules[:i], sm.schedules[i+1:]...)
			_ = sm.persistLocked()
			slog.Info("⏰ 排程已刪除", "id", id)
			return true
		}
	}
	return false
}

// Toggle enables/disables a schedule.
func (sm *ScheduleManager) Toggle(id string) (bool, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, s := range sm.schedules {
		if s.ID == id {
			sm.schedules[i].Enabled = !s.Enabled
			_ = sm.persistLocked()

			if sm.schedules[i].Enabled {
				sm.startLocked(sm.schedules[i])
			} else {
				if ch, ok := sm.stopChs[id]; ok {
					close(ch)
					delete(sm.stopChs, id)
				}
			}
			return sm.schedules[i].Enabled, true
		}
	}
	return false, false
}

// StartAll starts all enabled schedules.
func (sm *ScheduleManager) StartAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, s := range sm.schedules {
		if s.Enabled {
			sm.startLocked(s)
		}
	}
}

// StopAll stops all running schedules.
func (sm *ScheduleManager) StopAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for id, ch := range sm.stopChs {
		close(ch)
		delete(sm.stopChs, id)
	}
}

func (sm *ScheduleManager) startLocked(s Schedule) {
	// Don't start if already running
	if _, running := sm.stopChs[s.ID]; running {
		return
	}

	stopCh := make(chan struct{})
	sm.stopChs[s.ID] = stopCh

	go func(sched Schedule) {
		ticker := time.NewTicker(time.Duration(sched.Interval) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				sm.mu.RLock()
				fn := sm.execFn
				sm.mu.RUnlock()
				if fn != nil {
					fn(sched.ID, sched.Command)
				}
			}
		}
	}(s)
}

func (sm *ScheduleManager) load() error {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &sm.schedules)
}

func (sm *ScheduleManager) persistLocked() error {
	data, err := json.MarshalIndent(sm.schedules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.filePath, data, 0600)
}
