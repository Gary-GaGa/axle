package app

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// TaskManager enforces single-task execution with context-based cancellation.
// All methods are goroutine-safe.
type TaskManager struct {
	mu        sync.Mutex
	running   bool
	name      string
	startedAt time.Time
	cancelFn  context.CancelFunc
}

// TryStart attempts to claim the task slot.
// Returns (ctx, true) on success; (nil, false) if a task is already running.
func (tm *TaskManager) TryStart(name string) (context.Context, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.running {
		return nil, false
	}
	ctx, cancel := context.WithCancel(context.Background())
	tm.running = true
	tm.name = name
	tm.startedAt = time.Now()
	tm.cancelFn = cancel
	slog.Info("🟢 任務開始", "task", name)
	return ctx, true
}

// Cancel sends a cancellation signal to the running task.
// Returns true if a task was running, false otherwise.
func (tm *TaskManager) Cancel() bool {
	tm.mu.Lock()
	if !tm.running {
		tm.mu.Unlock()
		return false
	}
	cancel := tm.cancelFn // capture before releasing lock
	name := tm.name
	tm.mu.Unlock() // release lock before calling cancel

	slog.Info("🛑 取消任務", "task", name)
	cancel()
	return true
}

// Done releases the task slot. Must be called (via defer) by the task goroutine.
func (tm *TaskManager) Done() {
	tm.mu.Lock()
	cancel := tm.cancelFn // capture before releasing lock
	tm.running = false
	tm.name = ""
	tm.cancelFn = nil
	tm.mu.Unlock() // release lock before calling cancel (avoids calling CancelFunc under mutex)

	if cancel != nil {
		cancel()
	}
	slog.Info("🏁 任務結束")
}

// Status returns a snapshot of current task state.
func (tm *TaskManager) Status() (running bool, name string, elapsed time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running {
		return false, "", 0
	}
	return true, tm.name, time.Since(tm.startedAt)
}
