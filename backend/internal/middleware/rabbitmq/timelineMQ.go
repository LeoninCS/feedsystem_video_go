package rabbitmq

import (
	"context"
	"errors"
	"time"
)

type TimelineMQ struct {
	*RabbitMQ
}

const (
	timelineExchange   = "video.timeline.events"
	timelineQueue      = "video.timeline.update.queue"
	timelineBindingKey = "video.timeline.*"
	timelinePublishRK  = "video.timeline.publish"
)

type TimelineEvent struct {
	EventID    string    `json:"event_id"`
	VideoID    uint      `json:"video_id"`
	CreateTime int64     `json:"create_time"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewTimelineMQ(base *RabbitMQ) (*TimelineMQ, error) {
	if base == nil {
		return nil, errors.New("rabbitmq base is nil")
	}
	if err := base.DeclareTopic(timelineExchange, timelineQueue, timelineBindingKey); err != nil {
		return nil, err
	}
	return &TimelineMQ{RabbitMQ: base}, nil
}

func (t *TimelineMQ) PublishVideo(ctx context.Context, videoID uint, createTime time.Time) error {
	if t == nil || t.RabbitMQ == nil {
		return errors.New("timeline mq is not initialized")
	}
	if videoID == 0 {
		return errors.New("videoID are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	timeline := TimelineEvent{
		EventID:    id,
		VideoID:    videoID,
		CreateTime: createTime.UnixMilli(),
		OccurredAt: time.Now(),
	}
	return t.PublishJSON(ctx, timelineExchange, timelinePublishRK, timeline)
}
