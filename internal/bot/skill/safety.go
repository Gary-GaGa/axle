package skill

import execution "github.com/garyellow/axle/internal/domain/execution"

// DangerLevel classifies how dangerous a shell command is.
type DangerLevel = execution.DangerLevel

const (
	DangerNone    = execution.DangerNone
	DangerWarning = execution.DangerWarning
	DangerBlocked = execution.DangerBlocked
)

// CheckCommandSafety analyses a shell command string and returns the highest danger level found.
func CheckCommandSafety(cmd string) (DangerLevel, []string) {
	report := execution.ClassifyCommand(cmd)
	return report.Level, report.Reasons
}
