package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxTreeDepth = 5
	maxTreeItems = 200
)

// ListDir lists directory contents in a tree-like format.
// relPath is relative to workspace; depth controls recursion (0 = current only).
func ListDir(ctx context.Context, workspace, relPath string, depth int) (string, error) {
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace 解析失敗: %w", err)
	}

	target := absWorkspace
	if relPath != "" && relPath != "." {
		resolved, err := resolveAndValidate(workspace, relPath)
		if err != nil {
			return "", err
		}
		target = resolved
	}

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("路徑不存在: `%s`", relPath)
		}
		return "", fmt.Errorf("無法存取: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("`%s` 不是目錄", relPath)
	}

	if depth <= 0 || depth > maxTreeDepth {
		depth = maxTreeDepth
	}

	var sb strings.Builder
	displayPath := relPath
	if displayPath == "" || displayPath == "." {
		displayPath = filepath.Base(absWorkspace)
	}
	sb.WriteString(fmt.Sprintf("📂 %s/\n", displayPath))

	count := 0
	buildTree(ctx, target, "", depth, &sb, &count)

	if count >= maxTreeItems {
		sb.WriteString(fmt.Sprintf("\n⚠️ 結果已截斷（上限 %d 項）", maxTreeItems))
	}

	return sb.String(), nil
}

func buildTree(ctx context.Context, dir, prefix string, depth int, sb *strings.Builder, count *int) {
	if depth <= 0 || *count >= maxTreeItems {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Filter hidden files and sort: directories first
	var visible []os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			visible = append(visible, e)
		}
	}
	sort.Slice(visible, func(i, j int) bool {
		di, dj := visible[i].IsDir(), visible[j].IsDir()
		if di != dj {
			return di // dirs first
		}
		return visible[i].Name() < visible[j].Name()
	})

	for i, entry := range visible {
		if *count >= maxTreeItems {
			return
		}
		*count++

		isLast := i == len(visible)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("%s%s📁 %s/\n", prefix, connector, entry.Name()))
			buildTree(ctx, filepath.Join(dir, entry.Name()), childPrefix, depth-1, sb, count)
		} else {
			sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, entry.Name()))
		}
	}
}
