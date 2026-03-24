package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
)

// OpenWorkspaceRoot opens a workspace as a root-scoped filesystem boundary.
func OpenWorkspaceRoot(workspace string) (*os.Root, string, error) {
	absWorkspace, err := domainworkspace.ResolveWorkspacePath(workspace)
	if err != nil {
		return nil, "", err
	}

	root, err := os.OpenRoot(absWorkspace)
	if err != nil {
		return nil, "", fmt.Errorf("workspace 無法開啟: %w", err)
	}
	return root, absWorkspace, nil
}

// EnsurePathHasNoSymlink rejects any existing symlink component in the relative path.
func EnsurePathHasNoSymlink(root *os.Root, relPath, originalRelPath string) error {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return err
	}
	if cleaned == "." {
		return nil
	}

	current := "."
	for _, part := range strings.Split(cleaned, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("無法存取: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("⛔ 安全拒絕：路徑 `%s` 包含符號連結", originalRelPath)
		}
	}
	return nil
}

// EnsureRootedDirectory creates a directory tree within the rooted workspace boundary.
func EnsureRootedDirectory(root *os.Root, relPath, originalRelPath string, perm os.FileMode) (string, error) {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return "", err
	}
	if err := EnsurePathHasNoSymlink(root, cleaned, originalRelPath); err != nil {
		return "", err
	}
	if err := root.MkdirAll(cleaned, perm); err != nil {
		return "", fmt.Errorf("建立目錄失敗: %w", err)
	}
	return cleaned, nil
}

// WriteRootedFile writes a file within the rooted workspace boundary.
func WriteRootedFile(root *os.Root, relPath, originalRelPath string, data []byte, perm os.FileMode) error {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return err
	}
	if err := EnsurePathHasNoSymlink(root, cleaned, originalRelPath); err != nil {
		return err
	}

	parentDir := filepath.Dir(cleaned)
	if parentDir != "." {
		if err := EnsurePathHasNoSymlink(root, parentDir, originalRelPath); err != nil {
			return err
		}
	}
	if err := root.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}

	info, err := root.Lstat(cleaned)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("`%s` 不是一般檔案", originalRelPath)
		}
		if err := EnsureSingleLink(info, originalRelPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("寫入失敗: %w", err)
	}

	file, err := root.OpenFile(cleaned, WriteOpenFlags, perm)
	if err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	defer file.Close()

	info, err = file.Stat()
	if err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("`%s` 不是一般檔案", originalRelPath)
	}
	if err := EnsureSingleLink(info, originalRelPath); err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("寫入失敗: %w", err)
	}
	return nil
}

// RootedFileExists checks for a file within the rooted workspace boundary.
func RootedFileExists(root *os.Root, relPath, originalRelPath string) (bool, error) {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return false, err
	}
	if err := EnsurePathHasNoSymlink(root, cleaned, originalRelPath); err != nil {
		return false, err
	}

	info, err := root.Stat(cleaned)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode().IsRegular() {
		if err := EnsureSingleLink(info, originalRelPath); err != nil {
			return false, err
		}
	}
	return true, nil
}
