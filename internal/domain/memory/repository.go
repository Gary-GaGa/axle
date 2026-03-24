package memory

import "context"

// Repository persists per-user memory entries.
type Repository interface {
	Load(ctx context.Context, userID int64) ([]Entry, error)
	Save(ctx context.Context, userID int64, entries []Entry) error
	Clear(ctx context.Context, userID int64) error
}
