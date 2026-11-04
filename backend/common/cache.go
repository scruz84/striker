package common

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
	"strconv"
	"time"
)

type Cache interface {
	GetValue(key string) (string, error)
	SetValue(key string, value string)
	SetValueLongDuration(key string, value string)
	// SetValueNoExpiry stores a value that never changes once written (a past season's table).
	SetValueNoExpiry(key string, value string)
	DeleteValue(key string)
}

func NewRedisCache() *RedisCache {
	ctx := context.Background()
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	pwd := os.Getenv("REDIS_PASSWORD")
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: pwd,
		DB:       db,
	})
	return &RedisCache{client: rdb, ctx: &ctx}
}

type RedisCache struct {
	client *redis.Client
	ctx    *context.Context
}

func (r RedisCache) GetValue(key string) (string, error) {
	return r.client.Get(*r.ctx, key).Result()
}

func (r RedisCache) SetValue(key string, value string) {
	r.client.Set(*r.ctx, key, value, time.Minute)
}

func (r RedisCache) SetValueLongDuration(key string, value string) {
	r.client.Set(*r.ctx, key, value, time.Minute*30)
}

func (r RedisCache) SetValueNoExpiry(key string, value string) {
	r.client.Set(*r.ctx, key, value, 0)
}

func (r RedisCache) DeleteValue(key string) {
	r.client.Del(*r.ctx, key)
}
