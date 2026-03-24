package dto

// ReadCodeInput describes a read-code request from an input adapter.
type ReadCodeInput struct {
	Workspace string
	Path      string
}

// ReadCodeOutput is the usecase-safe response for a code file snapshot.
type ReadCodeOutput struct {
	Path      string
	Language  string
	Content   string
	Truncated bool
}

// ListDirectoryInput describes a directory listing request.
type ListDirectoryInput struct {
	Workspace string
	Path      string
	Depth     int
}

// ListDirectoryOutput represents a tree-like directory view.
type ListDirectoryOutput struct {
	DisplayPath string
	Lines       []string
	Truncated   bool
}

// SearchCodeInput describes a workspace text search.
type SearchCodeInput struct {
	Workspace string
	Pattern   string
}

// SearchMatch is returned to input adapters instead of exposing domain structs directly.
type SearchMatch struct {
	File    string
	Line    int
	Content string
}

// SearchCodeOutput contains lexical code-search results.
type SearchCodeOutput struct {
	Pattern   string
	Matches   []SearchMatch
	Truncated bool
}

// FileExistsInput describes a writable workspace existence check.
type FileExistsInput struct {
	Workspace string
	Path      string
}

// FileExistsOutput reports whether a workspace path currently exists.
type FileExistsOutput struct {
	Path   string
	Exists bool
}

// WriteFileInput describes a writable workspace file write.
type WriteFileInput struct {
	Workspace string
	Path      string
	Content   string
}

// WriteFileOutput summarizes a successful workspace write.
type WriteFileOutput struct {
	Path         string
	BytesWritten int
}
