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
	"github.com/TykTechnologies/gromit/devenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// envConfig holds global environment variables
type exposeEnvConfig struct {
	ZoneID string
	Domain string
}

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Upsert a record in Route53 for the given ECS cluster",
	Long: `Makes entries in Route53 as <task_name>.R53_DOMAIN
Env vars:
R53_ZONEID Route53 zone to use for external DNS
R53_DOMAIN domain served by GROMIT_ZONEID`,
	Run: func(cmd *cobra.Command, args []string) {
		var e exposeEnvConfig

		err := envconfig.Process("r53", &e)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not get env")
		}
		log.Info().Interface("env", e).Msg("loaded env")
		err = devenv.UpdateClusterIPs(args[0], e.ZoneID, e.Domain)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to update cluster IPs for 5s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// exposeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// exposeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
