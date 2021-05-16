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

	"github.com/TykTechnologies/gromit/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var branch string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Mess with the release engineering policy",
	Long: `Controls the automation that is active in each repo for each branch.
Operates directly on github and creates PRs for protected branches. Requires an OAuth2 token and a section in the config file describing the policy. `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := config.LoadRepoPolicies()
		if err != nil {
			log.Fatal().Err(err).Msg("could not parse repo policies")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			json, err := json.Marshal(config.RepoPolicies)
			if err != nil {
				log.Fatal().Interface("policies", config.RepoPolicies).Msg("could not marshal policies into JSON")
			}
			cmd.Println(string(json))
		} else {
			cmd.Println(config.RepoPolicies)
		}
	},
}

// genSubCmd generates a set of files ready to be checked in
var genSubCmd = &cobra.Command{
	Use:     "generate <repo>",
	Aliases: []string{"gen"},
	Args:    cobra.MinimumNArgs(1),
	Short:   "generate a set of file for a given repo and branch",
	Long:    `Will create a tree rooted at <repo> with the files release engineering depends on, creating directories as needed, this can be overlaid on a repo and merged into it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templateDir, _ := cmd.Flags().GetString("templates")
		return config.RepoPolicies.Gen(args[0], branch, templateDir)

	},
}

func init() {
	genSubCmd.Flags().StringVarP(&branch, "branch", "b", "master", "Branch for meta-automation")
	genSubCmd.Flags().StringP("meta", "m", "templates", "Directory with meta-automation templates")

	policyCmd.AddCommand(genSubCmd)
	policyCmd.Flags().BoolP("json", "j", false, "Output in JSON")
	rootCmd.AddCommand(policyCmd)
}
