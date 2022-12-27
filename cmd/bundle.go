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
	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path/filepath"
	"io/fs"
	"os"
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"templates"},
	Args:    cobra.MinimumNArgs(0),
	Short:   "Operate on an embedded bundle",
	Long: `A bundle is a collection of templates. A template is a top-level file which will be rendered with the same path as it is embedded as.
A template can have sub-templates which are in directories of the form, <template>.d. The contents of these directories will not be traversed looking for further templates but are collected into the list of files that is passed to template.New().`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		policy.ListBundles(".")
	},
}

var listSubCmd = &cobra.Command{
	Use:     "list <bundles...>",
	Aliases: []string{"ls"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "List the embedded template bundles like a directory listing",
	Long:    `Use this when you want to see what is inside this binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, bundle := range args {
			log.Logger = log.With().Str("bundle", bundle).Logger()
			bfs, err := fs.Sub(policy.Bundles, filepath.Join("templates", bundle))
			if err != nil {
				log.Warn().Err(err).Msg("fetching embedded templates")
			}
			b, err := policy.NewBundle(bfs, bundle)
			if err != nil {
				log.Fatal().Str("bundle", bundle).Err(err).Msg("could not get")
			}
			cmd.Println(b)
		}
	},
}

var localSubCmd = &cobra.Command{
	Use:     "local <dir>",
	Aliases: []string{"ls"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Render the locally supplied templates using the RepoPolicy object",
	Long:    `An easy way to work with templates without having to go through a gromit build. No involvement with git or github, takes the supplied templates and renders output using the config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir = args[0]
		log.Logger = log.With().Str("bundle", dir).Logger()
		opDir, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatal().Err(err).Msg("output dir")
		}
		bfs := os.DirFS(dir)
		b, err := policy.NewBundle(bfs, dir)
		if err != nil {
			log.Fatal().Str("bundle", dir).Err(err).Msg("instantiation")
		}
		for _, repo := range repos {
			log.Logger = log.With().Str("repo", repo).Logger()
			rp, err := configPolicies.GetRepo(repo, viper.GetString("prefix"), "master")
			if err != nil {
				log.Warn().Err(err).Msg("could not get repo")
				continue
			}
			err = b.Render(&rp, opDir, nil)
			if err != nil {
				log.Warn().Err(err).Msg("rendering")

			}
		}
	},
}

var genSubCmd = &cobra.Command{
	Use:     "gen <bundles...>",
	Aliases: []string{"generate", "render"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Render the given template bundles into the current dir, override with -o",
	Long:    `This command does not overlay the rendered output into a git tree. You will have to checkout the repo yourself if you want to check the rendered templates into a git repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		opDir, _ := cmd.Flags().GetString("output")
		for _, repo := range repos {
			log.Logger = log.With().Str("repo", repo).Logger()
			rp, err := configPolicies.GetRepo(repo, viper.GetString("prefix"), "master")
			if err != nil {
				log.Warn().Err(err).Msg("could not get repo")
				continue
			}
			for _, bundle := range args {
				log.Logger = log.With().Str("bundle", bundle).Logger()
				bfs, err := fs.Sub(policy.Bundles, filepath.Join("templates", bundle))
				if err != nil {
					log.Warn().Err(err).Msg("fetching embedded templates")
				}
				b, err := policy.NewBundle(bfs, bundle)
				if err != nil {
					log.Warn().Err(err).Msg("instantiation")

				}
				err = b.Render(&rp, opDir, nil)
				if err != nil {
					log.Warn().Err(err).Msg("rendering")

				}
			}
		}
	},
}

var diffSubCmd = &cobra.Command{
	Use:     "diff <bundle>",
	Args:    cobra.MinimumNArgs(1),
	Short:   "Render the bundle and diff it against known good output",
	Long:    `Known good templates`,
	Run: func(cmd *cobra.Command, args []string) {
		repo := repos[0]
		bundle := args[0]
		rp, err := configPolicies.GetRepo(repo, viper.GetString("prefix"), "master")
		if err != nil {
			log.Warn().Str("repo", repo).Err(err).Msg("could not get repo")
		}
		err = policy.RenderBundle(bundle, repo, &rp)
		if err != nil {
			log.Warn().Str("repo", repo).Str("bundle", bundle).Err(err).Msg("could not render")

		}
	},
}

func init() {
	genSubCmd.Flags().String("output", ".", "Output into this directory. Sub-directories will be created for each repo.")
	localSubCmd.Flags().String("output", ".", "Output into this directory. Sub-directories will be created for each repo.")
	bundleCmd.AddCommand(localSubCmd)
	bundleCmd.AddCommand(listSubCmd)
	bundleCmd.AddCommand(diffSubCmd)
	bundleCmd.AddCommand(genSubCmd)

	bundleCmd.PersistentFlags().StringSliceVar(&repos, "repos", []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink", "tyk-identity-broker", "portal", "tyk-analytics-ui"}, "Repos to operate upon, comma separated values accepted.")
	rootCmd.AddCommand(bundleCmd)
}
