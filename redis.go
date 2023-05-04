package bufrex

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	ctx      context.Context
	client   *redis.Client
	onDecode func(string, string) (interface{}, error)
	onEncode func(string, interface{}) (string, error)
}

// Redis adapter for second level caching
func SetupRedis(opts *redis.Options, encoder func(string, interface{}) (string, error), decoder func(string, string) (interface{}, error)) *RedisAdapter {
	ctx := context.Background()
	client := redis.NewClient(opts)
	// ping & confirm connection activity
	if pong, err := client.Ping(ctx).Result(); err != nil || pong != "PONG" {
		panic(pong)
	} else {
		return &RedisAdapter{
			ctx:      ctx,
			client:   client,
			onEncode: encoder,
			onDecode: decoder,
		}
	}
}

func (redix *RedisAdapter) SetToRedis(key string, value interface{}, expiry time.Duration) {
	// serialize interface{} to a string before storage
	if value, err := redix.onEncode(key, value); err == nil {
		redix.client.Set(redix.ctx, key, value, expiry)
	} else {
		panic(err)
	}
}

func (redix *RedisAdapter) GetFromRedis(key string) (interface{}, bool) {
	value, err := redix.client.Get(redix.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	} else {
		// if decoding fails, a panic is sent out.
		if obj, err := redix.onDecode(key, value); err == nil {
			return obj, true
		} else {
			panic(err)
		}
	}
}

func (redix *RedisAdapter) DeleteFromRedis(key string) bool {
	if count, _ := redix.client.Del(redix.ctx, key).Result(); count == 1 {
		return true
	}

	return false
}
