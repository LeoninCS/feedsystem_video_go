package video

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (vr *VideoRepository) CreateVideo(ctx context.Context, video *Video) error {
	return vr.CreateVideoInTx(ctx, nil, video)
}

func (vr *VideoRepository) CreateMsg(ctx context.Context, Msg *OutboxMsg) error {
	return vr.CreateMsgInTx(ctx, nil, Msg)
}

func (vr *VideoRepository) DeleteVideo(ctx context.Context, id uint) error {
	if err := vr.db.WithContext(ctx).Delete(&Video{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (vr *VideoRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]Video, error) {
	var videos []Video
	if err := vr.db.WithContext(ctx).
		Where("author_id = ?", authorID).
		Order("create_time desc").
		Limit(200).
		Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (vr *VideoRepository) GetByID(ctx context.Context, id uint) (*Video, error) {
	var video Video
	if err := vr.db.WithContext(ctx).First(&video, id).Error; err != nil {
		return (*Video)(nil), err
	}
	return &video, nil
}

func (vr *VideoRepository) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {
	if err := vr.db.WithContext(ctx).Model(&Video{}).
		Where("id = ?", id).
		Update("likes_count", likesCount).Error; err != nil {
		return err
	}
	return nil
}

func (vr *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var video Video
	if err := vr.db.WithContext(ctx).First(&video, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (vr *VideoRepository) UpdatePopularity(ctx context.Context, id uint, change int64) error {
	if err := vr.db.WithContext(ctx).Model(&Video{}).
		Where("id = ?", id).
		Update("popularity", gorm.Expr("popularity + ?", change)).Error; err != nil {
		return err
	}
	return nil
}

func (vr *VideoRepository) ChangeLikesCount(ctx context.Context, id uint, change int64) error {
	return vr.ChangeLikesCountInTx(ctx, nil, id, change)
}

func (vr *VideoRepository) ChangePopularity(ctx context.Context, id uint, change int64) error {
	return vr.ChangePopularityInTx(ctx, nil, id, change)
}

func (vr *VideoRepository) CountByAuthor(ctx context.Context, authorID uint) (int64, error) {
	var count int64
	if err := vr.db.WithContext(ctx).Model(&Video{}).Where("author_id = ?", authorID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (vr *VideoRepository) TotalLikesByAuthor(ctx context.Context, authorID uint) (int64, error) {
	var total int64
	if err := vr.db.WithContext(ctx).Model(&Video{}).Where("author_id = ?", authorID).Select("COALESCE(SUM(likes_count), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// CreateVideoInTx creates a video.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) CreateVideoInTx(ctx context.Context, tx *gorm.DB, video *Video) error {
	return transactionDB(ctx, tx, vr.db).Create(video).Error
}

// CreateMsgInTx creates an outbox message.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) CreateMsgInTx(ctx context.Context, tx *gorm.DB, msg *OutboxMsg) error {
	return transactionDB(ctx, tx, vr.db).Create(msg).Error
}

// RequireVideoInTx returns gorm.ErrRecordNotFound when the video is missing.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) RequireVideoInTx(ctx context.Context, tx *gorm.DB, id uint) error {
	var video Video
	return transactionDB(ctx, tx, vr.db).Select("id").First(&video, id).Error
}

// AddTagsInTx attaches tags to a video.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) AddTagsInTx(ctx context.Context, tx *gorm.DB, videoID uint, names []string) error {
	db := transactionDB(ctx, tx, vr.db)
	for _, name := range names {
		var tag Tag
		if err := db.Where("name = ?", name).FirstOrCreate(&tag, Tag{Name: name}).Error; err != nil {
			return err
		}
		if err := db.Create(&VideoTag{VideoID: videoID, TagID: tag.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ChangeLikesCountInTx changes the like count without going below zero.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) ChangeLikesCountInTx(ctx context.Context, tx *gorm.DB, id uint, change int64) error {
	return transactionDB(ctx, tx, vr.db).Model(&Video{}).Where("id = ?", id).UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count + ?, 0)", change)).Error
}

// ChangePopularityInTx changes popularity without going below zero.
// Pass tx inside a transaction. A nil tx uses the repository connection and commits
// independently; it can self-deadlock on rows locked by the caller until
// innodb_lock_wait_timeout (自锁).
func (vr *VideoRepository) ChangePopularityInTx(ctx context.Context, tx *gorm.DB, id uint, change int64) error {
	return transactionDB(ctx, tx, vr.db).Model(&Video{}).Where("id = ?", id).UpdateColumn("popularity", gorm.Expr("GREATEST(popularity + ?, 0)", change)).Error
}
