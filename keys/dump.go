package keys

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

func DumpOrgKeys(rdb *redis.UniversalClient, pattern string, org string) error {
	keys, err := GetKeys(pattern)
	if err != nil {
		return err
	}
	log.Debug().Str("org", org).Msg("looking for")
	log.Debug().Int("keys", len(keys)).Msgf("pattern: %s", pattern)

	found := 0
	for index, k := range keys {
		byteVal, err := Get(k)
		if err != nil {
			log.Error().Err(err).Str("key", k).Msg("could not retrieve")
			continue
		}
		var jsonVal = make(map[string]interface{})
		err = json.NewDecoder(bytes.NewReader(byteVal)).Decode(&jsonVal)
		if err != nil {
			log.Error().Err(err).Bytes("input", byteVal).Msg("unexpected error when parsing")
			continue
		}
		if jsonVal["org_id"] == org {
			found++
			ttl, err := TTL(k)
			if err != nil {
				log.Error().Err(err).Msgf("could not get TTL for key: %s", k)
				continue
			}

			output, err := json.Marshal(&redisKey{
				Name:  k,
				TTL:   ttl,
				Value: jsonVal,
			})
			if err != nil {
				log.Error().Err(err).Msgf("could not marshal")
				continue
			}
			os.Stdout.Write(output)
			// Write a new line
			os.Stdout.Write([]byte{10})
		}
		if index%1e4 == 0 {
			log.Trace().Int("keys", index).Msg("processed")
		}
	}
	log.Debug().Int("found", found).Msg("keys")
	return nil
}
