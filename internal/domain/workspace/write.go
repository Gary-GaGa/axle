package workspace

import (
	"fmt"
	"strings"
)

const MaxWriteBytes = 1024 * 1024

// FileExistsRequest checks whether a workspace file already exists.
type FileExistsRequest struct {
	Workspace    string
	RelativePath string
}

// WriteFileRequest requests a file write within the workspace sandbox.
type WriteFileRequest struct {
	Workspace    string
	RelativePath string
	Content      string
}

// WriteFileResult summarizes a successful workspace write.
type WriteFileResult struct {
	RelativePath string
	BytesWritten int
}

// ValidateWritePath ensures a writable path is present.
func ValidateWritePath(relPath string) error {
	if strings.TrimSpace(relPath) == "" {
		return fmt.Errorf("寫入路徑不能為空")
	}
	if _, err := ValidateRelativePath(relPath); err != nil {
		return err
	}
	return nil
}

// ValidateWriteContent ensures write content stays within the supported bound.
func ValidateWriteContent(content string) error {
	if len(content) > MaxWriteBytes {
		return fmt.Errorf("⚠️ 內容超過 %d KB 上限，已拒絕寫入", MaxWriteBytes/1024)
	}
	return nil
}
