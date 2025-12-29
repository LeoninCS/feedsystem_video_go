package video

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rediscache "feedsystem_video_go/internal/middleware/redis"
)

// 更新视频流行度缓存
func UpdatePopularityCache(ctx context.Context, cache *rediscache.Client, id uint, change int64) {
	if cache == nil || id == 0 || change == 0 {
		return
	}

	_ = cache.Del(context.Background(), fmt.Sprintf("video:detail:id=%d", id))

	now := time.Now().UTC().Truncate(time.Minute)
	windowKey := "hot:video:1m:" + now.Format("200601021504")
	member := strconv.FormatUint(uint64(id), 10)

	opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_ = cache.ZincrBy(opCtx, windowKey, member, float64(change))
	_ = cache.Expire(opCtx, windowKey, 2*time.Hour)
}
