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
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/terraform"
	"github.com/TykTechnologies/gromit/util"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var zoneID, domain, clusterName string

// clusterCmd is a top level command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster of tyk components",
	Long:  `Set cluster to use via -c flag. With no parameters it will list the clusters.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusters, err := devenv.ListClusters()
		if err != nil {
			panic(err)
		}
		fmt.Println(clusters)
	},
}

// runCmd will process envs from DynamoDB
var runCmd = &cobra.Command{
	Use:   "run <config bundle path>",
	Short: "Process envs from GROMIT_TABLENAME using supplied config bundle path",
	Long: `Read state and call the embedded devenv terraform manifest for new envs. The config bundle is a directory tree containing config files for all the components in the cluster. The names of the config dirs have to strictly match the repository names.

This component is meant to run in a scheduled task.
Env vars:
GROMIT_ZONEID Route53 zone to use for external DNS
GROMIT_DOMAIN Route53 domain corresponding to GROMIT_ZONEID
If testing locally, you may also have to set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and TF_API_TOKEN`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("name", util.Name()).Str("component", "run").Str("version", util.Version()).Msg("starting")
		terraform.Run(args[0])
	},
}

// reapCmd will delete envs from DynamoDB
var reapCmd = &cobra.Command{
	Use:   "reap <config bundle path>",
	Short: "Process envs from GROMIT_TABLENAME using supplied config bundle path",
	Long: `Read state and call the embedded devenv terraform manifest for new envs. The config bundle is a directory tree containing config files for all the components in the cluster. The names of the config dirs have to strictly match the repository names.

This component is meant to run in a scheduled task.
Env vars:
GROMIT_ZONEID Route53 zone to use for external DNS
GROMIT_DOMAIN Route53 domain corresponding to GROMIT_ZONEID
If testing locally, you may also have to set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and TF_API_TOKEN`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Str("name", util.Name()).Str("component", "reap").Str("version", util.Version()).Msg("starting")
		terraform.Reap(args[0])
	},
}

// exposeCmd adds r53 entries for ECS clusters
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Upsert a record in Route53 for the given ECS cluster",
	Long: `Given an ECS cluster, looks for all tasks with a public IP and 
makes A records in Route53 accessible as <task>.<cluster>.<domain>.`,
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			cnames, err := devenv.ListClusters()
			if err != nil {
				log.Fatal().Err(err).Msg("could not get list of clusters")
			}
			clusters := devenv.FastFetchClusters(cnames)
			for _, c := range clusters {
				c.SyncDNS(route53.ChangeActionUpsert, zoneID, domain)
			}
		} else {
			cluster, err := devenv.GetGromitCluster(clusterName)
			if err != nil {
				log.Error().Err(err).Str("cluster", clusterName).Msg("fetching")
			}
			cluster.SyncDNS(route53.ChangeActionUpsert, zoneID, domain)
		}
	},
}

// tdbCmd will run a debugging image in an ECS cluster
var tdbCmd = &cobra.Command{
	Use:   "tdb <image> [optional cmd line]",
	Short: "Starts a container connected to the cluster network",
	Long: `Will also upsert an entry into r53 to access this newly created container which is accessible over ssh using the key for this env.
Use this for debugging or for a quick load test`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Tdb(args[0], args[1:]...)
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.PersistentFlags().StringVarP(&clusterName, "cluster", "c", os.Getenv("GROMIT_CLUSTER"), "Cluster to be operated on")
	clusterCmd.PersistentFlags().StringVarP(&zoneID, "zone", "z", viper.GetString("cluster.zoneid"), "Route53 zone id to make entries in")
	clusterCmd.PersistentFlags().StringVarP(&domain, "domain", "d", viper.GetString("cluster.domain"), "Suffixed to the DNS record to make an FQDN")

	exposeCmd.Flags().BoolP("all", "a", false, "All available public IPs in all clusters will be exposed")
	clusterCmd.AddCommand(runCmd, exposeCmd, tdbCmd)
}
