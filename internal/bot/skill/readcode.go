package skill

import (
	"context"
	"fmt"

	localfs "github.com/garyellow/axle/internal/interface/out/persistence/localfs"
	"github.com/garyellow/axle/internal/usecase/dto"
	portin "github.com/garyellow/axle/internal/usecase/port/in"
	workspaceusecase "github.com/garyellow/axle/internal/usecase/workspace"
)

const maxReadBytes = 30 * 1024 // 30 KB Telegram display limit

func workspaceExplorer() portin.WorkspaceUsecase {
	return workspaceusecase.NewService(localfs.NewWorkspaceRepository())
}

// ReadCode reads a file within the workspace sandbox and returns formatted content.
func ReadCode(ctx context.Context, workspace, relPath string) (string, error) {
	result, err := workspaceExplorer().ReadCode(ctx, dto.ReadCodeInput{Workspace: workspace, Path: relPath})
	if err != nil {
		return "", err
	}

	truncated := ""
	if result.Truncated {
		truncated = fmt.Sprintf("\n\n⚠️ 檔案過大，僅顯示前 %d KB", maxReadBytes/1024)
	}
	return fmt.Sprintf("```%s\n%s\n```%s", result.Language, result.Content, truncated), nil
}
