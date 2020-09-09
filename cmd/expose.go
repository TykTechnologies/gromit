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
	Long: `Given an ECS cluster, looks for all tasks with a public IP and 
makes A records in Route53 accessible as <task_name>.<domain>.`,
	Run: func(cmd *cobra.Command, args []string) {
		zoneID, _ := cmd.Flags().GetString("zone")
		domain, _ := cmd.Flags().GetString("domain")
		
		err := devenv.UpdateClusterIPs(args[0], zoneID, domain)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to update cluster IPs for 5s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)

	exposeCmd.PersistentFlags().StringP("zone", "z", "", "Route53 zone id to make entries in")
	exposeCmd.MarkFlagRequired("zone")
	
	exposeCmd.PersistentFlags().StringP("domain", "d", "dev.tyk.technology", "Domain part of the DNS record")
	exposeCmd.MarkFlagRequired("domain")
}
