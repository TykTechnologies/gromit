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

	"github.com/TykTechnologies/gromit/server"
	"github.com/spf13/cobra"
)

var certPath string

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run endpoint for github requests",
	Long: `Runs an HTTPS server, bound to 443 that can be accesses only via mTLS. 

This endpoint is notified by the int-image workflows in the various repos when there is a new build`,
	Run: func(cmd *cobra.Command, args []string) {
		ca := filepath.Join(certPath, "ca.pem")
		sCert := filepath.Join(certPath, "server.pem")
		sKey := filepath.Join(certPath, "server-key.pem")
		server.Serve(ca, sCert, sKey)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&certPath, "certpath", "certs", "path to rootca and key pair. Expects files named ca.pem, server(-key).pem")
}
