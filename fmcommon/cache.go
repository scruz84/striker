package fmcommon

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type Cache interface {
	GetValue(key string) (string, error)
	SetValue(key string, value string)
}

func NewRedisCache() *RedisCache {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
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
	r.client.SetNX(*r.ctx, key, value, time.Minute)
}
