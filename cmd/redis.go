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
	"errors"
	"fmt"

	"github.com/TykTechnologies/gromit/redis"
	"github.com/spf13/cobra"
)

// redisCmd represents the redis command
var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Dump redis keys to files",
	Long: `Uses SCAN with COUNT to dump keys into files.

Meant to be run in prod.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("missing pattern for keys")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("server")
		redis.InitPool(host)

		org, _ := cmd.Flags().GetString("org")
		err := redis.DumpOrgKeys(args[0], org)
		fmt.Println(err)
	},
}

func init() {
	rootCmd.AddCommand(redisCmd)

	redisCmd.PersistentFlags().StringP("server", "s", "", "Redis host (required), uses REDISCLI_AUTH if set")
	redisCmd.MarkFlagRequired("host")

	redisCmd.PersistentFlags().StringP("org", "o", "", "Org to dump keys for (required)")
	redisCmd.MarkFlagRequired("org")
}
