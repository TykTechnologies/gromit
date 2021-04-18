package cmd

/*
   Copyright © 2020 Tyk Technology https://tyk.io

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
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var envName, inputFile string
var client devenv.GromitClient

// Using functions instead of vars makes testing easier
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Mess about with the env state",
	Long: `Certificates and such like are configured in the env.ccerts section 
of the gromit config file`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		server := viper.GetString("serve.url")
		if mtls, _ := cmd.Flags().GetBool("mtls"); mtls {
			tls := util.TLSAuthClient{
				CA:   []byte(viper.GetString("ca")),
				Cert: []byte(viper.GetString("client.cert")),
				Key:  []byte(viper.GetString("client.key")),
			}

			c, err := tls.GetHTTPSClient()
			if err != nil {
				log.Fatal().Err(err).Msg("getting http client")
			}
			client = devenv.GromitClient{
				Server: server,
				Client: c,
			}
		} else {
			client = devenv.GromitClient{
				Server:    server,
				AuthToken: viper.GetString("authtoken"),
				Client: http.Client{
					Timeout: time.Duration(10 * time.Second),
				},
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		env, err := client.Get(envName)
		if err != nil {
			log.Fatal().Err(err).Str("env", envName).Msg("fetching")
		}
		cmd.Println(env)
	},
}

// replaceSubCmd is under envCmd
var replaceSubCmd = &cobra.Command{
	Use:     "replace",
	Aliases: []string{"create", "new"},
	Short:   "Replace an environment",
	Long: `Blindly submits a PUT request to the API, which may create a new resource.
When the gromit run scheduled task runs, this environment in ECS will be updated.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inpFile, _ := cmd.Flags().GetString("file")
		var input io.Reader
		if inpFile == "-" {
			input = os.Stdin
		} else if len(inpFile) > 0 {
			var err error
			input, err = os.Open(inpFile)
			if err != nil {
				log.Fatal().Err(err).Str("file", inpFile).Msg("input for replace")
			}
		} else {
			input = strings.NewReader(args[0])
		}
		return client.Replace(envName, input)
	},
}

var deleteSubCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"del", "rm"},
	Short:   "Update an environment",
	Long: `Blindly submits a DELETE request to the API.
The environment in ECS will continue to run. Gromit run will no longer be aware of the environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.Delete(envName)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.AddCommand(replaceSubCmd)
	envCmd.AddCommand(deleteSubCmd)
	envCmd.PersistentFlags().BoolP("mtls", "m", true, "Use mTLS")
	envCmd.PersistentFlags().StringVarP(&envName, "envname", "e", "", "Name of the environment")
	envCmd.PersistentFlags().StringP("file", "f", "-", "File to use as input - reads from stdin")
}
