package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxReadBytes = 30 * 1024 // 30 KB Telegram display limit

// ReadCode reads a file within the workspace sandbox and returns formatted content.
// It rejects any path that attempts to escape the workspace via "../" traversal.
func ReadCode(ctx context.Context, workspace, relPath string) (string, error) {
	absTarget, err := resolveAndValidate(workspace, relPath)
	if err != nil {
		return "", err
	}

	select {
	case <-ctx.Done():
		return "", context.Canceled
	default:
	}

	data, err := os.ReadFile(absTarget)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("檔案不存在: `%s`", relPath)
		}
		return "", fmt.Errorf("讀取失敗: %w", err)
	}

	truncated := ""
	if len(data) > maxReadBytes {
		data = data[:maxReadBytes]
		truncated = fmt.Sprintf("\n\n⚠️ 檔案過大，僅顯示前 %d KB", maxReadBytes/1024)
	}

	ext := strings.TrimPrefix(filepath.Ext(relPath), ".")
	return fmt.Sprintf("```%s\n%s\n```%s", ext, string(data), truncated), nil
}
