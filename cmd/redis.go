package cmd

import "github.com/spf13/cobra"

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

// redisCmd represents the redis command
var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "operate at the org level with redis",
	Long:  `These commands are meant to be run in prod. However, it might be wise to run it against a read-replica.`,
}

func init() {
	rootCmd.AddCommand(redisCmd)

	redisCmd.PersistentFlags().StringP("server", "s", "", "Redis host (required), uses REDISCLI_AUTH if set")
	redisCmd.MarkFlagRequired("host")

	redisCmd.PersistentFlags().StringP("org", "o", "", "Org to dump keys for (required)")
	redisCmd.MarkFlagRequired("org")
}
