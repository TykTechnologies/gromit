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
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"github.com/TykTechnologies/gromit/git"
	"github.com/spf13/viper"
)

var gitCmd = &cobra.Command{
	Use:     "git <sub command>",
	Args:    cobra.MinimumNArgs(1),
	Short:   "Top-level git command, use a sub-command to perform an operation",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "Missing subcommand, see -h")
	},
}

var coSubCmd = &cobra.Command{
	Use:     "co <repo>",
	Aliases: []string{"checkout"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Make a local copy of a github repo from the TykTechnologies org",
	Long: `Uses git.prefix from viper to construct the fully qualified repo name. Changes can be made in this clone and pushed. This command is equivalent to:
git clone <git.prefix>/<repo> <dir>
cd <dir>; git checkout <branch>`,
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = repo
		}
		_, err := git.Init(fmt.Sprintf("%s/%s", viper.Get("git.prefix"), repo),
			Branch,
			1,
			dir,
			os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			cmd.Println(err)
		}
	},
}

func init() {
	gitCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")

	coSubCmd.Flags().String("dir", "", "Directory to check out into, default: <repo>")
	gitCmd.AddCommand(coSubCmd)
	rootCmd.AddCommand(gitCmd)
}
