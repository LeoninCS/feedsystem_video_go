package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rediscache "feedsystem_video_go/internal/redis"
)

type VideoService struct {
	repo     *VideoRepository
	cache    *rediscache.Client
	cacheTTL time.Duration
}

func NewVideoService(repo *VideoRepository, cache *rediscache.Client) *VideoService {
	return &VideoService{repo: repo, cache: cache, cacheTTL: 5 * time.Minute}
}

func (vs *VideoService) Publish(ctx context.Context, video *Video) error {
	if video.Title == "" {
		return errors.New("title is required")
	}
	if video.PlayURL == "" {
		return errors.New("play url is required")
	}
	if video.CoverURL == "" {
		return errors.New("cover url is required")
	}
	if err := vs.repo.CreateVideo(ctx, video); err != nil {
		return err
	}
	return nil
}

func (vs *VideoService) Delete(ctx context.Context, id uint, authorID uint) error {
	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if video.AuthorID != authorID {
		return errors.New("unauthorized")
	}
	if err := vs.repo.DeleteVideo(ctx, id); err != nil {
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

func (vs *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	if vs.cache != nil {
		cacheKey := fmt.Sprintf("video:detail:id=%d", id)
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if b, err := vs.cache.GetBytes(cacheCtx, cacheKey); err == nil {
			var cached Video
			if err := json.Unmarshal(b, &cached); err == nil {
				return &cached, nil
			}
		} else if rediscache.IsMiss(err) {
			lockKey := "lock:" + cacheKey
			token, locked, _ := vs.cache.Lock(cacheCtx, lockKey, 500*time.Millisecond)
			if locked {
				defer func() { _ = vs.cache.Unlock(context.Background(), lockKey, token) }()
				if b, err := vs.cache.GetBytes(cacheCtx, cacheKey); err == nil {
					var cached Video
					if err := json.Unmarshal(b, &cached); err == nil {
						return &cached, nil
					}
				} else { // 缓存未命中，从数据库中查询
					video, err := vs.repo.GetByID(ctx, id)
					if err != nil {
						return nil, err
					}
					if b, err := json.Marshal(video); err == nil {
						_ = vs.cache.SetBytes(cacheCtx, cacheKey, b, vs.cacheTTL)
					}
					return video, nil
				}
			} else { // 缓存未命中，其他goroutine正在查询，等待
				for i := 0; i < 5; i++ {
					time.Sleep(20 * time.Millisecond)
					if b, err := vs.cache.GetBytes(cacheCtx, cacheKey); err == nil {
						var cached Video
						if err := json.Unmarshal(b, &cached); err == nil {
							return &cached, nil
						}
					}
				}
			}
		}
	}

	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if vs.cache != nil {
		cacheKey := fmt.Sprintf("video:detail:id=%d", id)
		if b, err := json.Marshal(video); err == nil {
			cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			_ = vs.cache.SetBytes(cacheCtx, cacheKey, b, vs.cacheTTL)
		}
	}
	return video, nil
}

func (vs *VideoService) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {
	if err := vs.repo.UpdateLikesCount(ctx, id, likesCount); err != nil {
		return err
	}
	return nil
}
