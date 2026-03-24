package execution

import (
	"testing"
	"time"
)

func TestTaskSlotTryStartAndDone(t *testing.T) {
	var slot TaskSlot

	ctx, ok := slot.TryStart("first")
	if !ok || ctx == nil {
		t.Fatalf("first TryStart should succeed")
	}
	if _, ok := slot.TryStart("second"); ok {
		t.Fatalf("second TryStart should fail while running")
	}

	slot.Done()
	if _, ok := slot.TryStart("third"); !ok {
		t.Fatalf("TryStart after Done should succeed")
	}
	slot.Done()
}

func TestTaskSlotCancel(t *testing.T) {
	var slot TaskSlot
	if slot.Cancel() {
		t.Fatalf("Cancel with no running task should return false")
	}

	ctx, ok := slot.TryStart("cancellable")
	if !ok {
		t.Fatalf("TryStart should succeed")
	}
	if !slot.Cancel() {
		t.Fatalf("Cancel should return true for running task")
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatalf("context should be cancelled")
	}
	slot.Done()
}

func TestTaskSlotStatus(t *testing.T) {
	var slot TaskSlot
	running, name, elapsed := slot.Status()
	if running || name != "" || elapsed != 0 {
		t.Fatalf("unexpected empty status: running=%v name=%q elapsed=%s", running, name, elapsed)
	}

	if _, ok := slot.TryStart("status-test"); !ok {
		t.Fatalf("TryStart should succeed")
	}
	time.Sleep(10 * time.Millisecond)

	running, name, elapsed = slot.Status()
	if !running {
		t.Fatalf("expected running status")
	}
	if name != "status-test" {
		t.Fatalf("name = %q, want status-test", name)
	}
	if elapsed <= 0 {
		t.Fatalf("elapsed should be > 0")
	}
	slot.Done()
}
