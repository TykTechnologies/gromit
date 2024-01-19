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
	"fmt"
	"time"

	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	features       []string
	configPolicies policy.Policies
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"templates"},
	Short:   "Operate on bundles",
	Long: `A bundle is a collection of templates. A template is a top-level file which will be rendered with the same path as it is embedded as.
A template can have sub-templates which are in directories of the form, <template>.d. The contents of these directories will not be traversed looking for further templates but are collected into the list of files that used to instantiate <template>.
Templates can be organised into features, which is just a directory tree of templates. Rendering the same file from different templates is _not_ supported.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		b, err := policy.NewBundle(features)
		if err != nil {
			cmd.Println(err)
		}
		cmd.Println(b)
	},
}

var genSubCmd = &cobra.Command{
	Use:     "gen <dir>",
	Aliases: []string{"generate", "render"},
	Args:    cobra.ExactArgs(1),
	Short:   "Render <bundle> into <dir> using parameters from policy.<repo>. If <dir> is -, render to stdout.",
	Long:    `This command does not overlay the rendered output into a git tree. You will have to checkout the repo yourself if you want to check the rendered templates into a git repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		repoName, _ := cmd.Flags().GetString("repo")
		rp, err := configPolicies.GetRepoPolicy(repoName)
		rp.SetTimestamp(time.Now().UTC())
		rp.SetBranch(Branch)
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repoName, err)
		}
		b, err := policy.NewBundle(rp.Branchvals.Features)
		if err != nil {
			return fmt.Errorf("bundle: %v", err)
		}
		_, err = b.Render(rp, dir, nil)
		return err
	},
}

func init() {
	bundleCmd.Flags().StringSliceVar(&features, "features", []string{"releng"}, "Features to use")
	bundleCmd.PersistentFlags().String("repo", "", "Use parameters from policy.<repo>")
	bundleCmd.MarkPersistentFlagRequired("repo")
	bundleCmd.MarkFlagRequired("features")

	genSubCmd.Flags().StringVar(&Branch, "branch", "master", "Use branch values from policy.<repo>.branch")
	genSubCmd.MarkFlagRequired("branch")
	bundleCmd.AddCommand(genSubCmd)

	rootCmd.AddCommand(bundleCmd)
}
