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

// PolBranch so that it does not conflict with PrBranch
var PolBranch string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Templatised policies that are driven by the config file",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("You need to use a sub-command.")
	},
}

// genSubCmd is meant for debugging
var genSubCmd = &cobra.Command{
	Use:     "gen <dir>",
	Aliases: []string{"generate", "render"},
	Args:    cobra.ExactArgs(1),
	Short:   "Render <bundle> into <dir> using parameters from policy.<repo>. If <dir> is -, render to stdout.",
	Long: `A bundle is a collection of templates. A template is a top-level file which will be rendered with the same path as it is embedded as.
A template can have sub-templates which are in directories of the form, <template>.d. The contents of these directories will not be traversed looking for further templates but are collected into the list of files that used to instantiate <template>.
Templates can be organised into features, which is just a directory tree of templates. Rendering the same file from different templates is _not_ supported.
This command does not overlay the rendered output into a git tree. You will have to checkout the repo yourself if you want to check the rendered templates into a git repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		repoName, _ := cmd.Flags().GetString("repo")
		rp, err := configPolicies.GetRepoPolicy(repoName)
		rp.SetTimestamp(time.Now().UTC())
		rp.SetBranch(PolBranch)
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

// controllerSubCmd is used at runtime in release.yml:test-controller
var controllerSubCmd = &cobra.Command{
	Use:   "controller",
	Short: "Decide the test environment",
	Long:  `Based on the environment variables "JOB","REPO", "TAGS", "BASE_REF", "IS_PR", "IS_TAG" writes the github outputs required to run release.yml:api-tests`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Since IS_PR and IS_LTS can both be true, having IS_LTS last sets the trigger correctly
		params := policy.NewParams("JOB", "REPO", "TAGS", "BASE_REF", "IS_PR", "IS_TAG")
		var op bytes.Buffer
		if err := params.SetVersions(&op); err != nil {
			return err
		}
		op.WriteString("\n")

		// conf is the set of configuration variations
		// db is the databases to use
		// pump/sink are included only when needed
		defaults := policy.GHoutput{
			TestVariations: map[string][]string{
				params["job"] + "_conf":     {"sha256", "murmur128"},
				params["job"] + "_db":       {"mongo7", "postgres15"},
				params["job"] + "_cache_db": {"redis7"},
				"pump":                      {"tykio/tyk-pump-docker-pub:v1.8", "$ECR/tyk-pump:master"},
				"sink":                      {"tykio/tyk-mdcb-docker:v2.4", "$ECR/tyk-sink:master"},
			},
			Exclusions: []map[string]string{
				{"pump": "tykio/tyk-pump-docker-pub:v1.8", "sink": "$ECR/tyk-sink:master"},
				{"pump": "$ECR/tyk-pump:master", "sink": "tykio/tyk-mdcb-docker:v2.4"},
				{"db": "mongo7", "conf": "murmur128"},
				{"db": "postgres15", "conf": "sha256"},
			},
		}
		if err := params.SetOutputs(&op, defaults); err != nil {
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
If --pr is supplied, a PR will be created with the changes and @devops will be asked for a review.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pr, _ := cmd.Flags().GetBool("pr")
		ghToken := os.Getenv("GITHUB_TOKEN")
		if pr && ghToken == "" {
			return fmt.Errorf("Creating a PR requires GITHUB_TOKEN to be set")
		}
		repoName := args[0]
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			return fmt.Errorf("Could not load config file: %v", err)
		}
		rp, err := configPolicies.GetRepoPolicy(repoName)
		if err != nil {
			return fmt.Errorf("repopolicy %s: %v", repoName, err)
		}
		rp.SetTimestamp(time.Now().UTC())
		msg, _ := cmd.Flags().GetString("msg")
		autoMerge, _ := cmd.Flags().GetBool("auto")

		var prs, branches []string
		if PolBranch == "" {
			branches = rp.GetAllBranches()
		} else {
			branches = []string{PolBranch}
		}

		if pr {
			gh = policy.NewGithubClient(ghToken)
		}

		for _, branch := range branches {
			repo, err := policy.InitGit(fmt.Sprintf("https://github.com/%s/%s", Owner, repoName),
				branch,
				repoName,
				ghToken)
			if err != nil {
				return fmt.Errorf("git init %s: %v, is the repo private and GITHUB_TOKEN not set?", repoName, err)
			}
			pushOpts := &policy.PushOptions{
				OpDir:        repoName,
				Branch:       branch,
				RemoteBranch: Prefix + branch,
				CommitMsg:    msg,
				Repo:         repo,
			}
			err = rp.ProcessBranch(pushOpts)
			if err != nil {
				cmd.Printf("Could not process %s/%s: %v\n", repoName, branch, err)
				cmd.Println("Will not process remaining branches")
				break
			}
			if pr {
				prOpts := &policy.PullRequest{
					BaseBranch: repo.Branch(),
					PrBranch:   pushOpts.RemoteBranch,
					Owner:      Owner,
					Repo:       repoName,
					AutoMerge:  autoMerge,
				}
				pr, err := gh.CreatePR(rp, prOpts)
				if err != nil {
					cmd.Printf("gh create pr --base %s --head %s: %v", repo.Branch(), pushOpts.RemoteBranch, err)
				}
				prs = append(prs, *pr.HTMLURL)
			}
		}
		cmd.Println("PRs created:")
		for _, pr := range prs {
			cmd.Printf("- %s\n", pr)
		}
		return err
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

var matchSubCmd = &cobra.Command{
	Use:   "match <current_tag> <target_tag>",
	Args:  cobra.MinimumNArgs(2),
	Short: "Given the current build tag and the target tag, find the matching tags in the repos",
	Long:  `Find matching tags from gw, dash, pump and sink. The current tag is passed straigth as override image to be used by the test, but the target tag is used to find the matching tags for the other repos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dcFile, _ := cmd.Flags().GetString("config")
		config, err := policy.NewDockerAuths(dcFile)
		if err != nil {
			return err
		}

		repos := []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink"}
		tagOverride := args[0]
		tagMatch := args[1]

		p := policy.ParseImageName(tagMatch)
		o := policy.ParseImageName(tagOverride)

		matches, err := config.GetMatches(p.Registry, p.Tag, repos)
		if err != nil {
			log.Warn().Err(err).Msg("looking for matches")
		}

		matches.Repos[o.Repo] = tagOverride

		for _, repo := range repos {
			cmd.Println(matches.Match(repo))
		}
		return nil
	},
}

var serveSubCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the test controller backend",
	Long: `The test controller backend stores the test that are to be run for a specific combination of,
- trigger
- repo
- branch.
This an laternate implementation to the controller which does not embed a server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tvDir, _ := cmd.Flags().GetString("save")
		port, _ := cmd.Flags().GetString("port")
		return policy.Serve(port, tvDir)
	},
}

func init() {
	syncSubCmd.Flags().Bool("pr", false, "Create PR")
	syncSubCmd.Flags().String("title", "", "Title of PR, required if --pr is present")
	syncSubCmd.Flags().String("msg", "Auto generated from templates by gromit", "Commit message for the automated commit by gromit.")
	syncSubCmd.MarkFlagsRequiredTogether("pr", "title")
	syncSubCmd.Flags().StringVar(&Owner, "owner", "TykTechnologies", "Github org")
	syncSubCmd.Flags().StringVar(&Prefix, "prefix", "releng/", "Prefix for the branch with the changes. The default is releng/<branch>")

	diffSubCmd.Flags().Bool("colours", true, "Use colours in output")

	genSubCmd.Flags().String("repo", "", "Repository name to use from config file")

	serveSubCmd.Flags().String("port", ":3000", "Port that the backend will bind to")
	serveSubCmd.Flags().String("save", "testdata/tui", "Test variations are loaded from and saved to this directory")

	matchSubCmd.Flags().String("config", "$HOME/.docker/config.json", "Config file to read authentication token from")

	policyCmd.AddCommand(matchSubCmd)
	policyCmd.AddCommand(syncSubCmd)
	policyCmd.AddCommand(controllerSubCmd)
	policyCmd.AddCommand(diffSubCmd)
	policyCmd.AddCommand(genSubCmd)
	policyCmd.AddCommand(serveSubCmd)

	policyCmd.PersistentFlags().StringVar(&PolBranch, "branch", "", "Restrict operations to this branch, if not set all branches defined int he config will be processed.")
	policyCmd.PersistentFlags().Bool("auto", true, "Will automerge if all requirements are meet")
	rootCmd.AddCommand(policyCmd)
}
