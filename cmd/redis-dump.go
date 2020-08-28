package cmd

/*
Copyright © 2020 Tyk technologies

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
	"github.com/TykTechnologies/gromit/redis"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// dumpCmd represents the dump command
var redisDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump redis keys for an org",
	Long: `Uses SCAN with COUNT to dump keys into files.
Meant to be run in prod.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("server")
		redis.InitPool(host)

		org, _ := cmd.Flags().GetString("org")

		var keyPrefixes = []string{"apikey-*", "tyk-admin-api-*"}
		for _, prefix := range keyPrefixes {
			err := redis.DumpOrgKeys(prefix, org)
			if err != nil {
				log.Error().Err(err).Str("org", org).Msgf("getting %s", prefix)
			}
		}
	},
}

func init() {
	redisCmd.AddCommand(redisDumpCmd)
}
