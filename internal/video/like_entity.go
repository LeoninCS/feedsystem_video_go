package video

import "time"

type Like struct {
	ID        uint `gorm:"primaryKey"`
	VideoID   uint `gorm:"index, not null"`
	AccountID uint `gorm:"index, not null"`
	CreatedAt time.Time
}
