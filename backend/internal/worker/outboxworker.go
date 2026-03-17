package worker

import (
	"context"
	"encoding/json"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	"feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/video"
	"fmt"
	"log"
	"time"

	oredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func StartOutboxPoller(db *gorm.DB, tmq *rabbitmq.TimelineMQ) {
	go func() {
		for {
			var messages []video.OutboxMsg

			err := db.Where("status = ?", "pending").Order("create_time ASC").Limit(100).Find(&messages).Error

			if err != nil || len(messages) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}

			for _, msg := range messages {
				err := tmq.PublishVideo(context.Background(), msg.VideoID, msg.CreateTime)

				if err == nil {
					db.Delete(&msg)
				} else {
					log.Printf("投递MQ失败: VideoID: %d, err: %v", msg.VideoID, err)
				}
			}
		}
	}()
}

func StartConsumer(tmq *rabbitmq.TimelineMQ, queueName string, redisClient *redis.Client) {
	msgs, err := tmq.Ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Printf("注册消费失败")
		return
	}

	go func() {
		for msg := range msgs {
			var event rabbitmq.TimelineEvent
			err := json.Unmarshal(msg.Body, &event)

			if err != nil {
				log.Printf("反序列化失败")
				msg.Ack(false)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			timelineKey := "feed:global_timeline"
			err = redisClient.ZAdd(ctx, timelineKey, oredis.Z{
				Score:  float64(event.CreateTime),
				Member: fmt.Sprintf("%d", event.VideoID),
			})

			if err != nil {
				log.Printf("写入Zset失败")
				msg.Nack(false, true)
				cancel()
				continue
			}

			err = redisClient.ZRemRangeByRank(ctx, timelineKey, 0, -1001)

			if err != nil {
				log.Printf("ZRem失败")
			}

			msg.Ack(false)
			cancel()
		}
	}()
}
