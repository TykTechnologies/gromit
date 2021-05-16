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
	"fmt"
	"os"

	"github.com/TykTechnologies/gromit/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"

	"github.com/TykTechnologies/gromit/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var cfgFile, logLevel string
var textLogs bool

// AWScfg is used in cluster, sow, reap and the server
var AWScfg aws.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gromit",
	Short: "The glue that binds AWS Fargate, Github and DynamoDB",
	Long: `It also has a grab bag of various ops automation.
Global env vars:
These vars apply to all commands
GROMIT_TABLENAME DynamoDB tablename to use for env state
GROMIT_REPOS Comma separated list of ECR repos to answer for`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Call initConfig() for every command before running
	cobra.OnInitialize(initConfig)
	log.Logger = log.With().Caller().Str("version", util.Version()).Logger()

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "conf", "f", "config file (default is $HOME/.config/gromit.yaml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "loglevel", "l", "info", "Log verbosity: trace, info, warn, error")
	rootCmd.PersistentFlags().BoolVarP(&textLogs, "textlogs", "t", false, "Logs in plain text")
}

// initConfig reads in config file and env variables if set.
func initConfig() {
	ll, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Str("level", logLevel).Msg("Could not parse, defaulting to debug.")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(ll)
	}
	if textLogs {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	config.LoadConfig(cfgFile)
}
