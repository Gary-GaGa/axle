package skill

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const execTimeout = 60 * time.Second
const maxOutputBytes = 1024 * 1024 // 1 MB cap to prevent OOM

// limitedWriter is a bytes.Buffer wrapper that stops accepting data after a limit.
type limitedWriter struct {
	buf     bytes.Buffer
	limit   int
	dropped bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	remaining := lw.limit - lw.buf.Len()
	if remaining <= 0 {
		lw.dropped = true
		return len(p), nil // silently discard
	}
	if len(p) > remaining {
		lw.dropped = true
		p = p[:remaining]
	}
	return lw.buf.Write(p)
}

// ExecShell runs a shell command inside the workspace directory.
// The provided ctx is used for both cancellation and the timeout.
// Output is capped at 1 MB to prevent memory exhaustion.
func ExecShell(ctx context.Context, workspace, command string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workspace

	stdout := &limitedWriter{limit: maxOutputBytes}
	stderr := &limitedWriter{limit: maxOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()

	// Check cancellation first (more informative than generic error)
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return "", fmt.Errorf("⏱ 執行超時（限制 %s）", execTimeout)
	case context.Canceled:
		return "", context.Canceled
	}

	out := strings.TrimSpace(stdout.buf.String())
	errOut := strings.TrimSpace(stderr.buf.String())

	if stdout.dropped || stderr.dropped {
		out += "\n\n⚠️ 輸出過大，已截斷至 1 MB"
	}

	if errOut != "" {
		if out != "" {
			out += "\n\n⚠️ Stderr:\n" + errOut
		} else {
			out = "⚠️ Stderr:\n" + errOut
		}
	}
	if runErr != nil && out == "" {
		return "", fmt.Errorf("執行失敗: %w", runErr)
	}
	if out == "" {
		out = "（指令執行完畢，無輸出）"
	}
	return out, nil
}
