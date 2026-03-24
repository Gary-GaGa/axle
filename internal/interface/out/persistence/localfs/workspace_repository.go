package localfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
	infrafiles "github.com/garyellow/axle/internal/infrastructure/files"
)

const maxSearchFileBytes = 1024 * 1024

// WorkspaceRepository implements read-only workspace exploration against the local filesystem.
type WorkspaceRepository struct{}

// NewWorkspaceRepository creates a filesystem-backed workspace repository.
func NewWorkspaceRepository() *WorkspaceRepository {
	return &WorkspaceRepository{}
}

// ReadCode reads a file within the workspace sandbox.
func (r *WorkspaceRepository) ReadCode(ctx context.Context, req domainworkspace.ReadCodeRequest) (domainworkspace.CodeSnippet, error) {
	root, _, err := infrafiles.OpenWorkspaceRoot(req.Workspace)
	if err != nil {
		return domainworkspace.CodeSnippet{}, err
	}
	defer root.Close()

	relPath, err := domainworkspace.ValidateRelativePath(req.RelativePath)
	if err != nil {
		return domainworkspace.CodeSnippet{}, err
	}
	if err := infrafiles.EnsurePathHasNoSymlink(root, relPath, req.RelativePath); err != nil {
		return domainworkspace.CodeSnippet{}, err
	}

	select {
	case <-ctx.Done():
		return domainworkspace.CodeSnippet{}, ctx.Err()
	default:
	}

	info, err := root.Lstat(relPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domainworkspace.CodeSnippet{}, fmt.Errorf("檔案不存在: `%s`", req.RelativePath)
		}
		return domainworkspace.CodeSnippet{}, fmt.Errorf("讀取失敗: %w", err)
	}
	if !info.Mode().IsRegular() {
		return domainworkspace.CodeSnippet{}, fmt.Errorf("`%s` 不是一般檔案", req.RelativePath)
	}
	if err := infrafiles.EnsureSingleLink(info, req.RelativePath); err != nil {
		return domainworkspace.CodeSnippet{}, err
	}

	file, err := root.OpenFile(relPath, infrafiles.ReadOnlyOpenFlags, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return domainworkspace.CodeSnippet{}, fmt.Errorf("檔案不存在: `%s`", req.RelativePath)
		}
		return domainworkspace.CodeSnippet{}, fmt.Errorf("讀取失敗: %w", err)
	}
	defer file.Close()

	info, err = file.Stat()
	if err != nil {
		return domainworkspace.CodeSnippet{}, fmt.Errorf("讀取失敗: %w", err)
	}
	if !info.Mode().IsRegular() {
		return domainworkspace.CodeSnippet{}, fmt.Errorf("`%s` 不是一般檔案", req.RelativePath)
	}
	if err := infrafiles.EnsureSingleLink(info, req.RelativePath); err != nil {
		return domainworkspace.CodeSnippet{}, err
	}

	limit := domainworkspace.ClampReadBytes(req.MaxBytes)
	data, err := io.ReadAll(io.LimitReader(file, int64(limit+1)))
	if err != nil {
		return domainworkspace.CodeSnippet{}, fmt.Errorf("讀取失敗: %w", err)
	}
	truncated := false
	if len(data) > limit {
		data = data[:limit]
		truncated = true
	}

	return domainworkspace.CodeSnippet{
		RelativePath: req.RelativePath,
		Language:     domainworkspace.LanguageFromPath(req.RelativePath),
		Content:      string(data),
		Truncated:    truncated,
	}, nil
}

// ListDirectory returns a tree-like view of the target directory.
func (r *WorkspaceRepository) ListDirectory(ctx context.Context, req domainworkspace.ListDirectoryRequest) (domainworkspace.DirectoryTree, error) {
	root, absWorkspace, err := infrafiles.OpenWorkspaceRoot(req.Workspace)
	if err != nil {
		return domainworkspace.DirectoryTree{}, err
	}
	defer root.Close()

	targetRel, err := domainworkspace.ValidateRelativePath(req.RelativePath)
	if err != nil {
		return domainworkspace.DirectoryTree{}, err
	}
	if err := infrafiles.EnsurePathHasNoSymlink(root, targetRel, req.RelativePath); err != nil {
		return domainworkspace.DirectoryTree{}, err
	}

	info, err := root.Stat(targetRel)
	if err != nil {
		if os.IsNotExist(err) {
			return domainworkspace.DirectoryTree{}, fmt.Errorf("路徑不存在: `%s`", req.RelativePath)
		}
		return domainworkspace.DirectoryTree{}, fmt.Errorf("無法存取: %w", err)
	}
	if !info.IsDir() {
		return domainworkspace.DirectoryTree{}, fmt.Errorf("`%s` 不是目錄", req.RelativePath)
	}

	displayPath := targetRel
	if req.RelativePath == "" || req.RelativePath == "." {
		displayPath = filepath.Base(absWorkspace)
	}

	targetRoot := root
	if targetRel != "." {
		targetRoot, err = root.OpenRoot(targetRel)
		if err != nil {
			return domainworkspace.DirectoryTree{}, fmt.Errorf("無法存取: %w", err)
		}
		defer targetRoot.Close()
	}

	lines := []string{fmt.Sprintf("📂 %s/", displayPath)}
	count := 0
	truncated, err := r.buildTree(ctx, targetRoot, "", domainworkspace.ClampTreeDepth(req.Depth), domainworkspace.ClampTreeItems(req.ItemLimit), &lines, &count)
	if err != nil {
		return domainworkspace.DirectoryTree{}, err
	}
	return domainworkspace.DirectoryTree{DisplayPath: displayPath, Lines: lines, Truncated: truncated}, nil
}

func (r *WorkspaceRepository) buildTree(ctx context.Context, root *os.Root, prefix string, depth, limit int, lines *[]string, count *int) (bool, error) {
	if depth <= 0 || *count >= limit {
		return *count >= limit, nil
	}

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	entries, err := readDirEntries(root)
	if err != nil {
		if prefix != "" && isSkippableTraversalError(err) {
			return true, nil
		}
		return false, fmt.Errorf("讀取目錄失敗: %w", err)
	}

	truncated := false
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if *count >= limit {
			return true, nil
		}
		*count = *count + 1

		isLast := isLastVisibleEntry(entries, entry)
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if entry.IsDir() {
			*lines = append(*lines, fmt.Sprintf("%s%s📁 %s/", prefix, connector, name))

			childRoot, err := root.OpenRoot(name)
			if err != nil {
				if isSkippableTraversalError(err) {
					truncated = true
					continue
				}
				return false, fmt.Errorf("無法存取目錄 `%s`: %w", name, err)
			}
			childTruncated, err := r.buildTree(ctx, childRoot, childPrefix, depth-1, limit, lines, count)
			childRoot.Close()
			if err != nil {
				return false, err
			}
			if childTruncated {
				truncated = true
			}
			continue
		}
		*lines = append(*lines, fmt.Sprintf("%s%s%s", prefix, connector, name))
	}
	return truncated, nil
}

// SearchCode searches the workspace for matching text.
func (r *WorkspaceRepository) SearchCode(ctx context.Context, req domainworkspace.SearchCodeRequest) (domainworkspace.SearchCodeResult, error) {
	pattern, err := domainworkspace.ValidateSearchPattern(req.Pattern)
	if err != nil {
		return domainworkspace.SearchCodeResult{}, err
	}

	root, _, err := infrafiles.OpenWorkspaceRoot(req.Workspace)
	if err != nil {
		return domainworkspace.SearchCodeResult{}, err
	}
	defer root.Close()

	maxResults := domainworkspace.ClampSearchResults(req.MaxResults)
	lowerPattern := strings.ToLower(pattern)
	results := make([]domainworkspace.SearchMatch, 0, maxResults)

	truncated, err := r.searchTree(ctx, root, "", lowerPattern, maxResults, &results)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return domainworkspace.SearchCodeResult{}, ctxErr
			}
			return domainworkspace.SearchCodeResult{}, err
		}
		return domainworkspace.SearchCodeResult{Matches: results, Truncated: truncated}, fmt.Errorf("搜尋時發生錯誤: %w", err)
	}
	return domainworkspace.SearchCodeResult{Matches: results, Truncated: truncated}, nil
}

// FileExists checks whether a workspace path exists within the sandbox.
func (r *WorkspaceRepository) FileExists(ctx context.Context, req domainworkspace.FileExistsRequest) (bool, error) {
	if err := domainworkspace.ValidateWritePath(req.RelativePath); err != nil {
		return false, err
	}

	root, _, err := infrafiles.OpenWorkspaceRoot(req.Workspace)
	if err != nil {
		return false, err
	}
	defer root.Close()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	return infrafiles.RootedFileExists(root, req.RelativePath, req.RelativePath)
}

// WriteFile writes a file within the workspace sandbox.
func (r *WorkspaceRepository) WriteFile(ctx context.Context, req domainworkspace.WriteFileRequest) (domainworkspace.WriteFileResult, error) {
	if err := domainworkspace.ValidateWritePath(req.RelativePath); err != nil {
		return domainworkspace.WriteFileResult{}, err
	}
	if err := domainworkspace.ValidateWriteContent(req.Content); err != nil {
		return domainworkspace.WriteFileResult{}, err
	}

	root, _, err := infrafiles.OpenWorkspaceRoot(req.Workspace)
	if err != nil {
		return domainworkspace.WriteFileResult{}, err
	}
	defer root.Close()

	select {
	case <-ctx.Done():
		return domainworkspace.WriteFileResult{}, ctx.Err()
	default:
	}

	if err := infrafiles.WriteRootedFile(root, req.RelativePath, req.RelativePath, []byte(req.Content), 0644); err != nil {
		return domainworkspace.WriteFileResult{}, err
	}
	return domainworkspace.WriteFileResult{
		RelativePath: targetRelativePath(req.RelativePath),
		BytesWritten: len(req.Content),
	}, nil
}

func targetRelativePath(relPath string) string {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return domainworkspace.NormalizeRelativePath(relPath)
	}
	return cleaned
}

func (r *WorkspaceRepository) searchTree(ctx context.Context, root *os.Root, prefix, lowerPattern string, maxResults int, results *[]domainworkspace.SearchMatch) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	entries, err := readDirEntries(root)
	if err != nil {
		if prefix != "" && isSkippableTraversalError(err) {
			return true, nil
		}
		return false, fmt.Errorf("讀取目錄失敗: %w", err)
	}

	truncated := false
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				continue
			}
			childPrefix := name
			if prefix != "" {
				childPrefix = filepath.Join(prefix, name)
			}
			childRoot, err := root.OpenRoot(name)
			if err != nil {
				if isSkippableTraversalError(err) {
					truncated = true
					continue
				}
				return false, fmt.Errorf("無法存取目錄 `%s`: %w", childPrefix, err)
			}
			childTruncated, err := r.searchTree(ctx, childRoot, childPrefix, lowerPattern, maxResults, results)
			childRoot.Close()
			if err != nil {
				return false, err
			}
			if childTruncated {
				truncated = true
				if len(*results) >= maxResults {
					return true, nil
				}
			}
			continue
		}

		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if domainworkspace.IsBinaryExt(filepath.Ext(name)) {
			continue
		}

		info, err := root.Lstat(name)
		if err != nil {
			if isSkippableTraversalError(err) {
				truncated = true
				continue
			}
			return false, fmt.Errorf("無法存取檔案 `%s`: %w", filepath.Join(prefix, name), err)
		}
		if !info.Mode().IsRegular() || info.Size() > maxSearchFileBytes {
			continue
		}
		relPath := name
		if prefix != "" {
			relPath = filepath.Join(prefix, name)
		}
		if err := infrafiles.EnsureSingleLink(info, relPath); err != nil {
			return false, err
		}

		file, err := root.OpenFile(name, infrafiles.ReadOnlyOpenFlags, 0)
		if err != nil {
			if isSkippableTraversalError(err) {
				truncated = true
				continue
			}
			return false, fmt.Errorf("讀取檔案 `%s` 失敗: %w", relPath, err)
		}
		openInfo, statErr := file.Stat()
		if statErr != nil {
			file.Close()
			if isSkippableTraversalError(statErr) {
				truncated = true
				continue
			}
			return false, fmt.Errorf("讀取檔案 `%s` 失敗: %w", relPath, statErr)
		}
		if !openInfo.Mode().IsRegular() {
			file.Close()
			continue
		}
		if err := infrafiles.EnsureSingleLink(openInfo, relPath); err != nil {
			file.Close()
			return false, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxSearchFileBytes+1))
		file.Close()
		if readErr != nil {
			if isSkippableTraversalError(readErr) {
				truncated = true
				continue
			}
			return false, fmt.Errorf("讀取檔案 `%s` 失敗: %w", relPath, readErr)
		}
		if len(data) > maxSearchFileBytes {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for lineIdx, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerPattern) {
				if len(*results) >= maxResults {
					return true, nil
				}
				*results = append(*results, domainworkspace.SearchMatch{
					File:    relPath,
					Line:    lineIdx + 1,
					Content: domainworkspace.TrimSearchContent(line),
				})
			}
		}
	}
	return truncated, nil
}

func readDirEntries(root *os.Root) ([]os.DirEntry, error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di
		}
		return entries[i].Name() < entries[j].Name()
	})
	return entries, nil
}

func isLastVisibleEntry(entries []os.DirEntry, current os.DirEntry) bool {
	visibleIndex := -1
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		visibleIndex++
		if entry.Name() == current.Name() {
			break
		}
	}
	lastVisibleIndex := -1
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		lastVisibleIndex++
	}
	return visibleIndex == lastVisibleIndex
}

func isSkippableTraversalError(err error) bool {
	return err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) &&
		(os.IsPermission(err) || os.IsNotExist(err))
}
