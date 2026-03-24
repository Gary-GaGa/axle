package app

import execution "github.com/garyellow/axle/internal/domain/execution"

// TaskManager preserves the legacy API while delegating to the execution domain.
type TaskManager = execution.TaskSlot
