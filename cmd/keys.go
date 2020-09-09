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

	"github.com/TykTechnologies/gromit/keys"
	"github.com/spf13/cobra"
	"strings"
)

// keysCmd represents the top-level keys command
var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Dump/restore keys from/to redis",
	Long:  `This is meant to be run in prod but do take care.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("missing pattern for keys")
		}
		return nil
	},
}

// dump subcommand
var keysDumpCmd = &cobra.Command{
	Use:   "dump <pattern>",
	Short: "Dumps redis keys matching <pattern> to stdout",
	Long:  `Uses SCAN to dump keys so can be run in prod as long as you know what you are doing.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		addr, _ := cmd.Flags().GetString("")
		org, _ := cmd.Flags().GetString("org")
		rdb := keys.NewUniversalClient(strings.Split(addr, ","))
		return keys.DumpOrgKeys(rdb, args[0], org)
	},
}

// restore subcommand
var keysRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Sets string keys in redis, reading the keys and values from a JSON-lines formatted file.",
	Long:  `Uses a pipeline to batch updates, is aware of clusters`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr, _ := cmd.Flags().GetString("addr")
		rdb := keys.NewUniversalClient(strings.Split(addr, ","))
		keys.RestoreKeys(rdb, args[0])
	},
}

func init() {
	rootCmd.AddCommand(keysCmd)

	keysCmd.PersistentFlags().StringP("addr", "a", "", "Redis hosts (required), uses REDISCLI_AUTH if set. A comma-separated list will be used as a cluster.")
	keysCmd.MarkFlagRequired("addr")
	keysCmd.PersistentFlags().IntP("batch", "b", "100", "Batch size for pipeline.")

	keysCmd.PersistentFlags().StringP("db", "d", "", "Database name, used only for single-node and failover clients.")
	keysCmd.PersistentFlags().StringP("name", "n", "", "Sentinel master name, failover clients only.")

	keysDumpCmd.Flags().StringP("org", "o", "", "Org to dump keys for (required)")
	keysDumpCmd.MarkFlagRequired("org")
}
