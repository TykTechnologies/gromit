package cmd

import (
	"github.com/TykTechnologies/gromit/config"
	"github.com/TykTechnologies/gromit/devenv"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// sowCmd will process envs from DynamoDB
var sowCmd = &cobra.Command{
	Use:     "sow <config root path>",
	Aliases: []string{"run", "create", "update"},
	Short:   "Sow envs creating a config tree at <config root path>",
	Long: `Call the embedded devenv terraform manifest for new envs. The config root is where the config bundle, a directory tree containing config files for all the components in the cluster will be generated.

This component is meant to run in a scheduled task.
Env vars:
GROMIT_ZONEID Route53 zone to use for external DNS
GROMIT_DOMAIN Route53 domain corresponding to GROMIT_ZONEID
If testing locally, you may also have to set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and TF_API_TOKEN`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Logger = log.With().Str("component", "sow").Logger()
		log.Info().Msg("starting")
		var err error
		AWScfg, err = external.LoadDefaultAWSConfig()
		if err != nil {
			log.Fatal().Msg("Could not load AWS config")
		}
		envs, err := devenv.GetEnvsByState(dynamodb.New(AWScfg), config.TableName, devenv.NEW, config.Repos)
		if err != nil {
			log.Error().Err(err).Msg("could not get list of new envs")
		}
		for _, env := range envs {
			env.Sow(args[0])
		}
	},
}

// reapCmd will delete envs from DynamoDB
var reapCmd = &cobra.Command{
	Use:     "reap <config root path>",
	Aliases: []string{"del", "delete", "rm", "remove"},
	Short:   "Reap envs from GROMIT_TABLENAME, using a config tree at <config root path>",
	Long: `Call the embedded devenv terraform manifest for new envs. The config root is a directory tree containing config files for all the components in the cluster.

This component is meant to run in a scheduled task.
Env vars:
GROMIT_ZONEID Route53 zone to use for external DNS
GROMIT_DOMAIN Route53 domain corresponding to GROMIT_ZONEID
If testing locally, you may also have to set AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and TF_API_TOKEN`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Logger = log.With().Str("component", "sow").Logger()
		log.Info().Msg("starting")
		var err error
		AWScfg, err = external.LoadDefaultAWSConfig()
		if err != nil {
			log.Fatal().Msg("Could not load AWS config")
		}
		envs, err := devenv.GetEnvsByState(dynamodb.New(AWScfg), config.TableName, devenv.DELETED, config.Repos)
		if err != nil {
			log.Error().Err(err).Msg("could not get list of new envs")
		}
		for _, env := range envs {
			env.Reap(args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(sowCmd)
	rootCmd.AddCommand(reapCmd)
}
