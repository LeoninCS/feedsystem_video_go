package worker

import (
	"context"
	"encoding/json"
	"errors"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RecipientID uint    `gorm:"index;not null" json:"recipient_id"`
	SenderID  uint      `gorm:"not null" json:"sender_id"`
	Type      string    `gorm:"type:varchar(50);not null" json:"type"`
	TargetID  uint      `json:"target_id"`
	Content   string    `gorm:"type:varchar(255)" json:"content"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type NotificationWorker struct {
	ch      *amqp.Channel
	db      *gorm.DB
	queue   string
	hub     NotificationHub
}

type NotificationHub interface {
	Push(userID uint, n *Notification)
}

func NewNotificationWorker(ch *amqp.Channel, db *gorm.DB, queue string, hub NotificationHub) *NotificationWorker {
	return &NotificationWorker{ch: ch, db: db, queue: queue, hub: hub}
}

func (w *NotificationWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.db == nil {
		return errors.New("notification worker is not initialized")
	}
	if err := w.db.WithContext(ctx).AutoMigrate(&Notification{}); err != nil {
		return err
	}
	deliveries, err := w.ch.Consume(w.queue, "", false, false, false, false, nil)
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

func (w *NotificationWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	retryCount := rabbitmq.GetRetryCount(d)
	if err := w.process(ctx, d); err != nil {
		if retryCount >= rabbitmq.MaxRetryCount {
			log.Printf("notification worker: max retries, dropping: %v", err)
			_ = d.Ack(false)
			return
		}
		log.Printf("notification worker: failed (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func (w *NotificationWorker) process(ctx context.Context, d amqp.Delivery) error {
	body := d.Body
	if len(body) == 0 {
		return nil
	}
	routingKey := d.RoutingKey

	var notif *Notification

	switch {
	case routingKey == "like.like":
		var evt rabbitmq.LikeEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return nil
		}
		if evt.UserID == 0 || evt.VideoID == 0 {
			return nil
		}
		var authorID uint
		w.db.WithContext(ctx).Model(&struct{ ID uint; AuthorID uint }{}).Table("videos").Where("id = ?", evt.VideoID).Select("author_id").Scan(&authorID)
		if authorID == 0 || authorID == evt.UserID {
			return nil
		}
		notif = &Notification{RecipientID: authorID, SenderID: evt.UserID, Type: "like", TargetID: evt.VideoID, Content: "点赞了你的视频"}

	case routingKey == "comment.publish":
		var evt rabbitmq.CommentEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return nil
		}
		if evt.AuthorID == 0 || evt.VideoID == 0 {
			return nil
		}
		var authorID uint
		w.db.WithContext(ctx).Model(&struct{ ID uint; AuthorID uint }{}).Table("videos").Where("id = ?", evt.VideoID).Select("author_id").Scan(&authorID)
		if authorID == 0 || authorID == evt.AuthorID {
			return nil
		}
		notif = &Notification{RecipientID: authorID, SenderID: evt.AuthorID, Type: "comment", TargetID: evt.VideoID, Content: "评论了你的视频"}

	case routingKey == "social.follow":
		var evt rabbitmq.SocialEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return nil
		}
		if evt.FollowerID == 0 || evt.VloggerID == 0 {
			return nil
		}
		notif = &Notification{RecipientID: evt.VloggerID, SenderID: evt.FollowerID, Type: "follow", TargetID: evt.FollowerID, Content: "关注了你"}
	}

	if notif == nil {
		return nil
	}
	if err := w.db.WithContext(ctx).Create(notif).Error; err != nil {
		return err
	}
	if w.hub != nil {
		w.hub.Push(notif.RecipientID, notif)
	}
	return nil
}
