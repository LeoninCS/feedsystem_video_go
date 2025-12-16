package feed

import (
	"context"
	"feedsystem_video_go/internal/video"
	"time"

	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func (repo *FeedRepository) ListLatest(ctx context.Context, limit int, latestBefore time.Time) ([]video.Video, error) {
	var videos []video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).
		Order("create_time DESC")
	if !latestBefore.IsZero() {
		query = query.Where("create_time < ?", latestBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (repo *FeedRepository) ListLikesCount(ctx context.Context, limit int, likesCountBefore int64) ([]video.Video, error) {
	var videos []video.Video
	query := repo.db.WithContext(ctx).Model(&video.Video{}).
		Order("likes_count DESC")
	if likesCountBefore > 0 {
		query = query.Where("likes_count < ?", likesCountBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
