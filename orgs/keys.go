package orgs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"os"

	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	rdb       redis.UniversalClient
	batchSize int64
	ctx       context.Context
	keysChan  chan ([]string)
	foundChan chan (int)
}

func RedisClient(ctx context.Context, addrs []string, masterName string, batchSize int64) redisClient {
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

	rdb.Ping(ctx)

	return redisClient{
		rdb,
		batchSize,
		ctx,
		make(chan ([]string), batchSize),
		make(chan (int)),
	}
}

func (r *redisClient) WriteKeys(orgs []string, patterns []string) {
	for _, org := range orgs {
		log.Info().Str("org", org).Msg("processing keys")
		fname := fmt.Sprintf("%s.keys.jl", org)
		log.Info().Str("fname", fname).Msg("keys will be written to")
		f, err := os.Create(fname)
		if err != nil {
			log.Fatal().Err(err).Str("fname", fname).Msg("cannot create")
		}
		defer f.Close()
		w := bufio.NewWriter(f)

		// Scan keys in batches and write to r.keysChan
		go func() {
			scanned := 0
			for _, pattern := range patterns {
				scanned += r.ScanKeys(pattern, r.batchSize)
			}
			close(r.keysChan)
			log.Info().Int("scanned", scanned).Msg("total")
		}()

		var wg sync.WaitGroup
		// Read in batches and fire off a goroutine for each batch
		for keys := range r.keysChan {
			log.Debug().Int("keys", len(keys)).Msg("to filter")
			go func() {
				wg.Add(1)
				r.FilterOrg(org, keys, w)
				wg.Done()
			}()
		}
		// Wait for all the writers to finish
		go func() {
			wg.Wait()
			close(r.foundChan)
			w.Flush()
		}()
	}
	found := 0
	for f := range r.foundChan {
		found += f
	}
	log.Info().Int("found", found).Msg("written")
}

// FilterOrg will MGET the block of keys and write out all those belonging to org
func (r *redisClient) FilterOrg(org string, keys []string, w io.Writer) {
	found := 0
	values, err := r.rdb.MGet(r.ctx, keys...).Result()
	if err != nil {
		log.Error().Err(err).Msg("mget")
	}
	log.Debug().Int("keys", len(values)).Msg("mget")
	for i, val := range values {
		if val == nil {
			log.Trace().Str("key", keys[i]).Msg("nil value")
			continue
		}
		jsonVal := make(map[string]interface{})
		err = json.Unmarshal([]byte(val.(string)), &jsonVal)
		if err != nil {
			log.Error().Err(err).Interface("val", val).Msg("cannot decode")
			continue
		}
		if jsonVal["org_id"] == org {
			found++
			ttl, _ := r.getTTL(keys[i])

			output, err := json.Marshal(&redisKey{
				Name:  keys[i],
				TTL:   ttl,
				Value: jsonVal,
			})
			if err != nil {
				log.Error().Err(err).Bytes("output", output).Msg("could not marshal")
				continue
			}
			w.Write(output)
			// Write a new line
			w.Write([]byte{10})
		}
	}
	r.foundChan <- found
}

func (r *redisClient) ScanKeys(pattern string, batchSize int64) int {
	scanned := 0
	for {
		var cursor uint64 = 0
		keys, cursor, err := r.rdb.Scan(r.ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			log.Error().Err(err).Str("pattern", pattern).Msg("scan failure")
		}
		nKeys := len(keys)
		log.Debug().Int("keys", nKeys).Msg("scanned in this block")
		scanned += nKeys
		r.keysChan <- keys

		if cursor == 0 {
			break
		}
	}
	return scanned
}

func (r *redisClient) getTTL(keyName string) (ttl int64, err error) {
	duration, err := r.rdb.TTL(r.ctx, keyName).Result()
	return int64(duration.Seconds()), err
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
