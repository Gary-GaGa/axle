package workspace

import (
	"context"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
	"github.com/garyellow/axle/internal/usecase/dto"
	portin "github.com/garyellow/axle/internal/usecase/port/in"
)

// Service orchestrates workspace exploration and write flows.
type Service struct {
	repo domainworkspace.Repository
}

var _ portin.WorkspaceUsecase = (*Service)(nil)

// NewService creates a workspace usecase service.
func NewService(repo domainworkspace.Repository) *Service {
	return &Service{repo: repo}
}

// ReadCode loads a code file snapshot from the workspace.
func (s *Service) ReadCode(ctx context.Context, input dto.ReadCodeInput) (*dto.ReadCodeOutput, error) {
	snippet, err := s.repo.ReadCode(ctx, domainworkspace.ReadCodeRequest{
		Workspace:    input.Workspace,
		RelativePath: input.Path,
		MaxBytes:     domainworkspace.ClampReadBytes(0),
	})
	if err != nil {
		return nil, err
	}
	return &dto.ReadCodeOutput{
		Path:      snippet.RelativePath,
		Language:  snippet.Language,
		Content:   snippet.Content,
		Truncated: snippet.Truncated,
	}, nil
}

// ListDirectory renders a tree-like directory view from the workspace.
func (s *Service) ListDirectory(ctx context.Context, input dto.ListDirectoryInput) (*dto.ListDirectoryOutput, error) {
	tree, err := s.repo.ListDirectory(ctx, domainworkspace.ListDirectoryRequest{
		Workspace:    input.Workspace,
		RelativePath: input.Path,
		Depth:        domainworkspace.ClampTreeDepth(input.Depth),
		ItemLimit:    domainworkspace.ClampTreeItems(0),
	})
	if err != nil {
		return nil, err
	}
	return &dto.ListDirectoryOutput{
		DisplayPath: tree.DisplayPath,
		Lines:       append([]string(nil), tree.Lines...),
		Truncated:   tree.Truncated,
	}, nil
}

// SearchCode performs a lexical search inside the workspace.
func (s *Service) SearchCode(ctx context.Context, input dto.SearchCodeInput) (*dto.SearchCodeOutput, error) {
	pattern, err := domainworkspace.ValidateSearchPattern(input.Pattern)
	if err != nil {
		return nil, err
	}
	searchResult, err := s.repo.SearchCode(ctx, domainworkspace.SearchCodeRequest{
		Workspace:  input.Workspace,
		Pattern:    pattern,
		MaxResults: domainworkspace.ClampSearchResults(0),
	})
	if err != nil {
		return nil, err
	}
	result := &dto.SearchCodeOutput{Pattern: pattern, Truncated: searchResult.Truncated}
	for _, match := range searchResult.Matches {
		result.Matches = append(result.Matches, dto.SearchMatch{
			File:    match.File,
			Line:    match.Line,
			Content: match.Content,
		})
	}
	return result, nil
}

// FileExists checks whether a workspace path currently exists.
func (s *Service) FileExists(ctx context.Context, input dto.FileExistsInput) (*dto.FileExistsOutput, error) {
	if err := domainworkspace.ValidateWritePath(input.Path); err != nil {
		return nil, err
	}
	cleaned, err := domainworkspace.ValidateRelativePath(input.Path)
	if err != nil {
		return nil, err
	}
	exists, err := s.repo.FileExists(ctx, domainworkspace.FileExistsRequest{
		Workspace:    input.Workspace,
		RelativePath: input.Path,
	})
	if err != nil {
		return nil, err
	}
	return &dto.FileExistsOutput{
		Path:   cleaned,
		Exists: exists,
	}, nil
}

// WriteFile writes or overwrites a file within the workspace sandbox.
func (s *Service) WriteFile(ctx context.Context, input dto.WriteFileInput) (*dto.WriteFileOutput, error) {
	if err := domainworkspace.ValidateWritePath(input.Path); err != nil {
		return nil, err
	}
	if err := domainworkspace.ValidateWriteContent(input.Content); err != nil {
		return nil, err
	}
	result, err := s.repo.WriteFile(ctx, domainworkspace.WriteFileRequest{
		Workspace:    input.Workspace,
		RelativePath: input.Path,
		Content:      input.Content,
	})
	if err != nil {
		return nil, err
	}
	return &dto.WriteFileOutput{
		Path:         result.RelativePath,
		BytesWritten: result.BytesWritten,
	}, nil
}
