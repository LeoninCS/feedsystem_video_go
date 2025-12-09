package video

import (
	"context"
	"errors"
	"time"
)

type LikeService struct {
	repo      *LikeRepository
	VideoRepo *VideoRepository
}

func NewLikeService(repo *LikeRepository, videoRepo *VideoRepository) *LikeService {
	return &LikeService{repo: repo, VideoRepo: videoRepo}
}

func (s *LikeService) Like(ctx context.Context, like *Like) error {
	if isLiked, err := s.IsLiked(ctx, like.VideoID, like.AccountID); err == nil && isLiked {
		return errors.New("user has liked this video")
	}
	like.CreatedAt = time.Now()
	if err := s.repo.Like(ctx, like); err != nil {
		return err
	}
	likesCount, err := s.GetLikesCount(ctx, like.VideoID)
	if err != nil {
		return err
	}
	if err := s.VideoRepo.UpdateLikesCount(ctx, like.VideoID, likesCount); err != nil {
		return err
	}
	return nil
}

func (s *LikeService) Unlike(ctx context.Context, like *Like) error {
	if isLiked, err := s.IsLiked(ctx, like.VideoID, like.AccountID); err == nil && !isLiked {
		return errors.New("user has not liked this video")
	}
	if err := s.repo.Unlike(ctx, like); err != nil {
		return err
	}
	likesCount, err := s.GetLikesCount(ctx, like.VideoID)
	if err != nil {
		return err
	}
	if err := s.VideoRepo.UpdateLikesCount(ctx, like.VideoID, likesCount); err != nil {
		return err
	}
	return nil
}

func (s *LikeService) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	return s.repo.IsLiked(ctx, videoID, accountID)
}

func (s *LikeService) GetLikesCount(ctx context.Context, videoID uint) (int64, error) {
	return s.repo.GetLikesCount(ctx, videoID)
}
