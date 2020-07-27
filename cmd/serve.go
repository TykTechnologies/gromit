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
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/rs/zerolog/log"
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
		server.serve(ca, sCert, sKey)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&certPath, "certpath", "certs", "path to rootca and key pair. Expects files named ca.pem, server(-key).pem")
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
