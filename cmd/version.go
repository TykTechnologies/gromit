package cmd

/*
Copyright © 2020 Tyk Technologies

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

import (
	"fmt"

	"github.com/TykTechnologies/gromit/util"
	"github.com/spf13/cobra"
)

// orgsCmd represents the top-level orgs command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Long:  `Can also provide build information and commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		full, _ := cmd.Flags().GetBool("full")
		fmt.Println(util.Version())
		if full {
			fmt.Println(util.Commit())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolP("full", "f", false, "Add build information")
}
