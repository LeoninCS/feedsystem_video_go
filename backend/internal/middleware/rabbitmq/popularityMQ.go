package rabbitmq

import (
	"context"
	"errors"
	"time"
)

type PopularityMQ struct {
	*RabbitMQ
}

const (
	popularityExchange   = "video.popularity.events"
	popularityQueue      = "video.popularity.events"
	popularityBindingKey = "video.popularity.*"

	popularityUpdateRK = "video.popularity.update"
)

type PopularityEvent struct {
	EventID    string    `json:"event_id"`
	VideoID    uint      `json:"video_id"`
	Change     int64     `json:"change"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewPopularityMQ(base *RabbitMQ) (*PopularityMQ, error) {
	if base == nil {
		return nil, errors.New("rabbitmq base is nil")
	}
	if err := base.DeclareTopic(popularityExchange, popularityQueue, popularityBindingKey); err != nil {
		return nil, err
	}
	return &PopularityMQ{RabbitMQ: base}, nil
}

func (p *PopularityMQ) Update(ctx context.Context, videoID uint, change int64) error {
	if p == nil || p.RabbitMQ == nil {
		return errors.New("popularity mq is not initialized")
	}
	if videoID == 0 || change == 0 {
		return errors.New("videoID and change are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	event := PopularityEvent{
		EventID:    id,
		VideoID:    videoID,
		Change:     change,
		OccurredAt: time.Now().UTC(),
	}
	return p.PublishJSON(ctx, popularityExchange, popularityUpdateRK, event)
}

