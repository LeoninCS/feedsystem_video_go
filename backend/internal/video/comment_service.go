package video

import (
	"context"
	"errors"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/apierror"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type CommentService struct {
	repo            *CommentRepository
	VideoRepository *VideoRepository
	cache           *rediscache.Client
	commentMQ       *rabbitmq.CommentMQ
	popularityMQ    *rabbitmq.PopularityMQ
}

func NewCommentService(repo *CommentRepository, videoRepo *VideoRepository, cache *rediscache.Client, commentMQ *rabbitmq.CommentMQ, popularityMQ *rabbitmq.PopularityMQ) *CommentService {
	return &CommentService{repo: repo, VideoRepository: videoRepo, cache: cache, commentMQ: commentMQ, popularityMQ: popularityMQ}
}

func (s *CommentService) Publish(ctx context.Context, comment *Comment) error {
	if comment == nil {
		return errors.New("comment is nil")
	}
	comment.Username = strings.TrimSpace(comment.Username)
	comment.Content = strings.TrimSpace(comment.Content)
	if comment.VideoID == 0 || comment.AuthorID == 0 {
		return errors.New("video_id and author_id are required")
	}
	if comment.Content == "" {
		return errors.New("content is required")
	}

	exists, err := s.VideoRepository.IsExist(ctx, comment.VideoID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("video not found")
	}

	mysqlEnqueued := false
	redisEnqueued := false
	if s.commentMQ != nil {
		if err := s.commentMQ.Publish(ctx, comment.Username, comment.VideoID, comment.AuthorID, comment.Content); err == nil {
			mysqlEnqueued = true
		}
	}
	if s.popularityMQ != nil {
		if err := s.popularityMQ.Update(ctx, comment.VideoID, 1); err == nil {
			redisEnqueued = true
		}
	}
	if mysqlEnqueued && redisEnqueued {
		s.notifyMentions(ctx, comment)
		return nil
	}

	// Fallback: direct MySQL write when comment MQ publish fails.
	if !mysqlEnqueued {
		if err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Select("id").First(&Video{}, comment.VideoID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("video not found")
				}
				return err
			}
			if err := tx.Create(comment).Error; err != nil {
				return err
			}
			return tx.Model(&Video{}).Where("id = ?", comment.VideoID).
				UpdateColumn("popularity", gorm.Expr("popularity + 1")).Error
		}); err != nil {
			return err
		}
	}

	// Fallback: direct Redis update when popularity MQ publish fails.
	if !redisEnqueued {
		UpdatePopularityCache(ctx, s.cache, comment.VideoID, 1)
	}
	s.notifyMentions(ctx, comment)
	return nil
}

func (s *CommentService) Delete(ctx context.Context, commentID uint, accountID uint) error {
	comment, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("comment not found")
	}
	if comment.AuthorID != accountID {
		return apierror.ErrUnauthorized
	}
	if s.commentMQ != nil {
		if err := s.commentMQ.Delete(ctx, commentID); err == nil {
			return nil
		}
	}
	return s.repo.DeleteComment(ctx, comment)
}

func (s *CommentService) GetAll(ctx context.Context, videoID uint) ([]Comment, error) {
	exists, err := s.VideoRepository.IsExist(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("video not found")
	}
	return s.repo.GetAllComments(ctx, videoID)
}

var mentionRegex = regexp.MustCompile(`@(\w+)`)

func (s *CommentService) notifyMentions(ctx context.Context, comment *Comment) {
	matches := mentionRegex.FindAllStringSubmatch(comment.Content, -1)
	if len(matches) == 0 {
		return
	}
	seen := make(map[string]bool)
	for _, m := range matches {
		username := m[1]
		if seen[username] || username == comment.Username {
			continue
		}
		seen[username] = true
		var accID uint
		if err := s.repo.db.WithContext(ctx).Table("accounts").Where("username = ?", username).Select("id").Scan(&accID).Error; err != nil || accID == 0 {
			continue
		}
		notif := struct {
			RecipientID uint
			SenderID    uint
			Type        string
			TargetID    uint
			Content     string
		}{
			RecipientID: accID,
			SenderID:    comment.AuthorID,
			Type:        "mention",
			TargetID:    comment.VideoID,
			Content:     comment.Username + " 在评论中提到了你",
		}
		s.repo.db.WithContext(ctx).Table("notifications").Create(&notif)
	}
}
