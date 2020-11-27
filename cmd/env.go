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
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Mess about with the env state",
	Long:  `Certificates and such like are configured in the gromit config file (default ~/.config/gromit/config.yaml)`,
	Run: func(cmd *cobra.Command, args []string) {
		if mtls, _ := cmd.Flags().GetBool("mtls"); mtls {
			certs := viper.GetStringMapString("ccerts")
			for k, v := range certs {
				if !strings.HasPrefix(v, "/") {
					certs[k] = filepath.Join(viper.GetString("confpath"), v)
				}
			}
			log.Debug().Interface("certs", certs).Msg("loaded from viper")
		} else {
			log.Debug().Msg("not mtls")
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.PersistentFlags().StringP("server", "s", "gserve.dev.tyk.technology", "Server hostname, will use https always")
	envCmd.PersistentFlags().BoolP("mtls", "m", true, "Use mTLS")
	envCmd.PersistentFlags().StringVarP(&authToken, "auth", "a", viper.GetString("GROMIT_AUTHTOKEN"), "Auth token")
}
