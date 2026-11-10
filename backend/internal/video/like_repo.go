package video

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) Like(ctx context.Context, like *Like) error {
	return r.db.WithContext(ctx).Create(like).Error
}

func (r *LikeRepository) Unlike(ctx context.Context, like *Like) error {
	return r.db.WithContext(ctx).
		Where("video_id = ? AND account_id = ?", like.VideoID, like.AccountID).
		Delete(&Like{}).Error
}

func (r *LikeRepository) LikeIgnoreDuplicate(ctx context.Context, like *Like) (created bool, err error) {
	return r.LikeInTx(ctx, nil, like)
}

func (r *LikeRepository) DeleteByVideoAndAccount(ctx context.Context, videoID, accountID uint) (bool, error) {
	return r.UnlikeInTx(ctx, nil, &Like{VideoID: videoID, AccountID: accountID})
}

func (r *LikeRepository) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Like{}).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *LikeRepository) BatchGetLiked(ctx context.Context, videoIDs []uint, accountID uint) (map[uint]bool, error) {
	likeMap := make(map[uint]bool)
	if len(videoIDs) == 0 {
		return likeMap, nil
	}
	if accountID == 0 {
		return likeMap, nil
	}
	var likes []Like
	err := r.db.WithContext(ctx).Model(&Like{}).
		Where("video_id IN ? AND account_id = ?", videoIDs, accountID).
		Find(&likes).Error
	if err != nil {
		return nil, err
	}
	for _, like := range likes {
		likeMap[like.VideoID] = true
	}
	return likeMap, nil
}

func (r *LikeRepository) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	var videos []Video
	if accountID == 0 {
		return videos, nil
	}
	err := r.db.WithContext(ctx).
		Model(&Video{}).
		Joins("JOIN likes ON likes.video_id = videos.id").
		Where("likes.account_id = ?", accountID).
		Order("likes.created_at desc").
		Limit(200).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

// LikeInTx inserts a like, returning false for a duplicate or invalid IDs.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (r *LikeRepository) LikeInTx(ctx context.Context, tx *gorm.DB, like *Like) (bool, error) {
	if like == nil || like.VideoID == 0 || like.AccountID == 0 {
		return false, nil
	}
	err := transactionDB(ctx, tx, r.db).Create(like).Error
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return false, nil
	}
	return err == nil, err
}

// UnlikeInTx returns whether an existing like was deleted.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (r *LikeRepository) UnlikeInTx(ctx context.Context, tx *gorm.DB, like *Like) (bool, error) {
	if like == nil || like.VideoID == 0 || like.AccountID == 0 {
		return false, nil
	}
	result := transactionDB(ctx, tx, r.db).Where("video_id = ? AND account_id = ?", like.VideoID, like.AccountID).Delete(&Like{})
	return result.RowsAffected > 0, result.Error
}

// videoRepository shares this repository's database for optional service dependencies.
func (r *LikeRepository) videoRepository() *VideoRepository { return NewVideoRepository(r.db) }
