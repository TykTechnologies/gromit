package keys

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
)

func streamKeys(filepath string, keyChan chan<- []redisKey) {
	file, openErr := os.Open(filepath)
	if openErr != nil {
		panic(openErr)
	}

	defer file.Close()

	redisKeys := make([]redisKey, 0, batchSize)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key := redisKey{}

		if err := json.Unmarshal(scanner.Bytes(), &key); err != nil {
			log.Error().Err(err).Msg("error unmarshalling redis key")
			continue
		}

		redisKeys = append(redisKeys, key)

		if len(redisKeys) == batchSize {
			keyChan <- redisKeys
			redisKeys = make([]redisKey, 0, batchSize)
		}
	}

	if len(redisKeys) > 0 {
		keyChan <- redisKeys
	}

	close(keyChan)
}

func writeKeysToRedis(ctx context.Context, rdb *redis.UniversalClient, keys []redisKey) error {
	_, err := rdb.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, key := range keys {
			ttl := time.Duration(0)
			if key.Ttl > 0 {
				ttl = time.Duration(key.Ttl) * time.Second
			}

			keyValue, jsonErr := json.Marshal(key.Value)
			if jsonErr != nil {
				return jsonErr
			}

			pipeliner.Set(ctx, key.Name, keyValue, ttl)
		}

		return nil
	})

	return err
}

func RestoreKeys(rdb *redis.UniversalClient, dumpFile string) {
	keyChan := make(chan []redisKey, 2)

	go streamKeys(dumpFile, keyChan)

	nWrote := 0

	for redisKeys := range keyChan {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)

		if err := writeKeysToRedis(writeCtx, rdb, redisKeys); err != nil {
			writeCancel()
			log.Fatal().Err(err).Msg("could not write to redis")
		}

		nWrote += len(redisKeys)
		log.Info().Int("keys", nWrote).Msg("wrote to redis")
	}
}
