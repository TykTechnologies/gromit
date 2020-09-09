package keys

import (
	"context"
	"time"

	"os"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

type redisHook struct{}

var _ redis.Hook = redisHook{}

func (redisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	log.Trace().Interface("cmd", cmd).Msgf("starting processing")
	return ctx, nil
}

func (redisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	log.Trace().Interface("cmd", cmd).Msgf("finished processing")
	return nil
}

func (redisHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	log.Trace().Msgf("pipeline starting: %v", cmds)
	return ctx, nil
}

func (redisHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	log.Trace().Msgf("pipeline finished: %v", cmds)
	return nil
}

const (
	getKeysTimeout = 10 * time.Second
)

type redisKey struct {
	Name  string                 `json:"name"`
	TTL   int                    `json:"ttl"`
	Value map[string]interface{} `json:"value"`
}

func NewUniversalClient(addrs []string) *redis.UniversalClient {
	password := os.Getenv("REDISCLI_AUTH")
	var opts = redis.UniversalOptions{
		MaxRetries:   3,
		PoolSize:     3,
		MinIdleConns: 1,
		PoolTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	if len(password) > 1 {
		// Cluster
		opts.Password = password
	}

	rdb := redis.NewUniversalClient(opts)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)

	pingErr := rdb.ForEachShard(pingCtx, func(ctx context.Context, client *redis.Client) error {
		return client.Ping(ctx).Err()
	})

	if pingErr != nil {
		pingCancel()
		log.Fatal().Array("addrs", addrs).Err(pingErr).Msg("could not ping")
	}

	return &rdb
}

func getKeys(rdb *redis.UniversalClient, pattern string) ([]redisKey, error) {
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), getKeysTimeout)

}
