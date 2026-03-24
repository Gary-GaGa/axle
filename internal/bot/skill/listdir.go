package skill

import (
	"context"
	"fmt"
	"strings"

	"github.com/garyellow/axle/internal/usecase/dto"
)

const (
	maxTreeDepth = 5
	maxTreeItems = 200
)

// ListDir lists directory contents in a tree-like format.
func ListDir(ctx context.Context, workspace, relPath string, depth int) (string, error) {
	result, err := workspaceExplorer().ListDirectory(ctx, dto.ListDirectoryInput{
		Workspace: workspace,
		Path:      relPath,
		Depth:     depth,
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, line := range result.Lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)
	}
	if result.Truncated {
		sb.WriteString(fmt.Sprintf("\n\n⚠️ 結果可能不完整（上限 %d 項，或略過無法存取的路徑）", maxTreeItems))
	}
	return sb.String(), nil
}
