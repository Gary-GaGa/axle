package skill

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// GenerateBriefing creates a daily briefing message with system, git, and calendar status.
func GenerateBriefing(ctx context.Context, workspace string) string {
	var sb strings.Builder
	now := time.Now()

	sb.WriteString(fmt.Sprintf("📢 *每日簡報* — %s\n\n", now.Format("2006/01/02 (Mon) 15:04")))

	// System info
	sb.WriteString("🖥 *系統狀態*\n")
	sb.WriteString(fmt.Sprintf("• Go: %s\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("• OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("• Goroutines: %d\n\n", runtime.NumGoroutine()))

	// Git status
	gitStatus, err := GitStatus(ctx, workspace)
	if err == nil && strings.TrimSpace(gitStatus) != "" {
		sb.WriteString("🔀 *Git 狀態*\n")
		lines := strings.Split(strings.TrimSpace(gitStatus), "\n")
		if len(lines) > 8 {
			lines = lines[:8]
			lines = append(lines, "...（更多省略）")
		}
		for _, line := range lines {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// Calendar
	calText, err := CalendarToday(ctx)
	if err == nil {
		sb.WriteString(calText + "\n\n")
	}

	// Disk usage
	dfOut, err := exec.CommandContext(ctx, "df", "-h", "/").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(dfOut)), "\n")
		if len(lines) > 1 {
			sb.WriteString("💾 *磁碟空間*\n```\n" + lines[len(lines)-1] + "\n```\n")
		}
	}

	return sb.String()
}
