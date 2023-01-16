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

var cfgFile string

// AWScfg is used in cluster, sow, reap and the server
var AWScfg aws.Config

// Repos is used in git, bundle and policy
var Repos = []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink", "tyk-identity-broker", "portal"}

// Branch and Owner used in git and policy
var Branch, Owner string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gromit",
	Short: "The glue that binds AWS and Github",
	Long: `It also has a grab bag of various ops automation.
Each gromit command has its own config section. For instance, the policy command uses the policy key in the config file. Config values can be overridden by environment variables. For instance, policy.prefix can be overridden using the variable $GROMIT_POLICY_PREFIX.`,
	SilenceUsage: true,
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "conf", "f", "", "YAML config file. If not supplied, embedded defaults will be used")
	rootCmd.PersistentFlags().String("loglevel", "info", "Log verbosity: trace, debug, info, warn, error/fatal")
	rootCmd.PersistentFlags().Bool("textlogs", false, "Logs in plain text")
}

// initConfig reads in config file and env variables if set.
func initConfig() {
	logLevel, _ := rootCmd.Flags().GetString("loglevel")
	ll, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Str("level", logLevel).Msg("Could not parse, defaulting to debug.")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(ll)
	}
	log.Logger = log.With().Str("version", util.Version()).Caller().Logger()
	textLogs, _ := rootCmd.Flags().GetBool("textlogs")
	if textLogs {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	config.LoadConfig(cfgFile)
}
