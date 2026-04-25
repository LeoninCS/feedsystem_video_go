package worker

import (
	"context"
	"encoding/json"
	"errors"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	"feedsystem_video_go/internal/video"
	"log"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type LikeWorker struct {
	ch    *amqp.Channel
	likes  *video.LikeRepository
	videos *video.VideoRepository
	queue string
}

func NewLikeWorker(ch *amqp.Channel, likes *video.LikeRepository, videos *video.VideoRepository, queue string) *LikeWorker {
	return &LikeWorker{ch: ch, likes: likes, videos: videos, queue: queue}
}

func (w *LikeWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.likes == nil || w.videos == nil {
		return errors.New("like worker is not initialized")
	}
	if w.queue == "" {
		return errors.New("queue is required")
	}

	deliveries, err := w.ch.Consume(
		w.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("deliveries channel closed")
			}
			w.handleDelivery(ctx, d)
		}
	}
}

func (w *LikeWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	if err := w.process(ctx, d.Body); err != nil {
		retryCount := rabbitmq.GetRetryCount(d)
		if retryCount >= rabbitmq.MaxRetryCount {
			log.Printf("like worker: max retries exceeded (%d), moving to DLX: %v", retryCount, err)
			_ = d.Ack(false)
			return
		}
		log.Printf("like worker: failed (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func (w *LikeWorker) process(ctx context.Context, body []byte) error {
	var evt rabbitmq.LikeEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		// 解析事件失败，直接丢弃
		return nil
	}
	if evt.UserID == 0 || evt.VideoID == 0 {
		return nil
	}

	switch evt.Action {
	case "like":
		return w.applyLike(ctx, evt.UserID, evt.VideoID)
	case "unlike":
		return w.applyUnlike(ctx, evt.UserID, evt.VideoID)
	default:
		return nil
	}
}

func (w *LikeWorker) applyLike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	created, err := w.likes.LikeIgnoreDuplicate(ctx, &video.Like{
		VideoID:   videoID,
		AccountID: userID,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	if err := w.videos.ChangeLikesCount(ctx, videoID, 1); err != nil {
		return err
	}
	return w.videos.ChangePopularity(ctx, videoID, 1)
}

func (w *LikeWorker) applyUnlike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	deleted, err := w.likes.DeleteByVideoAndAccount(ctx, videoID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return nil
	}

	if err := w.videos.ChangeLikesCount(ctx, videoID, -1); err != nil {
		return err
	}
	return w.videos.ChangePopularity(ctx, videoID, -1)
}
