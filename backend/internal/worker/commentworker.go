package worker

import (
	"context"
	"encoding/json"
	"errors"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	"feedsystem_video_go/internal/video"
	"log"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

type CommentWorker struct {
	ch       *amqp.Channel
	comments *video.CommentRepository
	videos   *video.VideoRepository
	queue    string
}

func NewCommentWorker(ch *amqp.Channel, comments *video.CommentRepository, videos *video.VideoRepository, queue string) *CommentWorker {
	return &CommentWorker{ch: ch, comments: comments, videos: videos, queue: queue}
}

func (w *CommentWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.comments == nil || w.videos == nil {
		return errors.New("comment worker is not initialized")
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

func (w *CommentWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	if err := w.process(ctx, d.Body); err != nil {
		retryCount := rabbitmq.GetRetryCount(d)
		if retryCount >= rabbitmq.MaxRetryCount {
			log.Printf("comment worker: max retries exceeded (%d), moving to DLX: %v", retryCount, err)
			_ = d.Ack(false)
			return
		}
		log.Printf("comment worker: failed (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func (w *CommentWorker) process(ctx context.Context, body []byte) error {
	var evt rabbitmq.CommentEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return nil
	}
	switch evt.Action {
	case "publish":
		return w.applyPublish(ctx, &evt)
	case "delete":
		return w.applyDelete(ctx, &evt)
	default:
		return nil
	}
}

func (w *CommentWorker) applyPublish(ctx context.Context, evt *rabbitmq.CommentEvent) error {
	if evt == nil || evt.VideoID == 0 || evt.AuthorID == 0 || strings.TrimSpace(evt.Content) == "" {
		return nil
	}

	ok, err := w.videos.IsExist(ctx, evt.VideoID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	c := &video.Comment{
		Username: strings.TrimSpace(evt.Username),
		VideoID:  evt.VideoID,
		AuthorID: evt.AuthorID,
		Content:  strings.TrimSpace(evt.Content),
	}
	if err := w.comments.CreateComment(ctx, c); err != nil {
		return err
	}
	return w.videos.ChangePopularity(ctx, evt.VideoID, 1)
}

func (w *CommentWorker) applyDelete(ctx context.Context, evt *rabbitmq.CommentEvent) error {
	if evt == nil || evt.CommentID == 0 {
		return nil
	}
	c, err := w.comments.GetByID(ctx, evt.CommentID)
	if err != nil {
		return err
	}
	if c == nil {
		return nil
	}
	return w.comments.DeleteComment(ctx, c)
}

