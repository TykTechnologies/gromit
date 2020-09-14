package orgs

import (
	"context"
	"time"

	"os"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
)

const (
	getKeysTimeout = 10 * time.Second
)

type redisKey struct {
	Name  string                 `json:"name"`
	TTL   int64                  `json:"ttl"`
	Value map[string]interface{} `json:"value"`
}

type redisClient struct {
	rdb redis.UniversalClient
}

// A simple hack to work around lack of generics in logging an array to zerolog
// Depends on zerolog not allocating anything
func logArray(strs []string) *zerolog.Array {
	var array zerolog.Array
	for _, s := range strs {
		array.Str(s)
	}
	return &array
}

func RedisClient(addrs []string, masterName string) redisClient {
	var rdb redis.UniversalClient
	rdb = redis.NewUniversalClient(&redis.UniversalOptions{
		MaxRetries:   3,
		PoolSize:     3,
		MinIdleConns: 1,
		ReadTimeout:  getKeysTimeout,
		WriteTimeout: getKeysTimeout,
		PoolTimeout:  2 * getKeysTimeout,
		IdleTimeout:  getKeysTimeout,
		Password:     os.Getenv("REDISCLI_AUTH"),
		MasterName:   masterName,
		Addrs:        addrs,
	})

	pingCtx, _ := context.WithTimeout(context.Background(), getKeysTimeout)
	rdb.Ping(pingCtx)

	return redisClient{rdb}
}
