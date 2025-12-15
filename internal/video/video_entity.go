package video

import "time"

type Video struct {
	ID          uint      `gorm:"primaryKey"`
	AuthorID    uint      `gorm:"index;not null"`
	Username    string    `gorm:"type:varchar(255);not null"`
	Title       string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:varchar(255);"`
	PlayURL     string    `gorm:"type:varchar(255);not null"`
	CoverURL    string    `gorm:"type:varchar(255);not null"`
	CreateTime  time.Time `gorm:"autoCreateTime"`
	LikesCount  int64     `gorm:"column:likes_count;not null;default:0" json:"likes_count"`
}

type PublishVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PlayURL     string `json:"play_url"`
	CoverURL    string `json:"cover_url"`
}

type ListByAuthorIDRequest struct {
	AuthorID uint `json:"author_id"`
}

type GetDetailRequest struct {
	ID uint `json:"id"`
}

type UpdateLikesCountRequest struct {
	ID         uint  `json:"id"`
	LikesCount int64 `json:"likes_count"`
}
