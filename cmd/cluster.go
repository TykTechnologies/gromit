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

	"github.com/TykTechnologies/gromit/config"
	"github.com/TykTechnologies/gromit/devenv"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var clusterName string

// clusterCmd is a top level command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster of tyk components",
	Long:  `Set cluster to use via -c flag. With no parameters it will list the clusters.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		AWScfg, err = external.LoadDefaultAWSConfig()
		if err != nil {
			log.Fatal().Msg("Could not load AWS config")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		clusters, err := devenv.ListClusters(ecs.New(AWScfg))
		if err != nil {
			panic(err)
		}
		fmt.Println(clusters)
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
			cnames, err := devenv.ListClusters(ecs.New(AWScfg))
			if err != nil {
				log.Fatal().Err(err).Msg("could not get list of clusters")
			}
			clusters := devenv.FastFetchClusters(cnames)
			for _, c := range clusters {
				c.SyncDNS(route53.ChangeActionUpsert, config.ZoneID, config.Domain)
			}
		} else {
			cluster, err := devenv.GetGromitCluster(clusterName)
			if err != nil {
				log.Error().Err(err).Str("cluster", clusterName).Msg("fetching")
			}
			cluster.SyncDNS(route53.ChangeActionUpsert, config.ZoneID, config.Domain)
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
		return devenv.Tdb(args[0], args[1:]...)
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.PersistentFlags().StringVarP(&clusterName, "cluster", "c", os.Getenv("GROMIT_CLUSTER"), "Cluster to be operated on")

	exposeCmd.Flags().BoolP("all", "a", false, "All available public IPs in all clusters will be exposed")
	clusterCmd.AddCommand(exposeCmd, tdbCmd)
}
