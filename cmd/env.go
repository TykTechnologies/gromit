package cmd

import (
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/env"
	"github.com/spf13/cobra"
)

var EnvClient *env.Client
var envName string

// envCmd is the top level command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage cluster of tyk components",
	Long:  `The environment is a Fargate cluster. You will need AWS API access to use this command.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		EnvClient, err = env.NewClientFromProfile(os.Getenv("AWS_PROFILE"))
		return err
	},
}

// exposeCmd adds r53 entries for Fargate clusters
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Upsert a record in Route53 for the given ECS cluster",
	Long: `Given an ECS cluster, looks for all tasks with a public IP and 
makes A records in Route53 accessible as <task>.<cluster>.<domain>.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		zone, err := cmd.Flags().GetString("zone")
		if err != nil || zone == "" {
			return fmt.Errorf("expose requires zoneid")
		}
		return EnvClient.Expose(envName, zone)
	},
}

func init() {
	exposeCmd.Flags().String("zone", "dev.tyk.technology", "Name of the Route53 hosted zone in which to make entries in")
	envCmd.AddCommand(exposeCmd)

	envCmd.PersistentFlags().StringVar(&envName, "env", "", "ECS Cluster to operate on")
	rootCmd.AddCommand(envCmd)
}
