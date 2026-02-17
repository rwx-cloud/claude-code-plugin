package main

import (
	"context"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisConnection(t *testing.T) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		t.Skip("REDIS_URL not set")
	}

	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to ping redis: %v", err)
	}
}
