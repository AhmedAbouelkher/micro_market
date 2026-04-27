package main

import (
	"context"
	"fmt"
	"micro_market/common"

	"github.com/redis/go-redis/v9"
)

var (
	redisHost    = common.EnvOrDef("REDIS_HOST", "localhost")
	redisPort    = common.EnvOrDef("REDIS_PORT", "6379")
	RedisChannel = common.EnvOrDef("REDIS_CHANNEL", "create_invoice")

	redisClient *redis.Client
)

func InitRedisDB(ctx context.Context) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return err
	}

	if err := telemetry.UseRedisPlugin(redisClient); err != nil {
		return err
	}

	return nil
}

func CloseRedisDB() error {
	return redisClient.Close()
}
