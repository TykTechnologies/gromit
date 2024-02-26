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
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var dryRun, autoMerge bool
var owner string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Templatised policies that are driven by the config file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("You probably need to use a sub-command.")
	},
}

// controllerSubCmd is used at runtime in release.yml:test-controller
var controllerSubCmd = &cobra.Command{
	Use:   "controller",
	Short: "Decide the test environment",
	Long:  `Based on the environment variables "REPO", "TAGS", "BASE_REF", "IS_PR", "IS_TAG" writes the github outputs required to run release.yml:api-tests`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Since IS_PR and IS_LTS can both be true, having IS_LTS last sets the trigger correctly
		params := policy.NewParams("REPO", "TAGS", "BASE_REF", "IS_PR", "IS_TAG")
		var op bytes.Buffer
		if err := params.SetVersions(&op); err != nil {
			return err
		}
		op.WriteString("\n")

		// conf is the set of configuration variations
		// db is the databases to use
		// pump/sink are included only when needed
		defaults := policy.TestVariations{
			"conf": []string{"sha256", "murmur64"},
			"db":   []string{"mongo44", "postgres15"},
			"pump": []string{"tykio/tyk-pump-docker-pub:v1.8", "$ECR/tyk-pump:master"},
			"sink": []string{"tykio/tyk-mdcb-docker:v2.4", "$ECR/tyk-sink:master"},
		}
		if err := params.SetVariations(&op, defaults); err != nil {
			return err
		}

		_, err := op.WriteTo(os.Stdout)
		return err
	},
}

// syncSubCmd generates a set of files in an in-memory git repo and pushes it to origin.
var syncSubCmd = &cobra.Command{
	Use:   "sync <repo>",
	Args:  cobra.MinimumNArgs(1),
	Short: "(re-)generate the templates for all known branches for <repo>",
	Long: `Policies are driven by a config file. The config file models the variables of all the repositories under management. See https://github.com/TykTechnologies/gromit/tree/master/policy/config.yaml.
Operates directly on github and creates PRs. Requires an OAuth2 token (for private repos) and a section in the config file describing the policy. Will render templates, overlaid onto a git repo.
A PR will be created with the changes and @devops will be asked for a review.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var configPolicies policy.Policies
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}

		pr, _ := cmd.Flags().GetBool("pr")
		ghToken := os.Getenv("GITHUB_TOKEN")
		if pr && ghToken == "" {
			return fmt.Errorf("Creating a PR requires GITHUB_TOKEN to be set")
		}
		repoName := args[0]
		// Checkout code into a dir named repo
		repo, err := policy.InitGit(fmt.Sprintf("https://github.com/%s/%s", owner, repoName),
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

		var gh *policy.GithubClient
		if pr {
			gh = policy.NewGithubClient(ghToken)
		}

		for _, branch := range branches {
			remoteBranch, err := rp.ProcessBranch(repoName, branch, msg, repo)
			if err != nil {
				cmd.Printf("Could not process %s/%s: %v\n", repoName, branch, err)
				cmd.Println("Will not process remaining branches")
				break
			}
			if pr {
				prOpts := &policy.PullRequest{
					Title:      title,
					BaseBranch: repo.Branch(),
					PrBranch:   remoteBranch,
					Owner:      owner,
					Repo:       repoName,
				}
				pr, err := gh.CreatePR(rp, prOpts)
				if err != nil {
					cmd.Printf("gh create pr --base %s --head %s: %v", repo.Branch(), remoteBranch, err)
				}
				prs = append(prs, *pr.HTMLURL)
				if autoMerge {
					err = gh.EnableAutoMerge(pr.GetNodeID())
					if err != nil {
						cmd.Printf("Failed to enable auto-merge for %s: %v", *pr.HTMLURL, err)
					}
				}
			}
		}
		cmd.Printf("PRs created: %v\n", prs)
		return err
	},
}

var prSubCmd = &cobra.Command{
	Use:   "pr repos...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Create PRs for the named repos",
	Long: `For each of the supplied repos, PRs will created for the releng/* branches. These branches are kept updated by Dr. Releng which updates these branches when gromit:master is updated.
This command does not need a git repo. It does require GITHUB_TOKEN to be set.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var configPolicies policy.Policies
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			return fmt.Errorf("Could not parse repo policies: %v", err)
		}
		ghToken := os.Getenv("GITHUB_TOKEN")
		if ghToken == "" {
			return fmt.Errorf("Creating PRs requires GITHUB_TOKEN to be set")
		}
		title, _ := cmd.Flags().GetString("title")
		autoMerge, _ := cmd.Flags().GetBool("auto")
		var prs []string
		gh := policy.NewGithubClient(ghToken)
		for _, repoName := range args {
			rp, err := configPolicies.GetRepoPolicy(repoName)
			if err != nil {
				return fmt.Errorf("repopolicy %s: %v", repoName, err)
			}
			var branches []string
			if Branch == "" {
				branches = rp.GetAllBranches()
			} else {
				branches = []string{Branch}
			}
			for _, branch := range branches {
				prOpts := &policy.PullRequest{
					Title:      fmt.Sprintf("%s:%s %s", repoName, branch, title),
					BaseBranch: branch,
					PrBranch:   "releng/" + branch,
					Owner:      owner,
					Repo:       repoName,
				}
				pr, err := gh.CreatePR(rp, prOpts)
				if err != nil {
					cmd.Printf("Could not create PR for %s:%s: %v", repoName, branch, err)
				}
				prs = append(prs, *pr.HTMLURL)
				if autoMerge {
					err = gh.EnableAutoMerge(pr.GetNodeID())
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
		dfs, err := policy.NonTrivialDiff(dir, true, colours)
		if len(dfs) > 0 {
			return fmt.Errorf("non-trivial diffs in %s: %v", dir, dfs)
		}
		return err
	},
}

func init() {
	syncSubCmd.Flags().Bool("pr", false, "Create PR")
	syncSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	syncSubCmd.Flags().String("msg", "Auto generated from templates by gromit", "Commit message for the automated commit by gromit.")
	syncSubCmd.MarkFlagsRequiredTogether("pr", "title")
	syncSubCmd.PersistentFlags().StringVar(&owner, "owner", "TykTechnologies", "Github org")

	prSubCmd.Flags().String("title", "", "Title of PR, template interpolation from RepoPolicy allowed")

	diffSubCmd.Flags().Bool("colours", true, "Use colours in output")

	policyCmd.AddCommand(syncSubCmd)
	policyCmd.AddCommand(controllerSubCmd)
	policyCmd.AddCommand(diffSubCmd)
	policyCmd.AddCommand(prSubCmd)

	policyCmd.PersistentFlags().StringSliceVar(&Repos, "repos", Repos, "Repos to operate upon, comma separated values accepted.")
	// FIXME: Remove the default from Branch when we can process multiple branches in the same dir
	policyCmd.PersistentFlags().StringVar(&Branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	policyCmd.PersistentFlags().BoolVarP(&autoMerge, "auto", "a", true, "Will automerge if all requirements are meet")
	rootCmd.AddCommand(policyCmd)
}
