package main

import (
	"context"
	"feedsystem_video_go/internal/config"
	"feedsystem_video_go/internal/db"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/observability"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/video"
	"feedsystem_video_go/internal/worker"
	"log"
	
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	socialExchange   = "social.events"
	socialQueue      = "social.events"
	socialBindingKey = "social.*"

	likeExchange   = "like.events"
	likeQueue      = "like.events"
	likeBindingKey = "like.*"

	commentExchange   = "comment.events"
	commentQueue      = "comment.events"
	commentBindingKey = "comment.*"

	popularityExchange   = "video.popularity.events"
	popularityQueue      = "video.popularity.events"
	popularityBindingKey = "video.popularity.*"
)

func main() {
	// 加载配置
	const configPath = "configs/config.yaml"
	log.Printf("Loading config from %s", configPath)
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("Config File %s not found, using default local config", configPath)
	} else {
		log.Printf("Config loaded from file: %s", configPath)
	}
	// 连接数据库
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	defer db.CloseDB(sqlDB)

	// 连接 Redis（用于流行度更新）
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		log.Printf("Redis config error (popularity worker disabled): %v", err)
		cache = nil
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := cache.Ping(pingCtx); err != nil {
			log.Printf("Redis not available (popularity worker disabled): %v", err)
			_ = cache.Close()
			cache = nil
		} else {
			defer cache.Close()
			log.Printf("Redis connected (popularity worker enabled)")
		}
	}
	// 连接 RabbitMQ
	url := "amqp://" + cfg.RabbitMQ.Username + ":" + cfg.RabbitMQ.Password + "@" + cfg.RabbitMQ.Host + ":" + strconv.Itoa(cfg.RabbitMQ.Port) + "/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect rabbitmq: %v", err)
	}
	defer conn.Close()
	// 创建 RabbitMQ 通道
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open rabbitmq channel: %v", err)
	}
	defer ch.Close()
	// 声明 Social 交换机和队列
	if err := declareSocialTopology(ch); err != nil {
		log.Fatalf("Failed to declare social topology: %v", err)
	}
	if err := declareLikeTopology(ch); err != nil {
		log.Fatalf("Failed to declare like topology: %v", err)
	}
	if err := declareCommentTopology(ch); err != nil {
		log.Fatalf("Failed to declare comment topology: %v", err)
	}
	if cache != nil {
		if err := declarePopularityTopology(ch); err != nil {
			log.Fatalf("Failed to declare popularity topology: %v", err)
		}
	}
	if err := ch.Qos(50, 0, false); err != nil {
		log.Fatalf("Failed to set qos: %v", err)
	}

	repo := social.NewSocialRepository(sqlDB)
	socialWorker := worker.NewSocialWorker(ch, repo, socialQueue)
	videoRepo := video.NewVideoRepository(sqlDB)
	likeRepo := video.NewLikeRepository(sqlDB)
	commentRepo := video.NewCommentRepository(sqlDB)
	likeWorker := worker.NewLikeWorker(ch, likeRepo, videoRepo, likeQueue)
	commentWorker := worker.NewCommentWorker(ch, commentRepo, videoRepo, commentQueue)
	var popularityWorker *worker.PopularityWorker
	if cache != nil {
		popularityWorker = worker.NewPopularityWorker(ch, cache, popularityQueue)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pprofServer, err := observability.NewPprofServer(
		"Worker",
		cfg.ObservabilityConfig.Pprof.Enabled,
		cfg.ObservabilityConfig.Pprof.WorkerAddr,
	)
	if err != nil {
		log.Printf("Failed to start worker pprof server: %v", err)
	}
	defer pprofServer.Close()

	errCh := make(chan error, 4)
	log.Printf("Worker started, consuming queue=%s", socialQueue)
	go func() { errCh <- socialWorker.Run(ctx) }()
	log.Printf("Worker started, consuming queue=%s", likeQueue)
	go func() { errCh <- likeWorker.Run(ctx) }()
	log.Printf("Worker started, consuming queue=%s", commentQueue)
	go func() { errCh <- commentWorker.Run(ctx) }()
	if popularityWorker != nil {
		log.Printf("Worker started, consuming queue=%s", popularityQueue)
		go func() { errCh <- popularityWorker.Run(ctx) }()
	}

	err = <-errCh
	if err != nil && err != context.Canceled {
		log.Fatalf("Worker stopped: %v", err)
	}
	log.Printf("Worker stopped")
}

func declareSocialTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		socialExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		socialQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	if err := ch.QueueBind(
		q.Name,
		socialBindingKey,
		socialExchange,
		false,
		nil,
	); err != nil {
		return err
	}
	return nil
}

func declarePopularityTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		popularityExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		popularityQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.QueueBind(
		q.Name,
		popularityBindingKey,
		popularityExchange,
		false,
		nil,
	)
}

func declareLikeTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		likeExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		likeQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.QueueBind(
		q.Name,
		likeBindingKey,
		likeExchange,
		false,
		nil,
	)
}

func declareCommentTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commentExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		commentQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.QueueBind(
		q.Name,
		commentBindingKey,
		commentExchange,
		false,
		nil,
	)
}
