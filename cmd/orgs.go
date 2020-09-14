package cmd

/*
Copyright © 2020 Tyk Technologies

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"os"
	"strings"

	"github.com/TykTechnologies/gromit/orgs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	redisHosts      string
	redisMasterName string
	mongoURL        string
	oDir            string
)

// orgsCmd represents the top-level orgs command
var orgsCmd = &cobra.Command{
	Use:   "orgs <subcommand>",
	Short: "Dump/restore org keys and mongodb",
	Long:  `This is meant to be run in prod but do take care.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

// orgsDumpCmd operates on redis and mongo
var orgsDumpCmd = &cobra.Command{
	Use:   "dump org0 org1 ...",
	Short: "Concurrently dump mongo and redis",
	Long: `Dumps keys from redis that match patterns in -p.
Dumps mongo records from collections in -u -v -a.
Writes collections in {orgid}_colls/{db}/*.bson and keys in {orgid}.keys.jl. Existing files are clobbered.
Uses SCAN with COUNT to dump redis keys so can be run in prod.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Mongo
		org_idColls, _ := cmd.Flags().GetString("org_id_colls")
		orgidColls, _ := cmd.Flags().GetString("orgid_colls")
		aggColls, _ := cmd.Flags().GetString("agg_colls")
		muri, err := orgs.ParseMongoURI(mongoURL)
		if err != nil {
			log.Fatal().Err(err).Str("murl", mongoURL).Msg("could not parse")
		}
		topts := orgs.DumpCollectionOpts(muri)
		for _, org := range args {
			log.Info().Str("org", org).Msg("processing collections")

			orgs.DumpFilteredCollections(topts, "org_id", org, strings.Split(org_idColls, ","))
			orgs.DumpFilteredCollections(topts, "orgid", org, strings.Split(orgidColls, ","))

			var aggs []string
			for _, coll := range strings.Split(aggColls, ",") {
				aggs = append(aggs, coll+org)
			}
			orgs.DumpAnalyticzCollections(topts, org, aggs)
		}

		// Redis
		patterns, _ := cmd.Flags().GetString("patterns")
		count, _ := cmd.Flags().GetInt64("count")
		workers, _ := cmd.Flags().GetInt("workers")

		log.Info().Msg("Dumping keys")
		r := orgs.RedisClient(strings.Split(redisHosts, ","), redisMasterName)
		redisChans := make(map[string]chan ([]string))
		for _, org := range args {
			log.Info().Str("org", org).Msg("processing keys")
			redisChans[org] = make(chan []string, count)
			for w := 0; w < workers; w++ {
				// Workers will wait until ScanKeys() writes to the channel
				go r.FilterOrg(org, redisChans[org])
			}
			for _, pattern := range strings.Split(patterns, ",") {
				r.ScanKeys(pattern, count, redisChans[org])
			}
			close(redisChans[org])
			log.Info().Str("org", org).Msg("done dumping keys")
		}
	},
}

// orgsRestoreCmd operates on redis and mongo
var orgsRestoreCmd = &cobra.Command{
	Use:   "restore org0 org1 ...",
	Short: "Concurrently restore mongo and redis",
	Long:  `Expects files named {orgid}.mongo.bson and {orgid}.redis.jl`,
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(orgsCmd)
	orgsCmd.AddCommand(orgsDumpCmd)
	orgsCmd.AddCommand(orgsRestoreCmd)

	orgsCmd.PersistentFlags().StringVarP(&redisHosts, "redis", "r", os.Getenv("REDIS_HOSTS"), "Redis hosts (required), uses REDISCLI_AUTH if set. A comma-separated list will be used as a cluster.")
	orgsCmd.PersistentFlags().StringVarP(&mongoURL, "murl", "m", os.Getenv("MONGO_URL"), "Mongo URL mongodb://...")
	orgsCmd.PersistentFlags().StringVarP(&redisMasterName, "name", "n", os.Getenv("REDIS_MASTER"), "Sentinel master name, failover clients only.")
	orgsDumpCmd.Flags().StringVarP(&oDir, "dir", "d", ".", "Directory to read/write files")
	orgsCmd.MarkFlagRequired("redis")
	orgsCmd.MarkFlagRequired("murl")

	orgsDumpCmd.Flags().StringP("patterns", "p", "apikey-*,tyk-admin-api-*", "Comma separated list of patterns to SCAN for")
	orgsDumpCmd.PersistentFlags().Int64P("count", "c", 100, "Passed as COUNT to SCAN, effectively batchsize")
	orgsDumpCmd.Flags().IntP("workers", "w", 4, "Concurrency level of mgets")
	orgsDumpCmd.Flags().StringP("org_id_colls", "u", "portal_catalogue,portal_configurations,portal_css,portal_developers,portal_key_requests,portal_menus,portal_pages,tyk_apis,tyk_policies", "These will be queried by org_id")
	orgsDumpCmd.Flags().StringP("orgid_colls", "v", "tyk_analytics_users", "These will be queried by orgid")
	orgsDumpCmd.Flags().StringP("agg_colls", "a", "z_tyk_analyticz_,z_tyk_analyticz_aggregate_", "These will have the org_id suffixed to their names")
}
