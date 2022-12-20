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
	"encoding/json"
	"os"
	"strings"

	"github.com/TykTechnologies/gromit/config"
	"github.com/TykTechnologies/gromit/policy"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configPolicies policy.Policies
var repos []string
var branch string
var jsonOutput, dryRun, autoMerge, bundle bool
var ghToken string
var prBranch string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Templatised policies that are driven by the config file",
	Long:  `Policies are driven by a config file. The config file models the variables of all the repositories under management. See https://github.com/TykTechnologies/gromit/tree/master/testdata/policies for examples.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&configPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			json, err := json.Marshal(configPolicies)
			if err != nil {
				log.Fatal().Interface("policies", configPolicies).Msg("could not marshal policies into JSON")
			}
			cmd.Println(string(json))
		} else {
			cmd.Println(configPolicies)
		}
		if bundle {
			cmd.Println("Bundle list, directories have a trailing :")
			policy.ListBundles(".")
		}
	},
}

var gpacSubCmd = &cobra.Command{
	Use:     "github",
	Aliases: []string{"gpac"},
	Args:    cobra.MinimumNArgs(0),
	Short:   "Render the github policy as code terraform templates",
	Long: `These manifests control the aspects of a github repo that are relevant to releng and ci. Will locally render terraform manifests from embedded template files using configuration from the config file.
The files are only rendered, applying them requires an org-level PAT. For the purposes of automation this PAT is available as a secret in the gromit repo.`,
	Run: func(cmd *cobra.Command, args []string) {

		rp, err := configPolicies.GetAllRepos(config.RepoURLPrefix)
		if err != nil {
			log.Fatal().Err(err).Strs("repos", repos).Msg("could not aggregate all repos")
		}

		err = policy.RenderBundle("gpac", policy.BundleVars(&rp))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create local dir gpac")
		}
	},
}

// relengSubCmd generates a set of files in an in-memory git repo and pushes it to origin.
var relengSubCmd = &cobra.Command{
	Use:     "releng <commit msg> [commit_msg...]",
	Aliases: []string{"ci", "release"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "(re-)generate the releng template bundle and update git",
	Long: `Controls the automation that is active in each repo for each branch.
Operates directly on github and creates PRs for protected branches. Requires an OAuth2 token and a section in the config file describing the policy. Will render templates, overlaid onto an in-memory/regular filesystem git repo. 
If the branch is marked protected in the repo policies, a draft PR will be created with the changes and @devops will be asked for a review.
`,
	Run: func(cmd *cobra.Command, args []string) {
		commitMsg := strings.Join(args, "\n")
		signingKeyid := viper.GetUint64("signingkey")
		for _, repoName := range repos {
			repo, err := configPolicies.GetRepo(repoName, config.RepoURLPrefix, branch)
			if err != nil {
				log.Fatal().Err(err).Msg("getting repo")
			}
			// use dir as prefix if operating on multiple repos, append the repo
			// name to have different directories for different repos.
			checkoutDir := dir
			if len(repos) > 1 {
				checkoutDir = dir + "-" + repoName
			}
			err = repo.InitGit(1, signingKeyid, checkoutDir, ghToken)
			if err != nil {
				log.Fatal().Err(err).Msg("initialising git")
			}

			err = repo.SwitchBranch(prBranch)
			if err != nil {
				log.Fatal().Err(err).Msg("creating and switching to new branch for pr")
			}
			log.Info().Str("prbranch", prBranch).Msg("Switched to branch")
			err = policy.RenderBundle("releng", policy.BundleVars(&repo))
			if err != nil {
				log.Fatal().Err(err).Msg("template generation")
			}
			// Commit after we generate the files from templates.
			confirmBeforeCommit := false
			hash, err := repo.Commit(commitMsg, confirmBeforeCommit)
			if err != nil {
				log.Fatal().Err(err).Msg("Unable to commit the changes.")
			}
			log.Info().Str("hash", hash.String()).Msg("Commited the changes")

			prURL, err := repo.CreatePR("releng", commitMsg, branch, dryRun, autoMerge)
			if err != nil {
				log.Fatal().Err(err).Msg("unable to create PR")
			}
			log.Info().Str("prurl", prURL).Msg("created PR")
		}
	},
}

// docSubCmd represents the doctor subcommand
var docSubCmd = &cobra.Command{
	Use:     "doctor",
	Aliases: []string{"doc", "fix"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Diagnose problems with the release engineering code",
	Long: `For the supplied repo, release engineering branches release-* and master. For each of these branches, the checks are:
- sync-automation.yml only exists on the branches defined as a source for the backport branches in the config
  + if found on an inactive branch, it is removed
  + if the branch it is found on is protected, a draft PR is created to remove it
- if deprecated code is found, it is removed (not implemented)
  + if on a protected branch a draft PR is created`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("not implemented")
	},
}

func init() {

	relengSubCmd.Flags().StringVar(&prBranch, "prbranch", "", "The branch that will be used for creating the PR - this is the branch that gets pushed to remote")
	relengSubCmd.MarkFlagRequired("prbranch")
	policyCmd.AddCommand(relengSubCmd)
	policyCmd.AddCommand(gpacSubCmd)

	docSubCmd.Flags().String("pattern", "^(release-[[:digit:].]+|master)", "Regexp to match release engineering branches")
	policyCmd.AddCommand(docSubCmd)

	policyCmd.Flags().BoolVar(&bundle, "bundle", false, "List the embedded bundles")
	policyCmd.PersistentFlags().StringSliceVar(&repos, "repos", []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink", "tyk-identity-broker", "portal", "tyk-analytics-ui"}, "Repos to operate upon, comma separated values accepted.")
	policyCmd.PersistentFlags().StringVar(&branch, "branch", "master", "Restrict operations to this branch, all PRs generated will be using this as the base branch")
	policyCmd.PersistentFlags().Bool("sign", false, "Sign commits, requires -k/--key. gpgconf and an active gpg-agent are required if the key is protected by a passphrase.")
	policyCmd.PersistentFlags().StringVarP(&config.RepoURLPrefix, "prefix", "u", "https://github.com/TykTechnologies", "Prefix to derive the fqdn repo")
	policyCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON")
	policyCmd.PersistentFlags().BoolVarP(&dryRun, "dry", "d", false, "Will not make any changes")
	policyCmd.PersistentFlags().BoolVarP(&autoMerge, "auto", "a", true, "Will automerge if all requirements are meet")
	policyCmd.PersistentFlags().StringVar(&ghToken, "token", os.Getenv("GITHUB_TOKEN"), "Github token for private repositories")
	policyCmd.PersistentFlags().StringVar(&dir, "dir", "", "Use dir for git operations, instead of an in-memory fs, if more than one repos are speified in repos, this will be used as a prefix for dirs.")
	rootCmd.AddCommand(policyCmd)
}
