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
	Followers []*account.Account `json:"followers"`
}

type GetAllVloggersRequest struct {
	FollowerID uint `json:"follower_id"`
}

type GetAllVloggersResponse struct {
	Vloggers []*account.Account `json:"vloggers"`
}
