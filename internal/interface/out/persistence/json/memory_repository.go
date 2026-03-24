package jsonmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	domainmemory "github.com/garyellow/axle/internal/domain/memory"
	"github.com/garyellow/axle/internal/infrastructure/files"
)

const memoryDir = "memory"

type Repository struct {
	baseDir string
}

func NewMemoryRepository(axleDir string) (*Repository, error) {
	dir := filepath.Join(axleDir, memoryDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("建立記憶目錄失敗: %w", err)
	}
	return &Repository{baseDir: dir}, nil
}

func (r *Repository) Load(ctx context.Context, userID int64) ([]domainmemory.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(r.userFile(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("讀取記憶失敗: %w", err)
	}

	var entries []domainmemory.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("解析記憶失敗: %w", err)
	}
	return entries, nil
}

func (r *Repository) Save(ctx context.Context, userID int64, entries []domainmemory.Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化記憶失敗: %w", err)
	}
	if err := files.WriteFileAtomic(r.userFile(userID), data, 0600); err != nil {
		return fmt.Errorf("寫入記憶失敗: %w", err)
	}
	return nil
}

func (r *Repository) Clear(ctx context.Context, userID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(r.userFile(userID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("移除記憶失敗: %w", err)
	}
	return nil
}

func (r *Repository) userFile(userID int64) string {
	return filepath.Join(r.baseDir, fmt.Sprintf("%d.json", userID))
}
