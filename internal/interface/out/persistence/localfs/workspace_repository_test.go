package localfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
)

func TestWorkspaceRepositoryReadCodeRejectsNonRegularFile(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "subdir"), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	_, err := repo.ReadCode(context.Background(), domainworkspace.ReadCodeRequest{
		Workspace:    workspace,
		RelativePath: "subdir",
	})
	if err == nil {
		t.Fatalf("expected non-regular file to be rejected")
	}
}

func TestWorkspaceRepositoryReadCodePropagatesContextError(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.ReadCode(ctx, domainworkspace.ReadCodeRequest{
		Workspace:    workspace,
		RelativePath: "main.go",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestWorkspaceRepositoryReadCodeRejectsHardLink(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Link(outside, filepath.Join(workspace, "hard.txt")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}

	_, err := repo.ReadCode(context.Background(), domainworkspace.ReadCodeRequest{
		Workspace:    workspace,
		RelativePath: "hard.txt",
	})
	if err == nil {
		t.Fatalf("expected hard link to be rejected")
	}
}

func TestWorkspaceRepositoryListDirectoryPropagatesCancellation(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.ListDirectory(ctx, domainworkspace.ListDirectoryRequest{Workspace: workspace})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestWorkspaceRepositorySearchCodeResolvesSymlinkWorkspaceRoot(t *testing.T) {
	repo := NewWorkspaceRepository()
	realWorkspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(realWorkspace, "main.go"), []byte("pattern match"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	parent := t.TempDir()
	linkWorkspace := filepath.Join(parent, "workspace-link")
	if err := os.Symlink(realWorkspace, linkWorkspace); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	result, err := repo.SearchCode(context.Background(), domainworkspace.SearchCodeRequest{
		Workspace: linkWorkspace,
		Pattern:   "pattern",
	})
	if err != nil {
		t.Fatalf("SearchCode failed: %v", err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %+v", result)
	}
	if result.Matches[0].File != "main.go" {
		t.Fatalf("expected canonicalized relative path, got %+v", result.Matches[0])
	}
}

func TestWorkspaceRepositoryFileExists(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	exists, err := repo.FileExists(context.Background(), domainworkspace.FileExistsRequest{
		Workspace:    workspace,
		RelativePath: "main.go",
	})
	if err != nil {
		t.Fatalf("FileExists failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected file to exist")
	}
}

func TestWorkspaceRepositoryFileExistsAllowsDirectories(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "notes"), 0755); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	exists, err := repo.FileExists(context.Background(), domainworkspace.FileExistsRequest{
		Workspace:    workspace,
		RelativePath: "notes",
	})
	if err != nil {
		t.Fatalf("FileExists failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected directory to count as existing path")
	}
}

func TestWorkspaceRepositoryWriteFile(t *testing.T) {
	repo := NewWorkspaceRepository()
	workspace := t.TempDir()

	result, err := repo.WriteFile(context.Background(), domainworkspace.WriteFileRequest{
		Workspace:    workspace,
		RelativePath: "notes/todo.txt",
		Content:      "hello",
	})
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if result.RelativePath != filepath.Join("notes", "todo.txt") || result.BytesWritten != 5 {
		t.Fatalf("unexpected result: %+v", result)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "notes", "todo.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}
