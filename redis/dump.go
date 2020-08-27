package redis

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func DumpOrgKeys(pattern string, dumpDir string, org string) error {
	keys, err := GetKeys(pattern)
	if err != nil {
		return err
	}
	log.Debug().Int("keys", len(keys)).Msgf("pattern: %s", pattern)
	log.Debug().Str("org", org).Msg("looking for")
	found := 0
	for index, k := range keys {
		byteVal, err := Get(k)
		if err != nil {
			return err
		}
		var jsonVal map[string]interface{}
		err = json.NewDecoder(bytes.NewReader(byteVal)).Decode(&jsonVal)
		if err != nil {
			return err
		}
		if jsonVal["org_id"] == org {
			found++
			dumpPath := filepath.Join(dumpDir, k)
			dumpData, err := json.Marshal(jsonVal)
			if err != nil {
				log.Error().Err(err).Msgf("while marshalling json for key: %s", k)
			}
			ioutil.WriteFile(dumpPath, dumpData, 0644)
		}
		if index%100 == 0 {
			log.Trace().Int("records", index).Msg("processed")
		}
	}
	log.Debug().Int("found", found).Msg("keys")
	return nil
}
