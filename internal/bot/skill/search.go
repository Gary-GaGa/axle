package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxSearchResults = 30

// SearchResult represents a single grep match.
type SearchResult struct {
	File    string
	Line    int
	Content string
}

// SearchCode performs a recursive text search within the workspace sandbox.
// Returns matching lines with file paths and line numbers.
func SearchCode(ctx context.Context, workspace, pattern string) ([]SearchResult, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("搜尋關鍵字不能為空")
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("workspace 解析失敗: %w", err)
	}

	var results []SearchResult
	lowerPattern := strings.ToLower(pattern)

	err = filepath.WalkDir(absWorkspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		if len(results) >= maxSearchResults {
			return filepath.SkipAll
		}

		// Skip hidden dirs and common non-code dirs
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary/large files by extension
		if isBinaryExt(filepath.Ext(path)) {
			return nil
		}

		// Skip files > 1MB
		info, err := d.Info()
		if err != nil || info.Size() > 1024*1024 {
			return nil
		}

		relPath, _ := filepath.Rel(absWorkspace, path)

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for lineNum, line := range lines {
			if len(results) >= maxSearchResults {
				break
			}
			if strings.Contains(strings.ToLower(line), lowerPattern) {
				content := strings.TrimSpace(line)
				if len(content) > 120 {
					content = content[:120] + "..."
				}
				results = append(results, SearchResult{
					File:    relPath,
					Line:    lineNum + 1,
					Content: content,
				})
			}
		}
		return nil
	})

	if err != nil && err != context.Canceled {
		return results, fmt.Errorf("搜尋時發生錯誤: %w", err)
	}
	if err == context.Canceled {
		return nil, context.Canceled
	}
	return results, nil
}

// FormatSearchResults formats search results for Telegram display.
func FormatSearchResults(pattern string, results []SearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("🔍 未找到匹配：`%s`", pattern)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 搜尋結果：`%s`（%d 筆）\n\n", pattern, len(results)))

	currentFile := ""
	for _, r := range results {
		if r.File != currentFile {
			currentFile = r.File
			sb.WriteString(fmt.Sprintf("📄 *%s*\n", currentFile))
		}
		sb.WriteString(fmt.Sprintf("  L%d: `%s`\n", r.Line, r.Content))
	}

	if len(results) >= maxSearchResults {
		sb.WriteString(fmt.Sprintf("\n⚠️ 結果已截斷（上限 %d 筆）", maxSearchResults))
	}
	return sb.String()
}

var binaryExts = map[string]bool{
	".exe": true, ".bin": true, ".o": true, ".a": true, ".so": true,
	".dylib": true, ".dll": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".ico": true, ".pdf": true, ".zip": true, ".gz": true,
	".tar": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true, ".db": true,
	".sqlite": true, ".pyc": true, ".class": true, ".jar": true,
}

func isBinaryExt(ext string) bool {
	return binaryExts[strings.ToLower(ext)]
}
