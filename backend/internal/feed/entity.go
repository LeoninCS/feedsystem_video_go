package feed

import "time"

type FeedAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type FeedVideoItem struct {
	ID          uint       `json:"id"`
	Author      FeedAuthor `json:"author"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	PlayURL     string     `json:"play_url"`
	CoverURL    string     `json:"cover_url"`
	CreateTime  int64      `json:"create_time"`
	LikesCount  int64      `json:"likes_count"`
	IsLiked     bool       `json:"is_liked"`
}

type ListLatestRequest struct {
	Limit      int   `json:"limit"`
	LatestTime int64 `json:"latest_time"`
}

type ListLatestResponse struct {
	VideoList []FeedVideoItem `json:"video_list"`
	NextTime  int64           `json:"next_time"`
	HasMore   bool            `json:"has_more"`
}

type ListLikesCountRequest struct {
	Limit            int    `json:"limit"`
	LikesCountBefore *int64 `json:"likes_count_before,omitempty"`
	IDBefore         *uint  `json:"id_before,omitempty"`
}

type LikesCountCursor struct {
	LikesCount int64
	ID         uint
}

type ListLikesCountResponse struct {
	VideoList            []FeedVideoItem `json:"video_list"`
	NextLikesCountBefore *int64          `json:"next_likes_count_before,omitempty"`
	NextIDBefore         *uint           `json:"next_id_before,omitempty"`
	HasMore              bool            `json:"has_more"`
}

type ListByFollowingRequest struct {
	Limit      int   `json:"limit"`
	LatestTime int64 `json:"latest_time"`
}

type ListByFollowingResponse struct {
	VideoList []FeedVideoItem `json:"video_list"`
	NextTime  int64           `json:"next_time"`
	HasMore   bool            `json:"has_more"`
}

type ListByPopularityRequest struct {
	Limit          int   `json:"limit"`
	AsOf           int64 `json:"as_of"`  // 服务器返回的分钟时间戳；第一页传0
	Offset         int   `json:"offset"` // 下一页从这里开始；第一页传0
	LatestIDBefore *uint `json:"latest_id_before,omitempty"`

	// DB fallback 用（可选）
	LatestPopularity int64     `json:"latest_popularity"`
	LatestBefore     time.Time `json:"latest_before"`
}

type ListByPopularityResponse struct {
	VideoList  []FeedVideoItem `json:"video_list"`
	AsOf       int64           `json:"as_of"`
	NextOffset int             `json:"next_offset"`
	HasMore    bool            `json:"has_more"`

	NextLatestPopularity *int64     `json:"next_latest_popularity,omitempty"`
	NextLatestBefore     *time.Time `json:"next_latest_before,omitempty"`
	NextLatestIDBefore   *uint      `json:"next_latest_id_before,omitempty"`
}
