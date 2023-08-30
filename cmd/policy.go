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

	"time"

	"github.com/TykTechnologies/gromit/git"
	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var dryRun, autoMerge bool
var configPolicies policy.Policies

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Templatised policies that are driven by the config file",
	Long:  `Policies are driven by a config file. The config file models the variables of all the repositories under management. See https://github.com/TykTechnologies/gromit/tree/master/policy/config.yaml.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(configPolicies)
	},
}

// syncSubCmd generates a set of files in an in-memory git repo and pushes it to origin.
var syncSubCmd = &cobra.Command{
	Use:   "sync <repo>",
	Args:  cobra.MinimumNArgs(2),
	Short: "(re-)generate the templates for <repo> given <branch> and update git",
	Long: `Operates directly on github and creates PRs for protected branches. Requires an OAuth2 token (for private repos) and a section in the config file describing the policy. Will render templates, overlaid onto a git repo. 
A PR will be created with the changes and @devops will be asked for a review.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo = args[0]
		branch, _ := cmd.Flags().GetString("branch")
		// Checkout code into a dir named repo
		r, err := git.Init(repo,
			Owner,
			Branch,
			1,
			repo,
			os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			return fmt.Errorf("git init %s: %v", repo, err)
		}
		err = r.Checkout(branch)
		if err != nil {
			cmd.Printf("git checkout %s:%s: %v", repo, branch, err)
		}
		rp, err := configPolicies.GetRepoPolicy(repo, branch)
		rp.SetTimestamp(time.Now().UTC())
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repo, err)
		}
		b, err := policy.NewBundle(rp.Branchvals.Features)
		if err != nil {
			return fmt.Errorf("bundle %v: %v", rp.Branchvals.Features, err)
		}
		// Generate bundle into the dir named repo from above
		files, err := b.Render(&rp, repo, nil)
		log.Info().Strs("files", files).Msg("Rendered files")
		if err != nil {
			return fmt.Errorf("bundle gen %v: %v", rp.Branchvals.Features, err)
		}
		force, _ := cmd.Flags().GetBool("force")
		dfs, err := git.NonTrivial(repo)
		if err != nil {
			return fmt.Errorf("computing diff in %s: %v", repo, err)
		}
		if len(dfs) == 0 && !force {
			cmd.Printf("trivial changes for repo %s branch %s, stopping here", repo, r.Branch())
			return nil
		}

		// Add rendered files to git staging.
		for _, f := range files {
			_, err := r.AddFile(f)
			if err != nil {
				return fmt.Errorf("staging file to git worktree: %v", err)
			}
		}

		if len(dfs) > 0 {
			msg, _ := cmd.Flags().GetString("msg")
			err = r.Commit(msg)
			if err != nil {
				return fmt.Errorf("git commit %s ./%s: %v", repo, repo, err)
			}
		}
		remoteBranch, _ := cmd.Flags().GetString("remotebranch")
		err = r.Push(remoteBranch)
		if err != nil {
			return fmt.Errorf("git push %s %s:%s: %v", repo, r.Branch(), remoteBranch, err)
		}
		pr, _ := cmd.Flags().GetBool("pr")
		if pr {
			title, _ := cmd.Flags().GetString("title")
			draft, _ := cmd.Flags().GetBool("draft")
			pr, err := r.CreatePR(title, remoteBranch, draft)
			if err != nil {
				return fmt.Errorf("gh create pr --base %s --head %s: %v", r.Branch(), remoteBranch, err)
			}
			cmd.Println("PR created: ", *pr.HTMLURL)
			var auto bool
			if auto, err = cmd.Flags().GetBool("auto"); err == nil && auto {
				return r.EnableAutoMerge(pr.GetNodeID())
			}
			return err
		}
		return nil
	},
}

// docSubCmd represents the doctor subcommand
var docSubCmd = &cobra.Command{
	Use:     "doctor <repo>",
	Aliases: []string{"doc", "fix"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Diagnose problems with the release engineering code",
	Long: `For the supplied repo, for all branches known to gromit, generate and apply bundles that are appropriate.
Then test each repo for non-trivial diffs.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("not implemented")
	},
}

func init() {
	syncSubCmd.Flags().String("remotebranch", "", "The branch that will be used for creating the PR - this is the branch that gets pushed to remote")
	syncSubCmd.Flags().Bool("pr", false, "Create PR")
	syncSubCmd.Flags().Bool("draft", false, "The created PR will be in this mode")
	syncSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	syncSubCmd.Flags().String("msg", "Auto generated from templates by gromit", "Commit message for the automated commit by gromit.")
	syncSubCmd.MarkFlagRequired("remotebranch")
	syncSubCmd.MarkFlagRequired("branch")
	syncSubCmd.MarkFlagsRequiredTogether("pr", "title")
	syncSubCmd.PersistentFlags().StringVar(&Owner, "owner", "TykTechnologies", "Github org")
	syncSubCmd.Flags().Bool("force", false, "Proceed even if there are only trivial changes")

	policyCmd.AddCommand(syncSubCmd)

	docSubCmd.Flags().String("pattern", "^(release-[[:digit:].]+|master)", "Regexp to match release engineering branches")
	policyCmd.AddCommand(docSubCmd)

	policyCmd.PersistentFlags().StringSliceVar(&Repos, "repos", Repos, "Repos to operate upon, comma separated values accepted.")
	policyCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	policyCmd.PersistentFlags().BoolVarP(&autoMerge, "auto", "a", true, "Will automerge if all requirements are meet")
	rootCmd.AddCommand(policyCmd)
}
