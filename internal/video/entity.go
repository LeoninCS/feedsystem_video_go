package video

import "time"

type Video struct {
	ID          uint      `gorm:"primaryKey"`
	AuthorID    uint      `gorm:"index;not null"`
	Title       string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:varchar(255);"`
	PlayURL     string    `gorm:"type:varchar(255);not null"`
	CreateTime  time.Time `gorm:"autoCreateTime"`
}
