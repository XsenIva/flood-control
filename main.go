package main

import (
	"context"
	stv "strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	N      time.Duration = 10
	K      int           = 7
	mute   sync.Mutex
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
)

func main() {
}

type FloodControl interface {
	Check(ctx context.Context, userID int64) (bool, error)
}

func nil_check(err error) {
	if err != nil {
		panic(err)
	}
}

func Set(ctx context.Context, key string, val any) error {
	err := client.Set(ctx, key, val, 0).Err()
	nil_check(err)
	return err
}

func Get(ctx context.Context, key string) (int, error) {
	val_str, err := client.Get(ctx, key).Result()
	val_int, _ := stv.Atoi(val_str)
	return (val_int), err
}

func find_lim(ctx context.Context, userID int64) (bool, error) {
	iter := client.Scan(ctx, 0, "prefix:*", 0).Iterator()
	for iter.Next(ctx) {
		val, err := stv.Atoi(iter.Val())
		if val > K {
			Set(ctx, string(userID), "false")
			return false, err
		}
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
	return true, nil
}

func Check(ctx context.Context, userID int64) (bool, error) {
	ctx_lim, cancel := context.WithTimeout(ctx, N*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx_lim.Done():
			mute.Lock()
			ans, err := find_lim(ctx, userID)
			mute.Unlock()
			return ans, err
		default:
		}
		userID_s := string(userID)
		val, err := Get(ctx, string(userID_s))
		if err == redis.Nil {
			Set(ctx, userID_s, 0)
		} else {
			Set(ctx, userID_s, val+1)
		}
	}
}
