package video

import (
	"context"
	"errors"
)

type VideoService struct {
	repo *VideoRepository
}

func NewVideoService(repo *VideoRepository) *VideoService {
	return &VideoService{repo: repo}
}

func (vs *VideoService) Publish(ctx context.Context, video *Video) error {
	if video.Title == "" {
		return errors.New("title is required")
	}
	if video.PlayURL == "" {
		return errors.New("play url is required")
	}
	if err := vs.repo.CreateVideo(ctx, video); err != nil {
		return err
	}
	return nil
}

func (vs *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	videos, err := vs.repo.ListByAuthorID(ctx, int64(authorID))
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (vs *VideoService) ListLatest(ctx context.Context) ([]Video, error) {
	videos, err := vs.repo.ListLatest(ctx)
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (vs *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return video, nil
}
