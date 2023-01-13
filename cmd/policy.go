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
	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var dryRun, autoMerge bool

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Templatised policies that are driven by the config file",
	Long:  `Policies are driven by a config file. The config file models the variables of all the repositories under management. See https://github.com/TykTechnologies/gromit/tree/master/policy/config.yaml.`,
	Run: func(cmd *cobra.Command, args []string) {
		var configPolicies policy.Policies
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
		cmd.Println(configPolicies)
	},
}

// syncSubCmd generates a set of files in an in-memory git repo and pushes it to origin.
var syncSubCmd = &cobra.Command{
	Use:   "sync <bundle> <repo>",
	Args:  cobra.MinimumNArgs(2),
	Short: "(re-)generate the template <bundle> and update git",
	Long: `Operates directly on github and creates PRs for protected branches. Requires an OAuth2 token (for private repos) and a section in the config file describing the policy. Will render templates, overlaid onto a git repo. 
If the branch is marked protected in the repo policies, a draft PR will be created with the changes and @devops will be asked for a review.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		bundle = args[0]
		repo = args[1]
		branch, _ := cmd.Flags().GetString("branch")
		pr, _ := cmd.Flags().GetBool("pr")
		// Checkout code into a dir named repo
		r, err := git.Init(repo, Owner,
			Branch,
			1,
			repo,
			os.Getenv("GITHUB_TOKEN"), true)
		if err != nil {
			return fmt.Errorf("git init %s: %v", repo, err)
		}
		err = r.Checkout(branch)
		if err != nil {
			cmd.Printf("git checkout %s:%s: %v", repo, branch, err)
		}
		b, err := policy.NewBundle(bundle)
		if err != nil {
			return fmt.Errorf("bundle %s: %v", bundle, err)
		}
		rp, err := policy.GetRepoPolicy(repo)
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repo, err)
		}
		// Generate bundle into the dir named repo from above
		err = b.Render(&rp, repo, nil, r)
		if err != nil {
			return fmt.Errorf("bundle gen %s: %v", bundle, err)
		}
		dfs, err := git.NonTrivial(repo, pr)
		if len(dfs) == 0 {
			cmd.Printf("trivial changes for repo %s branch %s, stopping here", repo, r.Branch())
			return nil
		}
		err = r.Commit(fmt.Sprintf("gromit policy sync for %s", bundle))
		if err != nil {
			return fmt.Errorf("commit: %v", err)
		}
		remoteBranch, _ := cmd.Flags().GetString("remotebranch")
		err = r.Push(remoteBranch)
		if err != nil {
			return fmt.Errorf("git push %s %s:%s: %v", repo, r.Branch(), remoteBranch, err)
		}
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
	syncSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	syncSubCmd.MarkFlagRequired("remotebranch")
	syncSubCmd.MarkFlagRequired("branch")
	syncSubCmd.MarkFlagsRequiredTogether("pr", "title")
	syncSubCmd.PersistentFlags().StringVar(&Owner, "owner", "TykTechnologies", "Github org")

	policyCmd.AddCommand(syncSubCmd)

	docSubCmd.Flags().String("pattern", "^(release-[[:digit:].]+|master)", "Regexp to match release engineering branches")
	policyCmd.AddCommand(docSubCmd)

	policyCmd.PersistentFlags().StringSliceVar(&Repos, "repos", Repos, "Repos to operate upon, comma separated values accepted.")
	policyCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	policyCmd.PersistentFlags().BoolVarP(&autoMerge, "auto", "a", true, "Will automerge if all requirements are meet")
	rootCmd.AddCommand(policyCmd)
}
