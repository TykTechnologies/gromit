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
	"io/ioutil"
	"os"
	"strings"

	"net/http"

	"github.com/TykTechnologies/gromit/licenser"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// token is the endpoint auth, defaults to GROMIT_LICENSER_TOKEN
var token string

// baseURL is the product agnostic part of the endpoint
var baseURL string

// licenserCmd represents the client command
var licenserCmd = &cobra.Command{
	Use:   "licenser [flags] <mdcb-trial|dash-trial> <path>",
	Short: "Get a trial license and writes it to path, overwriting it",
	Long: `Uses the Tyk gateway in the internal k8s cluster. This is the same endpoint that the /*-trial commands use and needs the auth token in GROMIT_LICENSER_TOKEN
Supports:
- dashboard
- mdcb`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		product := args[0]
		opPath := args[1]
		l := licenser.Licenser{
			Client: http.DefaultClient,
		}
		license, err := l.Fetch(baseURL, product, token)
		if err != nil {
			log.Fatal().Err(err).Str("baseURL", baseURL).Msg("could not fetch license")
		}
		aws, _ := cmd.Flags().GetBool("aws")
		license = strings.TrimSuffix(license, "\n")
		if aws {
			err = util.UpdateSecret(opPath, license)
		} else {
			err = ioutil.WriteFile(opPath, []byte(license), 0444)
		}

		if err != nil {
			log.Error().Err(err).Str("opFile", opPath).Msg("could not write")
		}
	},
}

func init() {
	rootCmd.AddCommand(licenserCmd)
	licenserCmd.PersistentFlags().StringVarP(&baseURL, "baseurl", "b", "https://bots.cluster.internal.tyk.technology/license-bot/", "base url for the licenser endpoint")
	licenserCmd.PersistentFlags().StringVarP(&token, "token", "t", os.Getenv("GROMIT_LICENSER_TOKEN"), "Auth token")
	licenserCmd.Flags().BoolP("aws", "a", false, "The path is the AWS secret name to store the secret in")
}
