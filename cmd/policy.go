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

var repoPolicies policy.Policies
var repos []string
var branch string
var jsonOutput, dryRun bool
var ghToken string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Mess with the release engineering policy",
	Long: `Controls the automation that is active in each repo for each branch.
Operates directly on github and creates PRs for protected branches. Requires an OAuth2 token and a section in the config file describing the policy. `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := policy.LoadRepoPolicies(&repoPolicies)
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			json, err := json.Marshal(repoPolicies)
			if err != nil {
				log.Fatal().Interface("policies", repoPolicies).Msg("could not marshal policies into JSON")
			}
			cmd.Println(string(json))
		} else {
			cmd.Println(repoPolicies)
		}
	},
}

// genSubCmd generates a set of files in an in-memory git repo and pushes it to origin.
var genSubCmd = &cobra.Command{
	Use:     "generate <bundle> <commit msg> [commit_msg...]",
	Aliases: []string{"gen", "add", "new"},
	Args:    cobra.MinimumNArgs(2),
	Short:   "(re-)generate the template bundle and update git",
	Long: `Will render templates, overlaid onto an in-memory git repo. 
If the branch is marked protected in the repo policies, a draft PR will be created with the changes and @devops will be asked for a review.
`,
	Run: func(cmd *cobra.Command, args []string) {
		bundle := args[0]
		commitMsg := strings.Join(args[1:], "\n")
		signingKeyid := viper.GetUint64("signingkey")
		for _, repoName := range repos {
			repo, err := repoPolicies.GetRepo(repoName, config.RepoURLPrefix, branch)
			if err != nil {
				log.Fatal().Err(err).Msg("getting repo")
			}
			err = repo.InitGit(1, signingKeyid, dir, ghToken)
			if err != nil {
				log.Fatal().Err(err).Msg("initialising git")
			}
			err = repo.GenTemplate(bundle, commitMsg)
			if err != nil {
				log.Fatal().Err(err).Msg("template generation")
			}
			remoteBranch, err := repo.Push()
			log.Info().Str("origin", remoteBranch).Msg("pushed")
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
	policyCmd.AddCommand(genSubCmd)

	docSubCmd.Flags().String("pattern", "^(release-[[:digit:].]+|master)", "Regexp to match release engineering branches")
	policyCmd.AddCommand(docSubCmd)

	policyCmd.PersistentFlags().StringSliceVar(&repos, "repos", []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink", "tyk-identity-broker", "portal"}, "Repos to operate upon, comma separated values accepted.")
	policyCmd.PersistentFlags().StringVar(&branch, "branch", "", "Restrict operations to this branch")
	policyCmd.PersistentFlags().Bool("sign", false, "Sign commits, requires -k/--key. gpgconf and an active gpg-agent are required if the key is protected by a passphrase.")
	policyCmd.PersistentFlags().StringVarP(&config.RepoURLPrefix, "prefix", "u", "https://github.com/TykTechnologies", "Prefix to derive the fqdn repo")
	policyCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON")
	policyCmd.PersistentFlags().BoolVarP(&dryRun, "dry", "d", false, "Will not make any changes")
	policyCmd.PersistentFlags().StringVar(&ghToken, "token", os.Getenv("GITHUB_TOKEN"), "Github token for private repositories")
	policyCmd.PersistentFlags().StringVar(&dir, "dir", "", "Use dir for git operations, instead of an in-memory fs")
	rootCmd.AddCommand(policyCmd)
}
