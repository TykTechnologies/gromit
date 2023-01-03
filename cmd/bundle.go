/*
	Copyright © 2021 Tyk Technologies

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
package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	bundle, repo string
	b            *policy.Bundle
	rp           policy.RepoPolicy
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"templates"},
	Short:   "Operate on bundles",
	Long: `A bundle is a collection of templates. A template is a top-level file which will be rendered with the same path as it is embedded as.
A template can have sub-templates which are in directories of the form, <template>.d. The contents of these directories will not be traversed looking for further templates but are collected into the list of files that is passed to template.New().`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
		log.Logger = log.With().Str("bundle", bundle).Logger()
		var bfs fs.FS
		if strings.HasPrefix(bundle, ".") || strings.HasPrefix(bundle, "/") {
			bfs = os.DirFS(bundle)
		} else {
			bfs, err = fs.Sub(policy.Bundles, filepath.Join("templates", bundle))
			if err != nil {
				log.Fatal().Err(err).Msg("fetching embedded templates")
			}
		}
		b, err = policy.NewBundle(bfs, bundle)
		if err != nil {
			log.Fatal().Str("bundle", bundle).Err(err).Msg("could not get")
		}
		rp, err = configPolicies.GetRepo(repo, viper.GetString("prefix"), "master")
		if err != nil {
			log.Fatal().Err(err).Msg("could not get policy.repo")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(b)
		cmd.Println(rp)
	},
}

var genSubCmd = &cobra.Command{
	Use:     "gen <dir>",
	Aliases: []string{"generate", "render"},
	Args:    cobra.ExactArgs(1),
	Short:   "Render <bundle> into <dir> using parameters from policy.<repo>. If <dir> is -, render to stdout.",
	Long:    `This command does not overlay the rendered output into a git tree. You will have to checkout the repo yourself if you want to check the rendered templates into a git repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		err := b.Render(&rp, dir, nil)
		if err != nil {
			cmd.Println(b, rp, err)
		}
	},
}

func init() {
	bundleCmd.PersistentFlags().StringVar(&bundle, "bundle", "releng", "Bundle to use, local bundles should start with . or /")
	bundleCmd.PersistentFlags().StringVar(&repo, "repo", "tyk-pump", "Use parameters from policy.<repo>")
	bundleCmd.MarkPersistentFlagRequired("bundle")
	bundleCmd.MarkPersistentFlagRequired("repo")
	bundleCmd.AddCommand(genSubCmd)

	rootCmd.AddCommand(bundleCmd)
}
