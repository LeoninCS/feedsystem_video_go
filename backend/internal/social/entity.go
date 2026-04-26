package social

import "feedsystem_video_go/internal/account"

type Social struct {
	ID         uint `gorm:"primaryKey"`
	FollowerID uint `gorm:"not null;index:idx_social_follower;uniqueIndex:idx_social_follower_vlogger"`
	VloggerID  uint `gorm:"not null;index:idx_social_vlogger;uniqueIndex:idx_social_follower_vlogger"`
}

type FollowRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type UnfollowRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type GetAllFollowersRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type GetAllFollowersResponse struct {
	Followers     []*account.Account `json:"followers"`
	FollowerCount int64              `json:"follower_count"`
}

type GetAllVloggersResponse struct {
	Vloggers     []*account.Account `json:"vloggers"`
	VloggerCount int64              `json:"vlogger_count"`
}

type SocialCounts struct {
	FollowerCount int64 `json:"follower_count"`
	VloggerCount  int64 `json:"vlogger_count"`
}

type GetAllVloggersRequest struct {
	FollowerID uint `json:"follower_id"`
}
