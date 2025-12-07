package feed

import (
	"context"
	"feedsystem_video_go/internal/video"
	"time"
)

type FeedService struct {
	repo *FeedRepository
}

type FeedResponse struct {
	VideoList []video.Video `json:"video_list"`
	NextTime  time.Time     `json:"next_time"`
	HasMore   bool          `json:"has_more"`
}

func NewFeedService(repo *FeedRepository) *FeedService {
	return &FeedService{repo: repo}
}

func (f *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time) (FeedResponse, error) {
	videos, err := f.repo.ListLatest(ctx, limit, latestBefore)
	if err != nil {
		return FeedResponse{}, err
	}
	var nextTime time.Time
	if len(videos) > 0 {
		nextTime = videos[len(videos)-1].CreateTime
	} else {
		nextTime = time.Time{}
	}
	hasMore := len(videos) == limit
	resp := FeedResponse{
		VideoList: videos,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}
	return resp, nil
}
