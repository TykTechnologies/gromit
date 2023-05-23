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
	"github.com/spf13/cobra"
)

var (
	bundle, repo, branch string
	features []string
)

var bundleCmd = &cobra.Command{
	Use:     "bundle",
	Aliases: []string{"templates"},
	Short:   "Operate on bundles",
	Long: `A bundle is a collection of templates. A template is a top-level file which will be rendered with the same path as it is embedded as.
A template can have sub-templates which are in directories of the form, <template>.d. The contents of these directories will not be traversed looking for further templates but are collected into the list of files that is passed to template.New().`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := policy.NewBundle(bundle, features)
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
		b, err := policy.NewBundle(bundle, features)
		if err != nil {
			return fmt.Errorf("bundle %s: %v", bundle, err)
		}
		rp, err := policy.GetRepoPolicy(repo, branch)
		rp.SetTimestamp(time.Now().UTC())
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repo, err)
		}
		_, err = b.Render(&rp, dir, nil)
		return err
	},
}

func init() {
	bundleCmd.PersistentFlags().StringVar(&bundle, "bundle", "releng", "Bundle to use, local bundles should start with . or /")
	bundleCmd.PersistentFlags().StringVar(&repo, "repo", "tyk-pump", "Use parameters from policy.<repo>")
	bundleCmd.PersistentFlags().StringVar(&branch, "branch", "master", "Use branch values from policy.<repo>.branch")
	bundleCmd.PersistentFlags().StringSliceVar(&features, "feature", nil, "Features to enable")
	bundleCmd.MarkPersistentFlagRequired("bundle")
	bundleCmd.MarkPersistentFlagRequired("repo")
	bundleCmd.AddCommand(genSubCmd)

	rootCmd.AddCommand(bundleCmd)
}
