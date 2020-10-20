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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var zoneID, domain, cluster string

// clusterCmd is a top level command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster of tyk components",
	Long:  `Set cluster to use via -c flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("no top-level functions, see subcommands.")
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
		terraform.Run(args[0])
	},
}

// exposeCmd adds r53 entries for ECS clusters
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Upsert a record in Route53 for the given ECS cluster",
	Long: `Given an ECS cluster, looks for all tasks with a public IP and 
makes A records in Route53 accessible as <task_name>.<domain>.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := devenv.UpdateClusterIPs(cluster, zoneID, domain)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to update cluster IPs for 5s", cluster)
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
	clusterCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", os.Getenv("GROMIT_CLUSTER"), "Cluster to be operated on")
	clusterCmd.PersistentFlags().StringVarP(&zoneID, "zone", "z", os.Getenv("GROMIT_ZONEID"), "Route53 zone id to make entries in")
	clusterCmd.MarkFlagRequired("zone")
	clusterCmd.PersistentFlags().StringVarP(&domain, "domain", "d", os.Getenv("GROMIT_DOMAIN"), "Domain part of the DNS record")
	clusterCmd.MarkFlagRequired("domain")

	clusterCmd.AddCommand(runCmd, exposeCmd, tdbCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
