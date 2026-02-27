package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxWriteBytes = 1024 * 1024 // 1 MB safety limit

// WriteFile creates or overwrites a file within the workspace sandbox.
// It enforces the same "../" escape prevention as ReadCode.
// Parent directories are created automatically.
func WriteFile(workspace, relPath, content string) error {
	absTarget, err := resolveAndValidate(workspace, relPath)
	if err != nil {
		return err
	}

	if len(content) > maxWriteBytes {
		return fmt.Errorf("⚠️ 內容超過 %d KB 上限，已拒絕寫入", maxWriteBytes/1024)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}

	if err := os.WriteFile(absTarget, []byte(content), 0644); err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	return nil
}

// FileExists checks whether a file exists inside the workspace sandbox.
func FileExists(workspace, relPath string) (bool, error) {
	absTarget, err := resolveAndValidate(workspace, relPath)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(absTarget)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// resolveAndValidate resolves a relative path inside the workspace and ensures
// it does not escape. Shared by ReadCode and WriteFile.
func resolveAndValidate(workspace, relPath string) (string, error) {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace 路徑解析失敗: %w", err)
	}
	absTarget := filepath.Join(absWorkspace, filepath.Clean("/"+relPath))

	prefix := absWorkspace + string(os.PathSeparator)
	if absTarget != absWorkspace && !strings.HasPrefix(absTarget, prefix) {
		return "", fmt.Errorf("⛔ 安全拒絕：路徑 `%s` 逃逸 workspace", relPath)
	}
	return absTarget, nil
}
