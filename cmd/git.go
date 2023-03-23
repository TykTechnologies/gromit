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
	Use:   "push <dir> <repo> <remote branch> <bundle>",
	Args:  cobra.MinimumNArgs(4),
	Short: "Commit, push and create a PR from a local git repo in <dir>",
	Long: `Uses git.prefix from viper to construct the fully qualified repo name. Any changes will be committed. This command is equivalent to:
cd <dir>
git commit -m <msg>
git push origin
gh pr create`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		repo := args[1]
		remoteBranch := args[2]
		bundle := args[3]

		owner, _ := cmd.Flags().GetString("owner")
		r, err := git.Init(repo,
			owner,
			Branch,
			1,
			dir,
			os.Getenv("GITHUB_TOKEN"),
			true)
		if err != nil {
			return fmt.Errorf("git init %s ./%s: %v", repo, dir, err)
		}
		force, _ := cmd.Flags().GetBool("force")
		dfs, err := git.NonTrivial(dir)
		if err != nil {
			return fmt.Errorf("computing diff in %s: %v", dir, err)
		}
		if len(dfs) == 0 && !force {
			cmd.Printf("trivial changes for repo %s branch %s, stopping here", repo, r.Branch())
			return nil
		}
		if len(dfs) > 0 {
			msg, _ := cmd.Flags().GetString("msg")
			err = r.Commit(msg)
			if err != nil {
				return fmt.Errorf("git commit %s ./%s: %v", repo, dir, err)
			}
		}
		err = r.Push(remoteBranch)
		if err != nil {
			return fmt.Errorf("git push %s %s:%s: %v", repo, r.Branch(), remoteBranch, err)
		}
		pr, _ := cmd.Flags().GetBool("pr")
		if pr {
			title, _ := cmd.Flags().GetString("title")
			pr, err := r.CreatePR(title, remoteBranch, bundle)
			if err != nil {
				return fmt.Errorf("gh create pr --base %s --head %s: %v", r.Branch(), remoteBranch, err)
			}
			cmd.Println(pr)
		}
		return nil
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
			os.Getenv("GITHUB_TOKEN"),
			true)
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

var printPolicySubCmd = &cobra.Command{
	Use:   "print-policy <repo>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Prints the git branches policy for the given repo",
	Long:  "Dumps a markdown formatted output containing all the git release branches information for the given repo.",
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		owner, _ := cmd.Flags().GetString("owner")
		r, err := git.Init(repo,
			owner,
			Branch,
			1,
			os.TempDir(),
			os.Getenv("GITHUB_TOKEN"),
			false)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error init git repo: ", err)
			os.Exit(1)
		}
		buf, err := r.RenderPRBundle("policy-dump")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error rendering policy-dump: ", err)
			os.Exit(1)
		}
		_, err = buf.WriteTo(os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error displaying rendered tmpl: ", err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

func init() {
	gitCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	gitCmd.PersistentFlags().StringVar(&Owner, "owner", "TykTechnologies", "Github org")

	coSubCmd.Flags().String("dir", "", "Directory to check out into, default: <repo>")
	pushSubCmd.Flags().String("msg", "automated push by gromit", "Commit message")
	pushSubCmd.Flags().Bool("pr", false, "Create PR")
	pushSubCmd.Flags().Bool("force", false, "Proceed even if there are only trivial changes")
	pushSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	pushSubCmd.MarkFlagsRequiredTogether("pr", "title")

	gitCmd.AddCommand(coSubCmd)
	gitCmd.AddCommand(diffSubCmd)
	gitCmd.AddCommand(pushSubCmd)
	gitCmd.AddCommand(printPolicySubCmd)
	rootCmd.AddCommand(gitCmd)
}
