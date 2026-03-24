package workspace

import (
	"context"
	"errors"
	"testing"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
	"github.com/garyellow/axle/internal/usecase/dto"
)

type stubRepository struct {
	readResult   domainworkspace.CodeSnippet
	readErr      error
	listResult   domainworkspace.DirectoryTree
	listErr      error
	searchResult domainworkspace.SearchCodeResult
	searchErr    error
	existsResult bool
	existsErr    error
	writeResult  domainworkspace.WriteFileResult
	writeErr     error
}

func (s stubRepository) ReadCode(_ context.Context, _ domainworkspace.ReadCodeRequest) (domainworkspace.CodeSnippet, error) {
	return s.readResult, s.readErr
}

func (s stubRepository) ListDirectory(_ context.Context, _ domainworkspace.ListDirectoryRequest) (domainworkspace.DirectoryTree, error) {
	return s.listResult, s.listErr
}

func (s stubRepository) SearchCode(_ context.Context, _ domainworkspace.SearchCodeRequest) (domainworkspace.SearchCodeResult, error) {
	return s.searchResult, s.searchErr
}

func (s stubRepository) FileExists(_ context.Context, _ domainworkspace.FileExistsRequest) (bool, error) {
	return s.existsResult, s.existsErr
}

func (s stubRepository) WriteFile(_ context.Context, _ domainworkspace.WriteFileRequest) (domainworkspace.WriteFileResult, error) {
	return s.writeResult, s.writeErr
}

func TestServiceReadCode(t *testing.T) {
	svc := NewService(stubRepository{readResult: domainworkspace.CodeSnippet{RelativePath: "main.go", Language: "go", Content: "package main", Truncated: true}})
	out, err := svc.ReadCode(context.Background(), dto.ReadCodeInput{Workspace: "/tmp/project", Path: "main.go"})
	if err != nil {
		t.Fatalf("ReadCode returned error: %v", err)
	}
	if out.Language != "go" || out.Content != "package main" || !out.Truncated {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestServiceListDirectory(t *testing.T) {
	svc := NewService(stubRepository{listResult: domainworkspace.DirectoryTree{DisplayPath: "axle", Lines: []string{"📂 axle/", "└── main.go"}, Truncated: false}})
	out, err := svc.ListDirectory(context.Background(), dto.ListDirectoryInput{Workspace: "/tmp/project", Path: ".", Depth: 3})
	if err != nil {
		t.Fatalf("ListDirectory returned error: %v", err)
	}
	if len(out.Lines) != 2 || out.DisplayPath != "axle" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestServiceSearchCode(t *testing.T) {
	svc := NewService(stubRepository{searchResult: domainworkspace.SearchCodeResult{Matches: []domainworkspace.SearchMatch{{File: "main.go", Line: 3, Content: "func main()"}}, Truncated: true}})
	out, err := svc.SearchCode(context.Background(), dto.SearchCodeInput{Workspace: "/tmp/project", Pattern: "func"})
	if err != nil {
		t.Fatalf("SearchCode returned error: %v", err)
	}
	if len(out.Matches) != 1 || out.Matches[0].File != "main.go" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if !out.Truncated {
		t.Fatalf("expected truncated metadata to be preserved")
	}
}

func TestServiceSearchCodeRejectsEmptyPattern(t *testing.T) {
	svc := NewService(stubRepository{})
	if _, err := svc.SearchCode(context.Background(), dto.SearchCodeInput{Workspace: "/tmp/project", Pattern: "   "}); err == nil {
		t.Fatalf("expected validation error for empty pattern")
	}
}

func TestServicePropagatesRepositoryError(t *testing.T) {
	svc := NewService(stubRepository{readErr: errors.New("boom")})
	if _, err := svc.ReadCode(context.Background(), dto.ReadCodeInput{Workspace: "/tmp/project", Path: "main.go"}); err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestServiceFileExists(t *testing.T) {
	svc := NewService(stubRepository{existsResult: true})
	out, err := svc.FileExists(context.Background(), dto.FileExistsInput{Workspace: "/tmp/project", Path: "main.go"})
	if err != nil {
		t.Fatalf("FileExists returned error: %v", err)
	}
	if !out.Exists || out.Path != "main.go" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestServiceFileExistsRejectsTraversalPath(t *testing.T) {
	svc := NewService(stubRepository{existsResult: true})
	if _, err := svc.FileExists(context.Background(), dto.FileExistsInput{Workspace: "/tmp/project", Path: "../main.go"}); err == nil {
		t.Fatalf("expected traversal path to be rejected")
	}
}

func TestServiceWriteFile(t *testing.T) {
	svc := NewService(stubRepository{writeResult: domainworkspace.WriteFileResult{RelativePath: "notes/todo.txt", BytesWritten: 5}})
	out, err := svc.WriteFile(context.Background(), dto.WriteFileInput{
		Workspace: "/tmp/project",
		Path:      "notes/todo.txt",
		Content:   "hello",
	})
	if err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if out.Path != "notes/todo.txt" || out.BytesWritten != 5 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestServiceWriteFileRejectsInvalidInput(t *testing.T) {
	svc := NewService(stubRepository{})
	if _, err := svc.WriteFile(context.Background(), dto.WriteFileInput{Workspace: "/tmp/project", Path: "   ", Content: "hello"}); err == nil {
		t.Fatalf("expected blank path validation error")
	}
	if _, err := svc.WriteFile(context.Background(), dto.WriteFileInput{Workspace: "/tmp/project", Path: "notes/todo.txt", Content: string(make([]byte, domainworkspace.MaxWriteBytes+1))}); err == nil {
		t.Fatalf("expected oversized content validation error")
	}
}
