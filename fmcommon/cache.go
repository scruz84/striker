package fmcommon

import (
	"context"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
	"time"
)

type Cache interface {
	GetValue(key string) (string, error)
	SetValue(key string, value string)
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
	r.client.SetNX(*r.ctx, key, value, time.Minute)
}
