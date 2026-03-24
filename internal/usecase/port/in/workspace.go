package in

import (
	"context"

	"github.com/garyellow/axle/internal/usecase/dto"
)

// WorkspaceUsecase defines workspace exploration and write flows.
type WorkspaceUsecase interface {
	ReadCode(ctx context.Context, input dto.ReadCodeInput) (*dto.ReadCodeOutput, error)
	ListDirectory(ctx context.Context, input dto.ListDirectoryInput) (*dto.ListDirectoryOutput, error)
	SearchCode(ctx context.Context, input dto.SearchCodeInput) (*dto.SearchCodeOutput, error)
	FileExists(ctx context.Context, input dto.FileExistsInput) (*dto.FileExistsOutput, error)
	WriteFile(ctx context.Context, input dto.WriteFileInput) (*dto.WriteFileOutput, error)
}
