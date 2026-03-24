package execution

import (
	"context"
	"sync"
	"time"
)

// TaskSlot enforces single-task execution with context-based cancellation.
type TaskSlot struct {
	mu        sync.Mutex
	running   bool
	name      string
	startedAt time.Time
	cancelFn  context.CancelFunc
}

// TryStart attempts to claim the task slot.
func (ts *TaskSlot) TryStart(name string) (context.Context, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return nil, false
	}
	ctx, cancel := context.WithCancel(context.Background())
	ts.running = true
	ts.name = name
	ts.startedAt = time.Now()
	ts.cancelFn = cancel
	return ctx, true
}

// Cancel sends a cancellation signal to the running task.
func (ts *TaskSlot) Cancel() bool {
	ts.mu.Lock()
	if !ts.running {
		ts.mu.Unlock()
		return false
	}
	cancel := ts.cancelFn
	ts.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return true
}

// Done releases the task slot.
func (ts *TaskSlot) Done() {
	ts.mu.Lock()
	cancel := ts.cancelFn
	ts.running = false
	ts.name = ""
	ts.cancelFn = nil
	ts.startedAt = time.Time{}
	ts.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Status returns a snapshot of current task state.
func (ts *TaskSlot) Status() (running bool, name string, elapsed time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return false, "", 0
	}
	return true, ts.name, time.Since(ts.startedAt)
}
