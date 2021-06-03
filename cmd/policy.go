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
	"fmt"
	"os"
	"strings"

	"github.com/TykTechnologies/gromit/config"
	"github.com/TykTechnologies/gromit/policy"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repoPolicies policy.RepoPolicies
var signingKeyid uint64
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
	Use:     "generate <repo> <title> [commit_msg...]",
	Aliases: []string{"gen", "add", "new"},
	Args:    cobra.MinimumNArgs(2),
	Short:   "overwrite the meta-automation for a given repo and branch",
	Long: `Will overwite the existing meta-automation, and supports signing of the commits. This is not really meant for use in automation. Use this for manual fixes when you know better than the doctor.
If the branch requires reviews before merging, a draft PR will be created with the changes and @devops will be asked for a review.
`,
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		prTitle := args[1]
		commitMsg := strings.Join(args[2:], "\n")
		fqrn := fmt.Sprintf("%s/%s", config.RepoURLPrefix, repoName)

		log.Logger = log.With().Str("repo", repoName).Str("branch", config.Branch).Logger()

		r, err := policy.FetchRepo(repoName, fqrn, dir, ghToken, 1)
		if err != nil {
			log.Fatal().Err(err).Str("fqrn", fqrn).Msg("could not fetch")
		}
		r.SetDryRun(dryRun)
		signKeyid := viper.GetUint64("signingkey")
		if signKeyid != 0 {
			signer, err := util.GetSigningEntity(signingKeyid)
			if err != nil {
				log.Fatal().Err(err).Uint64("keyid", signingKeyid).Msg("could not obtain signing key")
			}
			err = r.EnableSigning(signer)
			if err != nil {
				log.Warn().Err(err).Msg("commits will not be signed")
			}
		}
		branches, err := repoPolicies.SrcBranches(repoName)
		if err != nil {
			log.Fatal().Err(err).Msg("src branches")
		}
		if config.Branch != "" {
			branches = []string{config.Branch}
		}
		log.Trace().Strs("branches", branches).Msg("to generate")
		for _, b := range branches {
			r.Checkout(b)
			log.Logger = log.With().Str("repo", r.Name).Str("branch", b).Logger()
			err = r.AddMetaAutomation(commitMsg, repoPolicies)
			if err != nil {
				log.Fatal().Err(err).Msg("could not generate meta-automation")
			}
			var remoteBranch string
			isProtected, err := r.IsProtected(b)
			if err != nil {
				log.Fatal().Err(err).Msg("getting protected status")
			}
			if isProtected {
				remoteBranch = fmt.Sprintf("releng/%s", b)
			} else {
				remoteBranch = b
			}
			err = r.Push(b, remoteBranch)
			if err != nil {
				log.Fatal().Err(err).Msg("could not push")
			}
			log.Info().Str("origin", remoteBranch).Msg("pushed to origin")
			if isProtected {
				err = r.CreatePR(prTitle, "sync-automation.tmpl", repoPolicies, false)
				if err != nil {
					log.Fatal().Err(err).Msg("could not create PR")
				}
			}
		}
		log.Info().Strs("prs", r.PRs()).Msg("created")
	},
}

// docSubCmd represents the doctor subcommand
var docSubCmd = &cobra.Command{
	Use:     "doctor <repo>",
	Aliases: []string{"doc", "fix"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "Diagnose and fix problems with the release engineering code",
	Long: `For the supplied repo, release engineering branches release-* and master. For each of these branches, the checks are:
- sync-automation.yml only exists on the branches defined as a source for the backport branches in the config
  + if found on an inactive branch, it is removed
  + if the branch it is found on is protected, a draft PR is created to remove it
- if deprecated code is found, it is removed (not implemented)
  + if on a protected branch a draft PR is created
This can be used in automation.`,
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		log.Logger = log.With().Str("repo", repoName).Logger()

		fqrn := fmt.Sprintf("%s/%s", config.RepoURLPrefix, repoName)
		// Shallow clones cause all sorts of transient faults
		// But with retries, the savings in bandwidth are considerable
		r, err := policy.FetchRepo(repoName, fqrn, dir, ghToken, 1)
		if err != nil {
			log.Fatal().Err(err).Str("fqrn", fqrn).Msg("could not fetch")
		}
		r.SetDryRun(dryRun)
		pattern, _ := cmd.Flags().GetString("pattern")
		relBranches, err := r.Branches(pattern)
		if err != nil {
			log.Fatal().Err(err).Msg("could not fetch branches")
		}
		if config.Branch != "" {
			relBranches = []string{config.Branch}
		}
		log.Trace().Strs("branches", relBranches).Msg("to minister")

		srcBranches, err := repoPolicies.SrcBranches(repoName)
		if err != nil {
			log.Fatal().Err(err).Msg("fetching src branches")
		}
		log.Trace().Strs("relBranches", relBranches).Strs("srcBranches", srcBranches).Msg("starting examination")
		for _, b := range relBranches {
			log.Logger = log.With().Str("branch", b).Logger()
			err = r.Checkout(b)
			if err != nil {
				log.Fatal().Err(err).Msg("checkout")
			}
			err = r.CheckMetaAutomation(repoPolicies)
			if err != nil {
				log.Fatal().Err(err).Msg("checking meta-automation")
			}
		}
		log.Info().Strs("prs", r.PRs()).Msg("created")
	},
}

func init() {
	policyCmd.AddCommand(genSubCmd)

	docSubCmd.Flags().String("pattern", "^(release-[[:digit:].]+|master)", "Regexp to match release engineering branches")
	policyCmd.AddCommand(docSubCmd)

	policyCmd.PersistentFlags().StringVarP(&config.Branch, "branch", "b", "", "Restrict operations to this branch")
	policyCmd.PersistentFlags().Bool("sign", true, "Sign commits, requires -k/--key. gpgconf and an active gpg-agent are required if the key is protected by a passphrase.")
	policyCmd.PersistentFlags().StringVarP(&config.RepoURLPrefix, "prefix", "u", "https://github.com/TykTechnologies", "Prefix to derive the fqdn repo")
	policyCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON")
	policyCmd.PersistentFlags().BoolVarP(&dryRun, "dry", "d", false, "Do not actually push or create PRs")
	policyCmd.PersistentFlags().StringVar(&ghToken, "token", os.Getenv("GITHUB_TOKEN"), "Github token for private repositories")
	policyCmd.PersistentFlags().StringVar(&dir, "dir", "", "Use dir for git operations, instead of an in-memory fs")
	rootCmd.AddCommand(policyCmd)
}
