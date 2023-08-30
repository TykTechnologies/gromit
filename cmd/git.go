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
	"os"

	"github.com/TykTechnologies/gromit/git"
	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git <sub command>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Top-level git command, use a sub-command to perform an operation",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "Missing subcommand, see -h")
	},
}

var pushSubCmd = &cobra.Command{
	Use:   "push --branch <local-branch> <dir> <repo> <remote branch>",
	Args:  cobra.MinimumNArgs(3),
	Short: "Commit, push and create a PR from a local git repo in <dir>",
	Long: `Any changes will be committed. This command is equivalent to:
cd <dir>
git push origin local-branch:remote-branch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		repo := args[1]
		remoteBranch := args[2]

		owner, _ := cmd.Flags().GetString("owner")
		r, err := git.Init(repo,
			owner,
			Branch,
			1,
			dir,
			os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			return fmt.Errorf("git init %s ./%s: %v", repo, dir, err)
		}
		err = r.Push(remoteBranch)
		if err != nil {
			return fmt.Errorf("git push %s %s:%s: %v", repo, r.Branch(), remoteBranch, err)
		}
		return nil
	},
}

var coSubCmd = &cobra.Command{
	Use:     "co <repo>",
	Aliases: []string{"checkout"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Make a local copy of a github repo from the TykTechnologies org",
	Long: `Changes can be made in this clone and pushed. This command is equivalent to:
git clone <git.prefix>/<repo> <dir>
cd <dir>; git checkout <branch>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = repo
		}
		r, err := git.Init(repo,
			Owner,
			Branch,
			1,
			dir,
			os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			return fmt.Errorf("git init %s: %v", repo, err)
		}
		return r.Checkout(Branch)
	},
}

var diffSubCmd = &cobra.Command{
	Use:   "diff <dir>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Compute if there are differences worth pushing (requires git)",
	Long:  `Parses the output of git diff --staged -G'(^[^#])' to make a decision. Fails if there are non-trivial diffs, or if there was a problem. This failure mode is chosen so that it can work as a gate.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		dfs, err := git.NonTrivial(dir)
		if len(dfs) > 0 {
			return fmt.Errorf("non-trivial diffs in %s: %v", dir, dfs)
		}
		return err
	},
}

func init() {
	gitCmd.PersistentFlags().StringVar(&Branch, "branch", "", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	gitCmd.PersistentFlags().StringVar(&Owner, "owner", "TykTechnologies", "Github org")

	coSubCmd.Flags().String("dir", "", "Directory to check out into, default: <repo>")
	pushSubCmd.Flags().String("msg", "automated push by gromit", "Commit message")
	pushSubCmd.Flags().Bool("force", false, "Proceed even if there are only trivial changes")

	gitCmd.AddCommand(coSubCmd)
	gitCmd.AddCommand(diffSubCmd)
	gitCmd.AddCommand(pushSubCmd)
	rootCmd.AddCommand(gitCmd)
}
