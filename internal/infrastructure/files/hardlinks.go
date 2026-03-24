package files

import (
	"fmt"
	"os"
)

// EnsureSingleLink rejects files backed by multiple hard links.
func EnsureSingleLink(info os.FileInfo, originalRelPath string) error {
	if HasMultipleLinks(info) {
		return fmt.Errorf("⛔ 安全拒絕：路徑 `%s` 使用多重硬連結", originalRelPath)
	}
	return nil
}
