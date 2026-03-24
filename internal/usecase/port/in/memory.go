package in

import (
	"context"

	"github.com/garyellow/axle/internal/usecase/dto"
)

type MemoryUsecase interface {
	Load(ctx context.Context, input dto.LoadMemoryInput) error
	Add(ctx context.Context, input dto.AddMemoryInput) error
	AddDetailed(ctx context.Context, input dto.AddDetailedMemoryInput) error
	Recent(ctx context.Context, input dto.RecentMemoryInput) *dto.MemoryEntriesOutput
	Count(ctx context.Context, input dto.CountMemoryInput) *dto.MemoryCountOutput
	Search(ctx context.Context, input dto.SearchMemoryInput) *dto.MemorySearchOutput
	SearchRelevant(ctx context.Context, input dto.SearchMemoryInput) *dto.MemoryEntriesOutput
	BuildContext(ctx context.Context, input dto.BuildContextInput) *dto.MemoryTextOutput
	BuildRAGContext(ctx context.Context, input dto.BuildRAGContextInput) *dto.MemoryTextOutput
	Clear(ctx context.Context, input dto.ClearMemoryInput) error
}
