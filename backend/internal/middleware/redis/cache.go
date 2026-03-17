package redis

import (
	"context"
	"time"
)

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) MGet(cacheCtx context.Context, cacheKeys ...string) ([]interface{}, error) {
	return c.rdb.MGet(cacheCtx, cacheKeys...).Result()
}
