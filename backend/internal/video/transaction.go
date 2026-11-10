package video

import (
	"context"

	"gorm.io/gorm"
)

// transactionDB keeps all operations on the caller's transaction and context.
// A nil tx explicitly selects the repository's non-transactional connection.
func transactionDB(ctx context.Context, tx, fallback *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = fallback
	}
	return tx.WithContext(ctx)
}

// Transaction commits when fn returns nil and rolls back on an error or panic.
// Every repository operation in fn must receive the supplied tx.
func (r *LikeRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// Transaction commits when fn returns nil and rolls back on an error or panic.
// Every repository operation in fn must receive the supplied tx.
func (vr *VideoRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return vr.db.WithContext(ctx).Transaction(fn)
}

// Transaction commits when fn returns nil and rolls back on an error or panic.
// Every repository operation in fn must receive the supplied tx.
func (r *CommentRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
