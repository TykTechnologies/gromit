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
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// EnvConfig holds global environment variables
type EnvConfig struct {
	Repos      []string
	TableName  string
	RegistryID string
}

// unexported globals
var e EnvConfig
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
		startmTLSServer(&ca, &sCert, &sKey)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&certPath, "certpath", "certs", "path to rootca and key pair. Expects files named ca.pem, server(-key).pem")
}

func startmTLSServer(ca *string, cert *string, key *string) {
	err := envconfig.Process("gromit", &e)
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Interface("env", e).Msg("loaded env")

	// Set global cfg
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load SDK config")
	}

	http.HandleFunc("/healthcheck", handleHealthcheck)
	http.HandleFunc("/loglevel", handleLoglevel)
	http.HandleFunc("/newbuild", handleAWSRequest(&cfg, newBuild))
	http.HandleFunc("/env", handleAWSRequest(&cfg, listEnvs))

	caCert, err := ioutil.ReadFile(*ca)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not find CA certificate")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}
	log.Info().Msgf("starting server, certs loaded from %s", certPath)
	if err := server.ListenAndServeTLS(*cert, *key); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server startup failed")
	}
}

// Build represents a single build event from a repo
type Build struct {
	Repo string
	Ref  string
	Sha  string
}

// A closure to hold aws.Config
// Calls readHandler to do the actual work
func handleAWSRequest(cfg *aws.Config, realHandler func(*aws.Config, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	if cfg == nil {
		log.Fatal().Msg("nil AWS config")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		realHandler(cfg, w, r)
	}
}

// Real handlers

func newBuild(cfg *aws.Config, w http.ResponseWriter, r *http.Request) {
	var req Build
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Trace().Interface("req", req).Msg("parsed from github")

	// Github sends a path like refs/.../integration/<ref that we want>
	ss := strings.Split(req.Ref, "/")
	req.Ref = ss[len(ss)-1]

	state, err := devenv.GetEnvState(ecr.New(*cfg), e.RegistryID, req.Ref, e.Repos)
	if err != nil {
		log.Warn().Err(err).Msg("could not unmarhsal state")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Trace().Interface("state", state).Msg("initial")
	state[req.Repo] = req.Sha
	log.Trace().Interface("state", state).Msg("final")

	err = devenv.UpsertNewBuild(dynamodb.New(*cfg), e.TableName, req.Ref, state)
	if err != nil {
		log.Warn().Err(err).Msg("could not add new build")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	io.WriteString(w, "OK New build "+req.Ref)
}

// XXX: listEnvs only lists master, make this a rest endpoint
func listEnvs(cfg *aws.Config, w http.ResponseWriter, r *http.Request) {
	state, err := devenv.GetEnvState(ecr.New(*cfg), e.RegistryID, "master", e.Repos)
	log.Trace().Interface("state", state).Msg("listEnvs")

	envs, err := json.Marshal(state)
	if err != nil {
		log.Warn().Err(err).Msg("could not unmarhsal state")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(envs)
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
	log.Debug().Msg("Healthcheck")
}

func handleLoglevel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		level, err := zerolog.ParseLevel(r.URL.Query().Get("level"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		zerolog.SetGlobalLevel(level)

	case "GET":
		io.WriteString(w, zerolog.GlobalLevel().String())
		return
	}
	currLevel := zerolog.GlobalLevel().String()
	log.Debug().Msgf("loglevel set to %s", currLevel)

	io.WriteString(w, "Level set to "+currLevel)
}
