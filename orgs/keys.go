package orgs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"os"

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
	keysChans map[string]chan ([]string)
	opFiles   map[string]string
}

func RedisClient(ctx context.Context, addrs []string, masterName string, batchSize int64, orgs []string, dir string) redisClient {
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

	keysChans := make(map[string]chan ([]string))
	opFiles := make(map[string]string)

	for _, org := range orgs {
		log.Debug().Str("org", org).Msg("setting up channels")
		keysChans[org] = make(chan ([]string), batchSize)
		opFiles[org] = filepath.Join(dir, org+".keys.jl")
	}

	return redisClient{
		rdb,
		batchSize,
		ctx,
		keysChans,
		opFiles,
	}
}

// FilterOrg will MGET the block of keys and write out all those belonging to org
func (r *redisClient) filterOrg(org string) {
	f, err := os.Create(r.opFiles[org])
	if err != nil {
		log.Fatal().Err(err).Str("opfile", r.opFiles[org]).Msg("cannot create")
	}
	defer f.Close()
	log.Info().Str("opfile", r.opFiles[org]).Msg("truncated")
	w := bufio.NewWriter(f)
	defer w.Flush()

	found := 0
	for keys := range r.keysChans[org] {
		values, err := r.rdb.MGet(r.ctx, keys...).Result()
		if err != nil {
			log.Error().Err(err).Msg("mget")
		}
		log.Debug().Int("keys", len(values)).Msg("mget")
		for i, val := range values {
			if val == nil || val == "0" {
				log.Trace().Str("key", keys[i]).Msg("nil/zero value")
				continue
			}
			jsonVal := make(map[string]interface{})
			err = json.Unmarshal([]byte(val.(string)), &jsonVal)
			if err != nil {
				log.Error().Err(err).Interface("val", val).Msg("cannot decode")
				continue
			}

			orgIdValue := ""

			var goiErr error
			if orgIdValue, goiErr = getOrgId(jsonVal); goiErr != nil {
				log.Error().Err(goiErr).Interface("val", jsonVal).
					Msg("couldn't find the org_id field, skipping")
				continue
			}

			if orgIdValue == org {
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
				// Add a newline
				w.Write(append(output, byte(10)))
			}
		}
	}
	log.Info().Int("found", found).Str("org", org).Msg("done")
}

func (r *redisClient) scanKeys(org string, pattern string, batchSize int64) int {
	scanned := 0
	var cursor uint64 = 0
	for {
		var (
			keys []string
			err  error
		)
		keys, cursor, err = r.rdb.Scan(r.ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			log.Error().Err(err).Uint64("cursor", cursor).Msg("scan failure")
		}
		nKeys := len(keys)
		scanned += nKeys
		log.Debug().Int("keys", nKeys).Uint64("cursor", cursor).Msg("scanned in this block")
		r.keysChans[org] <- keys

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

// getOrgId tries to find the org_id field in several known places within different kinds of keys
func getOrgId(jsonVal map[string]interface{}) (string, error) {
	orgIdValue := ""
	dataContainer := jsonVal

	// Maybe it's a session object?
	if udi, ok := jsonVal["UserData"]; ok {
		if ud, convOK := udi.(map[string]interface{}); convOK {
			dataContainer = ud
		}
	}

	// Maybe it's an API token or anything else with org_id on the root level?
	if vi, ok := dataContainer["org_id"]; ok {
		if v, convOK := vi.(string); convOK {
			orgIdValue = v
		} else {
			return "", errors.New("org_id is not a string")
		}
	} else {
		return "", errors.New("org_id container not found")
	}

	return orgIdValue, nil
}

// DumpOrgKeys is suited to run in prod. Just one goroutine per org to write the output file.
// The run time is about 3x that of the threaded version
func (r *redisClient) DumpOrgKeys(orgs []string, patterns []string, batchSize int64) {
	scanned := 0
	start := time.Now()

	var wg sync.WaitGroup
	for _, org := range orgs {
		go func() {
			wg.Add(1)
			log.Debug().Str("org", org).Msg("spawning filterOrg")
			r.filterOrg(org)
			wg.Done()
		}()
		for _, pattern := range patterns {
			log.Info().Str("org", org).Str("pattern", pattern).Msg("processing")
			scanned += r.scanKeys(org, pattern, batchSize)
		}
		close(r.keysChans[org])
	}
	wg.Wait()
	log.Info().Dur("time", time.Since(start)).Int("scanned", scanned).Msg("done with keys")
}
