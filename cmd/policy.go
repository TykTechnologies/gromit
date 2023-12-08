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

	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var dryRun, autoMerge bool
var configPolicies policy.Policies
var owner string

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
	Args:  cobra.MinimumNArgs(1),
	Short: "(re-)generate the templates for all known branches for <repo>",
	Long: `Operates directly on github and creates PRs. Requires an OAuth2 token (for private repos) and a section in the config file describing the policy. Will render templates, overlaid onto a git repo. 
A PR will be created with the changes and @devops will be asked for a review.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pr, _ := cmd.Flags().GetBool("pr")
		ghToken := os.Getenv("GITHUB_TOKEN")
		if pr && ghToken == "" {
			return fmt.Errorf("Creating a PR requires GITHUB_TOKEN to be set")
		}
		repoName := args[0]
		// Checkout code into a dir named repo
		repo, err := policy.InitGit(repoName,
			owner,
			Branch,
			repoName,
			ghToken)
		if err != nil {
			return fmt.Errorf("git init %s: %v, is the repo private and GITHUB_TOKEN not set?", repoName, err)
		}
		rp, err := configPolicies.GetRepoPolicy(repoName)
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repoName, err)
		}
		rp.SetTimestamp(time.Now().UTC())
		title, _ := cmd.Flags().GetString("title")
		msg, _ := cmd.Flags().GetString("msg")
		autoMerge, _ := cmd.Flags().GetBool("auto")

		var prs, branches []string
		if Branch == "" {
			branches = rp.GetAllBranches()
		} else {
			branches = []string{Branch}
		}

		for _, branch := range branches {
			remoteBranch, err := rp.ProcessBranch(repoName, branch, msg, repo)
			if err != nil {
				cmd.Printf("Could not process %s/%s: %v\n", repoName, branch, err)
				cmd.Println("Will not process remaining branches")
				break
			}
			if pr {
				pr, err := repo.CreatePR(rp, title, remoteBranch, false)
				if err != nil {
					cmd.Printf("gh create pr --base %s --head %s: %v", repo.Branch(), remoteBranch, err)
				}
				prs = append(prs, *pr.HTMLURL)
				if autoMerge {
					err = repo.EnableAutoMerge(pr.GetNodeID())
					if err != nil {
						cmd.Printf("Failed to enable auto-merge for %s: %v", *pr.HTMLURL, err)
					}
				}
			}
		}
		cmd.Printf("PRs created: %v\n", prs)
		return nil
	},
}

var diffSubCmd = &cobra.Command{
	Use:   "diff <dir>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Compute if there are differences worth pushing (requires git)",
	Long:  `Parses the output of git diff --staged -G'(^[^#])' to make a decision. Fails if there are non-trivial diffs, or if there was a problem. This failure mode is chosen so that it can work as a gate.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		colours, _ := cmd.Flags().GetBool("colours")
		dfs, err := policy.NonTrivialDiff(dir, colours)
		if len(dfs) > 0 {
			return fmt.Errorf("non-trivial diffs in %s: %v", dir, dfs)
		}
		return err
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
	syncSubCmd.Flags().Bool("pr", false, "Create PR")
	syncSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	syncSubCmd.Flags().String("msg", "Auto generated from templates by gromit", "Commit message for the automated commit by gromit.")
	syncSubCmd.MarkFlagsRequiredTogether("pr", "title")
	syncSubCmd.PersistentFlags().StringVar(&owner, "owner", "TykTechnologies", "Github org")

	diffSubCmd.Flags().Bool("colours", true, "Use colours in output")

	policyCmd.AddCommand(syncSubCmd)
	policyCmd.AddCommand(diffSubCmd)
	policyCmd.AddCommand(docSubCmd)

	policyCmd.PersistentFlags().StringSliceVar(&Repos, "repos", Repos, "Repos to operate upon, comma separated values accepted.")
	// FIXME: Remove the default from Branch when we can process multiple branches in the same dir
	policyCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	policyCmd.PersistentFlags().BoolVarP(&autoMerge, "auto", "a", true, "Will automerge if all requirements are meet")
	rootCmd.AddCommand(policyCmd)
}
