package feed

import (
	"context"
	"encoding/json"
	"feedsystem_video_go/internal/video"
	"fmt"
	rediscache "feedsystem_video_go/internal/redis"
	"time"
)

type FeedService struct {
	repo     *FeedRepository
	likeRepo *video.LikeRepository
	cache    *rediscache.Client
	cacheTTL time.Duration
}

func NewFeedService(repo *FeedRepository, likeRepo *video.LikeRepository, cache *rediscache.Client) *FeedService {
	return &FeedService{repo: repo, likeRepo: likeRepo, cache: cache, cacheTTL: 5 * time.Second}
}

func (f *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
	var cacheKey string
	if viewerAccountID == 0 && f.cache != nil {
		before := int64(0)
		if !latestBefore.IsZero() {
			before = latestBefore.Unix()
		}
		cacheKey = fmt.Sprintf("feed:listLatest:limit=%d:before=%d", limit, before)

		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		b, err := f.cache.GetBytes(cacheCtx, cacheKey)
		if err == nil {
			var cached ListLatestResponse
			if err := json.Unmarshal(b, &cached); err == nil {
				return cached, nil
			}
		}
	}

	videos, err := f.repo.ListLatest(ctx, limit, latestBefore)
	if err != nil {
		return ListLatestResponse{}, err
	}
	var nextTime int64
	if len(videos) > 0 {
		nextTime = videos[len(videos)-1].CreateTime.Unix()
	} else {
		nextTime = 0
	}
	hasMore := len(videos) == limit
	feedVideos := make([]FeedVideoItem, 0, len(videos))
	for _, video := range videos {
		var isLiked bool
		if viewerAccountID == 0 {
			isLiked = false
		} else {
			isLiked, err = f.likeRepo.IsLiked(ctx, video.ID, viewerAccountID)
			if err != nil {
				return ListLatestResponse{}, err
			}
		}
		feedVideos = append(feedVideos, FeedVideoItem{
			ID:          video.ID,
			Author:      FeedAuthor{ID: video.AuthorID, Username: video.Username},
			Title:       video.Title,
			Description: video.Description,
			PlayURL:     video.PlayURL,
			CoverURL:    video.CoverURL,
			CreateTime:  video.CreateTime.Unix(),
			LikesCount:  video.LikesCount,
			IsLiked:     isLiked,
		})
	}
	resp := ListLatestResponse{
		VideoList: feedVideos,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}

	if cacheKey != "" {
		if b, err := json.Marshal(resp); err == nil {
			cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			_ = f.cache.SetBytes(cacheCtx, cacheKey, b, f.cacheTTL)
		}
	}
	return resp, nil
}

func (f *FeedService) ListLikesCount(ctx context.Context, limit int, cursor *LikesCountCursor, viewerAccountID uint) (ListLikesCountResponse, error) {
	videos, err := f.repo.ListLikesCountWithCursor(ctx, limit, cursor)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	hasMore := len(videos) == limit
	feedVideos := make([]FeedVideoItem, 0, len(videos))
	for _, video := range videos {
		var isLiked bool
		if viewerAccountID == 0 {
			isLiked = false
		} else {
			isLiked, err = f.likeRepo.IsLiked(ctx, video.ID, viewerAccountID)
			if err != nil {
				return ListLikesCountResponse{}, err
			}
		}
		feedVideos = append(feedVideos, FeedVideoItem{
			ID:          video.ID,
			Author:      FeedAuthor{ID: video.AuthorID, Username: video.Username},
			Title:       video.Title,
			Description: video.Description,
			PlayURL:     video.PlayURL,
			CoverURL:    video.CoverURL,
			CreateTime:  video.CreateTime.Unix(),
			LikesCount:  video.LikesCount,
			IsLiked:     isLiked,
		})
	}
	resp := ListLikesCountResponse{
		VideoList: feedVideos,
		HasMore:   hasMore,
	}
	if len(videos) > 0 {
		last := videos[len(videos)-1]
		nextLikesCountBefore := last.LikesCount
		nextIDBefore := last.ID
		resp.NextLikesCountBefore = &nextLikesCountBefore
		resp.NextIDBefore = &nextIDBefore
	}
	return resp, nil
}

func (f *FeedService) ListByFollowing(ctx context.Context, limit int, viewerAccountID uint) (ListByFollowingResponse, error) {
	videos, err := f.repo.ListByFollowing(ctx, limit, viewerAccountID)
	if err != nil {
		return ListByFollowingResponse{}, err
	}
	var nextTime int64
	if len(videos) > 0 {
		nextTime = videos[len(videos)-1].CreateTime.Unix()
	} else {
		nextTime = 0
	}
	hasMore := len(videos) == limit
	feedVideos := make([]FeedVideoItem, 0, len(videos))
	for _, video := range videos {
		var isLiked bool
		if viewerAccountID == 0 {
			isLiked = false
		} else {
			isLiked, err = f.likeRepo.IsLiked(ctx, video.ID, viewerAccountID)
			if err != nil {
				return ListByFollowingResponse{}, err
			}
		}
		feedVideos = append(feedVideos, FeedVideoItem{
			ID:          video.ID,
			Author:      FeedAuthor{ID: video.AuthorID, Username: video.Username},
			Title:       video.Title,
			Description: video.Description,
			PlayURL:     video.PlayURL,
			CoverURL:    video.CoverURL,
			CreateTime:  video.CreateTime.Unix(),
			LikesCount:  video.LikesCount,
			IsLiked:     isLiked,
		})
	}
	resp := ListByFollowingResponse{
		VideoList: feedVideos,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}
	return resp, nil
}
