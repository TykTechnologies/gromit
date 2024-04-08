/*
Copyright © 2024 Tyk Technologies

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

	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var gh *policy.GithubClient

// PrBranch so that it does not conflict with PolBranch
var PrBranch string

// prsCmd represents the prs command
var prsCmd = &cobra.Command{
	Use:   "prs <action> <repos>...",
	Short: "Operate upon PRs for the named repos",
	Long:  `These commands do not need a git repo. They does require GITHUB_TOKEN to be set.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
		ghToken := os.Getenv("GITHUB_TOKEN")
		if ghToken == "" {
			log.Fatal().Msg("Working with PRs requires GITHUB_TOKEN")
		}
		gh = policy.NewGithubClient(ghToken)
	},
}

var cprSubCmd = &cobra.Command{
	Use:     "createprs repos...",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"cpr"},
	Short:   "Create PRs for the named repos",
	Long: `PRs will be created for the branches with <prefix>. Existing PRs will be not cause duplicates.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jToken := os.Getenv("JIRA_TOKEN")
		jUser := os.Getenv("JIRA_USER")
		if jToken == "" || jUser == "" {
			log.Fatal().Msg("Working with PRs requires GITHUB_TOKEN")
		}
		j := policy.NewJiraClient(jUser, jToken)
		issue, err := cmd.Flags().GetString("jira")
		if err != nil {
			log.Fatal().Err(err).Msgf("could not get Jira issue %s", issue)
		}
		jIssue, err := j.GetIssue(issue)
		if err != nil {
			log.Fatal().Err(err).Msgf("could not get Jira issue %s", issue)
		}
		autoMerge, _ := cmd.Flags().GetBool("auto")
		var prs []string
		for _, repoName := range args {
			rp, err := configPolicies.GetRepoPolicy(repoName)
			if err != nil {
				return fmt.Errorf("repopolicy %s: %v", repoName, err)
			}
			var branches []string
			if PrBranch == "" {
				branches = rp.GetAllBranches()
			} else {
				branches = []string{PrBranch}
			}
			for _, branch := range branches {
				prOpts := &policy.PullRequest{
					Jira:       jIssue,
					BaseBranch: branch,
					PrBranch:   Prefix + branch,
					Owner:      Owner,
					Repo:       repoName,
					AutoMerge:  autoMerge,
				}
				pr, err := gh.CreatePR(rp, prOpts)
				if err != nil {
					cmd.Printf("Could not create PR for %s:%s: %v", repoName, branch, err)
				}
				prs = append(prs, *pr.HTMLURL)
			}
		}
		cmd.Println("PRs created:")
		for _, pr := range prs {
			cmd.Printf("- %s\n", pr)
		}
		return nil
	},
}

var dprSubCmd = &cobra.Command{
	Use:     "deleteprs repos...",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"dpr"},
	Short:   "Close PRs for the named repos",
	Long: `For each of the supplied repos, PRs will be closed without merging.
This command does not need a git repo. It does require GITHUB_TOKEN to be set.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, repoName := range args {
			err := processRepo(repoName, gh.ClosePR)
			if err != nil {
				cmd.Printf("Could not delete PR for %s: %v", repoName, err)
			}
		}
	},
}

var uprSubCmd = &cobra.Command{
	Use:     "updateprs repos...",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"upr"},
	Short:   "Update the releng PR branch for the named repos",
	Long: `For each of the supplied repos, trigger a Github managed update of the PR branch. This will fail if there are conflicts.
This command does not need a git repo. It does require GITHUB_TOKEN to be set.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, repoName := range args {
			err := processRepo(repoName, gh.UpdatePrBranch)
			if err != nil {
				cmd.Printf("Could not update PR branch for %s: %v", repoName, err)
			}
		}
	},
}

var oprSubCmd = &cobra.Command{
	Use:     "openprs repos...",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"opr"},
	Short:   "Open the releng PR in the default browser",
	Long: `For each of the supplied repos, trigger a Github managed update of the PR branch. This will fail if there are conflicts.
This command does not need a git repo. It does require GITHUB_TOKEN to be set.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, repoName := range args {
			err := processRepo(repoName, gh.Open)
			if err != nil {
				cmd.Printf("Could not open PR for %s: %v", repoName, err)
			}
		}
	},
}

// processRepo abstracts a simple flow for a repo
func processRepo(repoName string, f func(*policy.PullRequest) error) error {
	rp, err := configPolicies.GetRepoPolicy(repoName)
	if err != nil {
		return fmt.Errorf("repopolicy %s: %v", repoName, err)
	}
	var branches []string
	if PrBranch == "" {
		branches = rp.GetAllBranches()
	} else {
		branches = []string{PrBranch}
	}
	for _, branch := range branches {
		prOpts := &policy.PullRequest{
			BaseBranch: branch,
			PrBranch:   Prefix + branch,
			Owner:      Owner,
			Repo:       repoName,
		}
		err := f(prOpts)
		if err != nil {
			fmt.Printf("Could not operate on PR for %s:%s: %v\n", repoName, branch, err)
		}
	}
	return nil
}

func init() {
	prsCmd.PersistentFlags().StringVar(&PrBranch, "branch", "", "Restrict operations to this branch, if not set all branches defined int he config will be processed.")
	prsCmd.PersistentFlags().StringVar(&Prefix, "prefix", "releng/", "Given the base branch from --branch, the head branch will be assumed to be <prefix><branch>")

	cprSubCmd.Flags().Bool("auto", true, "Will automerge if all requirements are meet")
	cprSubCmd.Flags().String("jira", "", "Title and body will be filled in from Jira issue")
	cprSubCmd.MarkFlagRequired("jira")

	prsCmd.AddCommand(cprSubCmd)
	prsCmd.AddCommand(dprSubCmd)
	prsCmd.AddCommand(uprSubCmd)
	prsCmd.AddCommand(oprSubCmd)

	rootCmd.AddCommand(prsCmd)
}
