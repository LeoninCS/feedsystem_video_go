package redis

import (
	"context"
	"os"
	"strconv"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewFromEnv() (*Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		db = n
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func IsMiss(err error) bool {
	return err == redis.Nil
}
