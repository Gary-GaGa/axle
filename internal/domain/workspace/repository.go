package workspace

import "context"

const (
	MaxReadBytes     = 30 * 1024
	MaxSearchResults = 30
	MaxTreeDepth     = 5
	MaxTreeItems     = 200
)

// ReadCodeRequest requests a code file snapshot from a workspace.
type ReadCodeRequest struct {
	Workspace    string
	RelativePath string
	MaxBytes     int
}

// CodeSnippet represents a code file prepared for display.
type CodeSnippet struct {
	RelativePath string
	Language     string
	Content      string
	Truncated    bool
}

// ListDirectoryRequest requests a tree view from the workspace.
type ListDirectoryRequest struct {
	Workspace    string
	RelativePath string
	Depth        int
	ItemLimit    int
}

// DirectoryTree represents a formatted tree structure without transport decoration.
type DirectoryTree struct {
	DisplayPath string
	Lines       []string
	Truncated   bool
}

// SearchCodeRequest requests a lexical workspace search.
type SearchCodeRequest struct {
	Workspace  string
	Pattern    string
	MaxResults int
}

// SearchMatch represents one matched line inside the workspace.
type SearchMatch struct {
	File    string
	Line    int
	Content string
}

// SearchCodeResult contains search matches plus truncation metadata.
type SearchCodeResult struct {
	Matches   []SearchMatch
	Truncated bool
}

// Repository defines workspace exploration and write operations.
type Repository interface {
	ReadCode(ctx context.Context, req ReadCodeRequest) (CodeSnippet, error)
	ListDirectory(ctx context.Context, req ListDirectoryRequest) (DirectoryTree, error)
	SearchCode(ctx context.Context, req SearchCodeRequest) (SearchCodeResult, error)
	FileExists(ctx context.Context, req FileExistsRequest) (bool, error)
	WriteFile(ctx context.Context, req WriteFileRequest) (WriteFileResult, error)
}
