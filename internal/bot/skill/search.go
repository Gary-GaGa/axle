package skill

import (
	"context"
	"fmt"
	"strings"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
	"github.com/garyellow/axle/internal/usecase/dto"
)

const maxSearchResults = 30

// SearchResult represents a single grep match.
type SearchResult = dto.SearchMatch

// SearchResults keeps compatibility data plus transport metadata.
type SearchResults struct {
	Matches   []SearchResult
	Truncated bool
}

// SearchCode performs a recursive text search within the workspace sandbox.
func SearchCode(ctx context.Context, workspace, pattern string) ([]SearchResult, error) {
	result, err := SearchCodeDetailed(ctx, workspace, pattern)
	if err != nil {
		return nil, err
	}
	return result.Matches, nil
}

// SearchCodeDetailed performs a recursive text search and preserves truncation metadata.
func SearchCodeDetailed(ctx context.Context, workspace, pattern string) (SearchResults, error) {
	result, err := workspaceExplorer().SearchCode(ctx, dto.SearchCodeInput{Workspace: workspace, Pattern: pattern})
	if err != nil {
		return SearchResults{}, err
	}
	return SearchResults{
		Matches:   result.Matches,
		Truncated: result.Truncated,
	}, nil
}

// FormatSearchResults formats search results for Telegram display.
func FormatSearchResults(pattern string, results []SearchResult, truncated ...bool) string {
	isTruncated := len(truncated) > 0 && truncated[0]
	if len(results) == 0 {
		msg := fmt.Sprintf("🔍 未找到匹配：`%s`", pattern)
		if isTruncated {
			msg += fmt.Sprintf("\n\n⚠️ 結果可能不完整（上限 %d 筆，或略過無法存取的路徑）", maxSearchResults)
		}
		return msg
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

	if isTruncated {
		sb.WriteString(fmt.Sprintf("\n⚠️ 結果可能不完整（上限 %d 筆，或略過無法存取的路徑）", maxSearchResults))
	}
	return sb.String()
}

func isBinaryExt(ext string) bool {
	return domainworkspace.IsBinaryExt(ext)
}
