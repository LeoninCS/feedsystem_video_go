package video

import (
	"context"

	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (vr *VideoRepository) CreateVideo(ctx context.Context, video *Video) error {
	if err := vr.db.WithContext(ctx).Create(video).Error; err != nil {
		return err
	}
	return nil
}

func (vr *VideoRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]Video, error) {
	var videos []Video
	if err := vr.db.WithContext(ctx).
		Where("author_id = ?", authorID).
		Order("create_time desc").
		Limit(5).
		Offset(0).
		Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (vr *VideoRepository) ListLatest(ctx context.Context) ([]Video, error) {
	var videos []Video
	if err := vr.db.WithContext(ctx).
		Order("create_time desc").
		Limit(5).
		Offset(0).
		Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (vr *VideoRepository) GetByID(ctx context.Context, id uint) (*Video, error) {
	var video Video
	if err := vr.db.WithContext(ctx).First(&video, id).Error; err != nil {
		return nil, err
	}
	return &video, nil
}
