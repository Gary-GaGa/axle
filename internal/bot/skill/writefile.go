package skill

import (
	"context"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
	"github.com/garyellow/axle/internal/usecase/dto"
)

const maxWriteBytes = domainworkspace.MaxWriteBytes

// WriteResult summarizes a successful write through the compatibility wrapper.
type WriteResult struct {
	Path         string
	BytesWritten int
}

// FileExistsResult summarizes a workspace existence check through the compatibility wrapper.
type FileExistsResult struct {
	Path   string
	Exists bool
}

// WriteFile creates or overwrites a file within the workspace sandbox.
func WriteFile(workspace, relPath, content string) error {
	_, err := WriteFileDetailed(context.Background(), workspace, relPath, content)
	return err
}

// FileExists checks whether a file exists inside the workspace sandbox.
func FileExists(workspace, relPath string) (bool, error) {
	result, err := FileExistsDetailed(context.Background(), workspace, relPath)
	if err != nil {
		return false, err
	}
	return result.Exists, nil
}

// WriteFileDetailed writes a file and preserves the normalized target path.
func WriteFileDetailed(ctx context.Context, workspace, relPath, content string) (WriteResult, error) {
	result, err := workspaceExplorer().WriteFile(ctx, dto.WriteFileInput{
		Workspace: workspace,
		Path:      relPath,
		Content:   content,
	})
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{
		Path:         result.Path,
		BytesWritten: result.BytesWritten,
	}, nil
}

// FileExistsDetailed checks path existence and preserves the normalized target path.
func FileExistsDetailed(ctx context.Context, workspace, relPath string) (FileExistsResult, error) {
	result, err := workspaceExplorer().FileExists(ctx, dto.FileExistsInput{
		Workspace: workspace,
		Path:      relPath,
	})
	if err != nil {
		return FileExistsResult{}, err
	}
	return FileExistsResult{
		Path:   result.Path,
		Exists: result.Exists,
	}, nil
}

// resolveAndValidate resolves a relative path inside the workspace and ensures it does not escape.
func resolveAndValidate(workspace, relPath string) (string, error) {
	return domainworkspace.ResolvePath(workspace, relPath)
}
