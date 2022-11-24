package orgs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"

	"github.com/TykTechnologies/tyk/certs"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	getKeysTimeout = 10 * time.Second
)

type redisKey struct {
	Name  string `json:"name"`
	TTL   int64  `json:"ttl"`
	Value string `json:"value"`
}

type RedisOptions struct {
	Addrs      []string
	MasterName string
	MaxRetries int
	BatchSize  int64
}

type RedisClient struct {
	rdb       redis.UniversalClient
	ctx       context.Context
	keysChans map[string]chan []string
	opFiles   map[string]string
}

func NewRedisClient(ctx context.Context, opts *RedisOptions, orgs []string, dir string) RedisClient {
	var rdb redis.UniversalClient
	rdb = redis.NewUniversalClient(&redis.UniversalOptions{
		MaxRetries:      opts.MaxRetries,
		PoolSize:        10,
		MinIdleConns:    1,
		ReadTimeout:     getKeysTimeout,
		WriteTimeout:    getKeysTimeout,
		PoolTimeout:     2 * getKeysTimeout,
		IdleTimeout:     10 * getKeysTimeout,
		Password:        os.Getenv("REDISCLI_AUTH"),
		MasterName:      opts.MasterName,
		Addrs:           opts.Addrs,
		MaxRetryBackoff: 1 * time.Second,
	})

	rdb.Ping(ctx)

	keysChans := make(map[string]chan []string)
	opFiles := make(map[string]string)

	for _, org := range orgs {
		log.Debug().Str("org", org).Msg("setting up channels")
		keysChans[org] = make(chan []string, opts.BatchSize)
		opFiles[org] = filepath.Join(dir, org+".keys.jl")
	}

	return RedisClient{
		rdb,
		ctx,
		keysChans,
		opFiles,
	}
}

// filterOrg will MGET the block of keys and write out all those belonging to org
func (r *RedisClient) filterOrg(org, oldEncodingSecret, newEncodingSecret string) {
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

			keyOrg, err := getOrgId(keys[i], val.(string))
			if err != nil {
				log.Error().Err(err)
				continue
			}

			if keyOrg == org {
				found++
				ttl, _ := r.getTTL(keys[i])
				keyData, err := getKeyData(keys[i], val.(string), oldEncodingSecret, newEncodingSecret)
				if err != nil {
					log.Error().Err(err).Msg("Issue while reading key data")
					continue
				}

				output, err := json.Marshal(&redisKey{
					Name:  keys[i],
					TTL:   ttl,
					Value: keyData,
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

func (r *RedisClient) scanKeys(org string, pattern string, batchSize int64) int {
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

func (r *RedisClient) getTTL(keyName string) (int64, error) {
	var duration time.Duration
	var err error

	if duration, err = r.rdb.TTL(r.ctx, keyName).Result(); err != nil {
		return 0, err
	}

	return int64(duration.Seconds()), nil
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
func getOrgId(key, value string) (string, error) {
	if strings.HasPrefix(key, "cert-") {
		return key[8:14], nil
	}

	// Check if session object
	orgId, err := jsonparser.GetString([]byte(value), "UserData", "org_id")
	if err == nil {
		return orgId, nil
	}

	orgId, err = jsonparser.GetString([]byte(value), "org_id")
	if err != nil || orgId == "" {
		return "", errors.New("org_id not found")
	}

	return orgId, nil
}

func getKeyData(key, value, oldEncodingSecret string, newEncodingSecret string) (string, error) {
	// For certificates decode them first before dumping
	if strings.HasPrefix(key, "cert-") {
		var decodedPEM [][]byte
		blocks, perr := certs.ParsePEM([]byte(value), oldEncodingSecret)

		if perr != nil {
			return "", errors.New("Error decoding certificate. CertID: " + key)
		}

		for _, block := range blocks {
			decodedPEM = append(decodedPEM, pem.EncodeToMemory(block))
		}

		pemString := string(bytes.Join(decodedPEM, []byte("\n")))

		_, newValue, err := certs.GetCertIDAndChainPEM(pemString, newEncodingSecret)

		if err != nil {
			return "", errors.New("Error re-encoding certificate. CertID: " + key + "." + err)
		}

		return pemString, nil
	}

	return value, nil
}

// DumpOrgKeys is suited to run in prod. Just one goroutine per org to write the output file.
// The run time is about 3x that of the threaded version
func (r *RedisClient) DumpOrgKeys(orgs []string, patterns []string, batchSize int64, prev_cert_encoding_secret, new_cert_encoding_secret) {
	scanned := 0
	start := time.Now()

	var wg sync.WaitGroup
	for _, org := range orgs {
		go func() {
			wg.Add(1)
			log.Debug().Str("org", org).Msg("spawning filterOrg")
			r.filterOrg(org, prev_cert_encoding_secret, new_cert_encoding_secret)
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
