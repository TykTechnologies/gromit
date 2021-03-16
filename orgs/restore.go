package orgs

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	batchSize        = 100
	writeKeysTimeout = 10 * time.Second
)

func streamKeys(filepath string, keyChan chan<- []redisKey, batchSize int) {
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

func (r *RedisClient) writeKeys(ctx context.Context, keys []redisKey) error {
	// _, err := r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
	// 	for _, key := range keys {
	// 		ttl := time.Duration(0)
	// 		if key.TTL > 0 {
	// 			ttl = time.Duration(key.TTL) * time.Second
	// 		}

	// 		keyValue, jsonErr := json.Marshal(key.Value)
	// 		if jsonErr != nil {
	// 			return jsonErr
	// 		}

	// 		pipe.Set(key.Name, keyValue, ttl)
	// 	}

	// 	return nil
	// })

	// return err
	return nil
}

func (r *RedisClient) RestoreKeys(dumpFile string) {
	keyChan := make(chan []redisKey, 2)

	go streamKeys(dumpFile, keyChan, batchSize)

	nWrote := 0

	for redisKeys := range keyChan {
		writeCtx, writeCancel := context.WithTimeout(context.Background(), writeKeysTimeout)

		if err := r.writeKeys(writeCtx, redisKeys); err != nil {
			writeCancel()
			log.Fatal().Err(err).Msg("could not write to redis")
		}

		nWrote += len(redisKeys)
		log.Info().Int("keys", nWrote).Msg("wrote to redis")
	}
}
