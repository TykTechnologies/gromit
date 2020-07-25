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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type mtlsAuth struct {
	CA   string
	Cert string
	Key  string
}

func (auth *mtlsAuth) GetClient() (http.Client, error) {
	caCert, err := ioutil.ReadFile(auth.CA)
	if err != nil {
		return http.Client{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(auth.Cert, auth.Key)
	if err != nil {
		return http.Client{}, err
	}

	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
		Timeout: time.Duration(10 * time.Second),
	}, nil
}

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Interact with the gromit server",
	Long: `It can use:
- auth token
- mTLS`,
	Run: func(cmd *cobra.Command, args []string) {
		certPath, _ := cmd.Flags().GetString("certpath")
		mtls, _ := cmd.Flags().GetBool("mtls")

		client := http.Client{
			Timeout: time.Duration(10 * time.Second),
		}

		if mtls {
			auth := mtlsAuth{
				filepath.Join(certPath, "ca.pem"),
				filepath.Join(certPath, "client.pem"),
				filepath.Join(certPath, "client-key.pem"),
			}
			var err error
			client, err = auth.GetClient()
			if err != nil {
				log.Fatal().Err(err).Msg("could not construct client for mtls auth")
			}
		}
		host, _ := cmd.Flags().GetString("server")
		healthcheck(client, fmt.Sprintf("https://%s/healthcheck", host))
	},
}

func healthcheck(client http.Client, url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("could not construct request")
	}
	if len(authToken) > 0 {
		req.Header.Add("Authorization", authToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("error in response")
	}
	defer resp.Body.Close()
	respContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read response")
	}
	log.Info().Msg(string(respContent))
}

var authToken string

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.PersistentFlags().StringP("server", "s", "gromit.dev.tyk.technology", "Server hostname, will use https always")
	clientCmd.PersistentFlags().BoolP("mtls", "m", false, "Use mTLS")
	clientCmd.PersistentFlags().StringVarP(&authToken, "auth", "a", viper.GetString("GROMIT_AUTHTOKEN"), "Auth token")
	clientCmd.PersistentFlags().StringP("certpath", "c", os.Getenv("GROMIT_CLIENTCERTPATH"), "Path to client key pair")
}
