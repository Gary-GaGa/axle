package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var binaryExts = map[string]bool{
	".exe": true, ".bin": true, ".o": true, ".a": true, ".so": true,
	".dylib": true, ".dll": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".ico": true, ".pdf": true, ".zip": true, ".gz": true,
	".tar": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true, ".db": true,
	".sqlite": true, ".pyc": true, ".class": true, ".jar": true,
}

// ResolveWorkspacePath resolves the workspace path to an absolute path.
func ResolveWorkspacePath(workspace string) (string, error) {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace 路徑解析失敗: %w", err)
	}
	return absWorkspace, nil
}

// NormalizeRelativePath cleans a user-provided relative path into a root-scoped path.
func NormalizeRelativePath(relPath string) string {
	cleaned := strings.TrimPrefix(filepath.Clean("/"+relPath), string(os.PathSeparator))
	if cleaned == "" {
		return "."
	}
	return cleaned
}

// ValidateRelativePath rejects absolute and parent-traversal paths for workspace operations.
func ValidateRelativePath(relPath string) (string, error) {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" {
		return ".", nil
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("⛔ 安全拒絕：路徑 `%s` 不可跳出 workspace", relPath)
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("⛔ 安全拒絕：路徑 `%s` 不可跳出 workspace", relPath)
	}
	return NormalizeRelativePath(trimmed), nil
}

// ResolvePath resolves a relative path inside the workspace and ensures it does not escape.
func ResolvePath(workspace, relPath string) (string, error) {
	absWorkspace, err := ResolveWorkspacePath(workspace)
	if err != nil {
		return "", err
	}

	realWorkspace, err := filepath.EvalSymlinks(absWorkspace)
	if err != nil {
		if os.IsNotExist(err) {
			realWorkspace = absWorkspace
		} else {
			return "", fmt.Errorf("workspace 路徑解析失敗: %w", err)
		}
	}

	cleaned, err := ValidateRelativePath(relPath)
	if err != nil {
		return "", err
	}
	target := filepath.Join(realWorkspace, cleaned)
	if err := ensurePathHasNoSymlink(realWorkspace, target, relPath); err != nil {
		return "", err
	}
	return target, nil
}

// ResolveListTarget resolves the list target and returns a display path for UI formatting.
func ResolveListTarget(workspace, relPath string) (string, string, error) {
	absWorkspace, err := ResolveWorkspacePath(workspace)
	if err != nil {
		return "", "", fmt.Errorf("workspace 解析失敗: %w", err)
	}

	target, err := ResolvePath(workspace, ".")
	if err != nil {
		return "", "", err
	}
	if relPath != "" && relPath != "." {
		resolved, err := ResolvePath(workspace, relPath)
		if err != nil {
			return "", "", err
		}
		target = resolved
	}

	displayPath := relPath
	if displayPath == "" || displayPath == "." {
		displayPath = filepath.Base(absWorkspace)
	}
	return target, displayPath, nil
}

// ClampTreeDepth normalizes tree depth into a safe range.
func ClampTreeDepth(depth int) int {
	if depth <= 0 || depth > MaxTreeDepth {
		return MaxTreeDepth
	}
	return depth
}

// ClampTreeItems normalizes the maximum number of tree items.
func ClampTreeItems(limit int) int {
	if limit <= 0 {
		return MaxTreeItems
	}
	return limit
}

// ClampReadBytes normalizes the file read limit.
func ClampReadBytes(limit int) int {
	if limit <= 0 {
		return MaxReadBytes
	}
	return limit
}

// ClampSearchResults normalizes the search result limit.
func ClampSearchResults(limit int) int {
	if limit <= 0 {
		return MaxSearchResults
	}
	return limit
}

// ValidateSearchPattern ensures the search pattern is usable.
func ValidateSearchPattern(pattern string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", fmt.Errorf("搜尋關鍵字不能為空")
	}
	return pattern, nil
}

// LanguageFromPath returns the code fence language inferred from the relative path.
func LanguageFromPath(relPath string) string {
	return strings.TrimPrefix(filepath.Ext(relPath), ".")
}

// IsBinaryExt reports whether a file extension should be treated as binary.
func IsBinaryExt(ext string) bool {
	return binaryExts[strings.ToLower(ext)]
}

// TrimSearchContent limits a displayed search match to a readable length.
func TrimSearchContent(line string) string {
	content := strings.TrimSpace(line)
	if len(content) > 120 {
		return content[:120] + "..."
	}
	return content
}

func ensurePathHasNoSymlink(root, target, originalRelPath string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return fmt.Errorf("路徑解析失敗: %w", err)
	}
	if rel == "." {
		return nil
	}

	current := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
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
