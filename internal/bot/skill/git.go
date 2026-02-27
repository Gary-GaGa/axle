package skill

import (
	"context"
	"fmt"
	"strings"
)

// GitStatus runs `git status --short` in the workspace.
func GitStatus(ctx context.Context, workspace string) (string, error) {
	out, err := ExecShell(ctx, workspace, "git status --short")
	if err != nil {
		return "", fmt.Errorf("git status 失敗: %w", err)
	}
	if strings.TrimSpace(out) == "（指令執行完畢，無輸出）" || strings.TrimSpace(out) == "" {
		return "✅ 工作區乾淨，沒有變更", nil
	}
	return out, nil
}

// GitDiff runs `git diff` (unstaged) or `git diff --cached` (staged) in the workspace.
func GitDiff(ctx context.Context, workspace string, staged bool) (string, error) {
	cmd := "git diff"
	if staged {
		cmd = "git diff --cached"
	}
	out, err := ExecShell(ctx, workspace, cmd)
	if err != nil {
		return "", fmt.Errorf("git diff 失敗: %w", err)
	}
	if strings.TrimSpace(out) == "（指令執行完畢，無輸出）" || strings.TrimSpace(out) == "" {
		label := "未暫存"
		if staged {
			label = "已暫存"
		}
		return fmt.Sprintf("ℹ️ 沒有%s的變更", label), nil
	}
	return out, nil
}

// GitLog runs `git log --oneline -N` in the workspace.
func GitLog(ctx context.Context, workspace string, count int) (string, error) {
	if count <= 0 || count > 30 {
		count = 10
	}
	out, err := ExecShell(ctx, workspace, fmt.Sprintf("git log --oneline -%d", count))
	if err != nil {
		return "", fmt.Errorf("git log 失敗: %w", err)
	}
	return out, nil
}

// GitAddCommitPush stages all changes, commits with message, and pushes.
// This is a destructive operation and should only be called after user confirmation.
func GitAddCommitPush(ctx context.Context, workspace, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit 訊息不能為空")
	}

	// Sanitize commit message for shell safety
	safe := strings.ReplaceAll(message, "'", "'\\''")
	cmd := fmt.Sprintf("git add -A && git commit -m '%s' && git push", safe)

	out, err := ExecShell(ctx, workspace, cmd)
	if err != nil {
		return "", err
	}
	return out, nil
}
