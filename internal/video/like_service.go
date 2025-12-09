package video

import (
	"context"
	"errors"
	"time"
)

type LikeService struct {
	repo *LikeRepository
}

func NewLikeService(repo *LikeRepository) *LikeService {
	return &LikeService{repo: repo}
}

func (s *LikeService) Like(ctx context.Context, like *Like) error {
	if isLiked, err := s.IsLiked(ctx, like.VideoID, like.AccountID); err == nil && isLiked {
		return errors.New("user has liked this video")
	}
	like.CreatedAt = time.Now()
	return s.repo.Like(ctx, like)
}

func (s *LikeService) Unlike(ctx context.Context, like *Like) error {
	if isLiked, err := s.IsLiked(ctx, like.VideoID, like.AccountID); err == nil && !isLiked {
		return errors.New("user has not liked this video")
	}
	return s.repo.Unlike(ctx, like)
}

func (s *LikeService) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	return s.repo.IsLiked(ctx, videoID, accountID)
}

func (s *LikeService) GetLikesCount(ctx context.Context, videoID uint) (int64, error) {
	return s.repo.GetLikesCount(ctx, videoID)
}
