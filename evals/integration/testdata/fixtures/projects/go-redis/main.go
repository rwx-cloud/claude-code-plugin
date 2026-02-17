package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected to redis")
}
