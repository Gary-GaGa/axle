package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces a file atomically using a temp file in the same directory.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("建立暫存檔失敗: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("設定暫存檔權限失敗: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("寫入暫存檔失敗: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("關閉暫存檔失敗: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("替換檔案失敗: %w", err)
	}
	return nil
}
