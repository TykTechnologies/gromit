package orgs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/util"
	"github.com/mongodb/mongo-tools-common/db"
	"github.com/mongodb/mongo-tools-common/options"
	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/rs/zerolog/log"
)

// GetMultiKey gets multiple keys from the database
func (r *redisClient) FilterOrg(org string, keyChan chan []string) error {
	found := 0
	for keys := range keyChan {
		getCtx, getCancel := context.WithTimeout(context.Background(), getKeysTimeout)
		defer getCancel()

		values, err := r.rdb.MGet(getCtx, keys...).Result()
		if err != nil {
			getCancel()
			log.Error().Err(err).Msg("mget")
		}
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
				os.Stdout.Write(output)
				// Write a new line
				os.Stdout.Write([]byte{10})
			}
		}
	}
	log.Info().Int("found", found).Msg("keys")

	return nil
}

func (r *redisClient) ScanKeys(pattern string, batchSize int64, keyChan chan []string) {
	var cursor uint64
	for {
		var keys []string
		var err error
		scanCtx, scanCancel := context.WithTimeout(context.Background(), getKeysTimeout)
		defer scanCancel()
		keys, cursor, err = r.rdb.Scan(scanCtx, cursor, pattern, batchSize).Result()
		if err != nil {
			scanCancel()
			log.Error().Err(err).Str("pattern", pattern).Msg("scan failure")
		}
		log.Debug().Int("records", len(keys)).Msg("found")
		keyChan <- keys
		if cursor == 0 {
			break
		}
	}
}

func (r *redisClient) getTTL(keyName string) (ttl int64, err error) {
	getCtx, getCancel := context.WithTimeout(context.Background(), getKeysTimeout)
	defer getCancel()

	duration, err := r.rdb.TTL(getCtx, keyName).Result()
	if err != nil {
		getCancel()
	}
	return int64(duration.Seconds()), err
}

func DumpCollectionOpts(uri *options.URI) *options.ToolOptions {
	opts := options.New(util.Name, util.Version, util.Commit, "see gromit help", false, options.EnabledOptions{Auth: true, Connection: true, Namespace: true, URI: true})
	connOpts := uri.ParsedConnString()
	opts.URI = uri
	opts.Direct = false
	opts.Namespace.DB = connOpts.Database

	err := opts.NormalizeOptionsAndURI()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot setup tooloptions")
	}
	return opts
}

func DumpFilteredCollections(topts *options.ToolOptions, queryField string, queryValue string, colls []string) {
	sp, err := db.NewSessionProvider(*topts)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot configure sessionprovider")
	}
	for _, coll := range colls {
		topts.Namespace.Collection = coll
		mdb := mongodump.MongoDump{
			ToolOptions: topts,
			InputOptions: &mongodump.InputOptions{
				Query: fmt.Sprintf(`{"%s": "%s"}`, queryField, queryValue),
			},
			OutputOptions: &mongodump.OutputOptions{
				Out:                    fmt.Sprintf("%s_colls", queryValue),
				NumParallelCollections: 4,
			},
			SessionProvider: sp,
		}
		err := mdb.Init()
		if err != nil {
			log.Fatal().Err(err).Interface("mdb", mdb).Msg("could not init")
		}
		err = mdb.Dump()
		if err != nil {
			log.Fatal().Err(err).Str("collection", coll).Msg("dumping")
		}
	}
}

func DumpAnalyticzCollections(topts *options.ToolOptions, org string, colls []string) {
	sp, err := db.NewSessionProvider(*topts)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot configure sessionprovider")
	}
	for _, coll := range colls {
		topts.Namespace.Collection = coll
		mdb := mongodump.MongoDump{
			ToolOptions:  topts,
			InputOptions: &mongodump.InputOptions{},
			OutputOptions: &mongodump.OutputOptions{
				Out:                    fmt.Sprintf("%s_colls", org),
				NumParallelCollections: 4,
			},
			SessionProvider: sp,
		}
		err := mdb.Init()
		if err != nil {
			log.Fatal().Err(err).Interface("mdb", mdb).Msg("could not init")
		}
		err = mdb.Dump()
		if err != nil {
			log.Fatal().Err(err).Str("collection", coll).Msg("dumping")
		}
	}
}
