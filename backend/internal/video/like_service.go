package video

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type LikeService struct {
	repo      *LikeRepository
	VideoRepo *VideoRepository
}

func NewLikeService(repo *LikeRepository, videoRepo *VideoRepository) *LikeService {
	return &LikeService{repo: repo, VideoRepo: videoRepo}
}

func isDupKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func (s *LikeService) Like(ctx context.Context, like *Like) error {
	like.CreatedAt = time.Now()
	return s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&Video{}, like.VideoID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("video not found")
			}
			return err
		}
		if err := tx.Create(like).Error; err != nil {
			if isDupKey(err) {
				return errors.New("user has liked this video")
			}
			return err
		}
		if err := tx.Model(&Video{}).Where("id = ?", like.VideoID).
			UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *LikeService) Unlike(ctx context.Context, like *Like) error {
	return s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		del := tx.Where("video_id = ? AND account_id = ?", like.VideoID, like.AccountID).Delete(&Like{})
		if del.Error != nil {
			return del.Error
		}
		if del.RowsAffected == 0 {
			return errors.New("user has not liked this video")
		}

		return tx.Model(&Video{}).Where("id = ?", like.VideoID).
			UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error
	})
}

func (s *LikeService) IsLiked(ctx context.Context, videoID, accountID uint) (bool, error) {
	return s.repo.IsLiked(ctx, videoID, accountID)
}
