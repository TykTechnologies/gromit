package cmd

import (
	"fmt"
	"net/http"
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

// licenserCmd represents the client command
var licenserCmd = &cobra.Command{
	Use:   "licenser [flags] <mdcb-trial|dashboard-trial> <path>",
	Short: "Get a trial license and (over)writes it to SSM path",
	Long: `Uses the Tyk gateway in the internal k8s cluster. This is the same endpoint that the /*-trial commands use and needs the auth token in LICENSER_TOKEN
Supports:
- dashboard
- mdcb`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		product := args[0]
		opPath := args[1]
		l := env.Licenser{
			Client: http.DefaultClient,
		}
		baseURL, _ := cmd.Flags().GetString("baseurl")
		license, err := l.Fetch(baseURL, product, os.Getenv("LICENSER_TOKEN"))
		if err != nil {
			return fmt.Errorf("Could not fetch licenses for %s from %s: %w", product, baseURL, err)
		}
		keyid, _ := cmd.Flags().GetString("key")
		return EnvClient.StoreLicense(license, opPath, keyid)
	},
}

func init() {
	licenserCmd.Flags().String("baseurl", "https://bots.cluster.internal.tyk.technology/license-bot/", "base url for the licenser endpoint")
	licenserCmd.Flags().String("token", os.Getenv("GROMIT_LICENSER_TOKEN"), "Auth token for fetching trial license")
	licenserCmd.Flags().String("key", "215a7274-5652-4521-8a88-b18e02b8f13e", "KMS key id used to encrypt the license")

	exposeCmd.Flags().String("zone", "dev.tyk.technology", "Name of the Route53 hosted zone in which to make entries in")
	envCmd.AddCommand(exposeCmd)
	envCmd.AddCommand(licenserCmd)

	envCmd.PersistentFlags().StringVar(&envName, "env", "", "ECS Cluster to operate on")
	rootCmd.AddCommand(envCmd)
}
