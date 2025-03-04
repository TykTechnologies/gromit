/*
Copyright © 2025 Tyk Technologies

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

	"github.com/TykTechnologies/gromit/pkgs"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var pkgClient *pkgs.Client
var repos *pkgs.Repos

// pkgsCmd represents the pkgs command
var pkgsCmd = &cobra.Command{
	Use:   "pkgs <subcmd>",
	Short: "Interact with package repositories",
	Long: `Binary packages are stored in packcloud.io.

You can perform maintenance using this command tree.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		pcToken := os.Getenv("PACKAGECLOUD_TOKEN")
		if pcToken == "" {
			log.Fatal().Msg("Working with packagecloud.io requires PACKAGECLOUD_TOKEN")
		}
		owner, _ := cmd.Flags().GetString("owner")
		rps, _ := cmd.Flags().GetFloat64("rps")
		burst, _ := cmd.Flags().GetInt("burst")
		pkgClient = pkgs.NewClient(pcToken, owner, rps, burst)
		var err error
		repos, err = pkgs.LoadConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Could not load repo config")
		}
	},
}

// pkgsCmd represents the pkgs command
var cleanSubCmd = &cobra.Command{
	Use:   "clean <repo>",
	Args:  cobra.MinimumNArgs(1),
	Short: "Cleanup packages from the repository",
	Long: `The packages are removed from the repository. The removed pacakges are downloaded before being removed.
Each repo is processed sequentially, Deletions within a repo are processed concurrently, limited by the rps and burst parameters. The concurrency level affects the run time by controlling the number of concurrent downloads. 4 downloads `,
	Run: func(cmd *cobra.Command, args []string) {
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		savedir, _ := cmd.Flags().GetString("savedir")
		delete, err := cmd.Flags().GetBool("delete")
		if err != nil {
			log.Fatal().Err(err).Msg("parsing -delete flag")
		}
		cc := pkgs.CleanConfig{
			Concurrency: concurrency,
			Savedir:     savedir,
			Delete:      delete,
		}
		for _, repoName := range args {
			log.Logger = log.With().Str("repo", repoName).Logger()
			cc.RepoName = repoName
			cc.Backup = repos.ShouldBackup(repoName)
			filter, err := repos.MakeFilter(repoName)
			if err != nil {
				log.Warn().Err(err).Msg("making filter")
				break
			}
			pkgChan, pkgs := pkgClient.AllPackages(repoName, filter)
			cleanErr := pkgClient.Clean(pkgChan, cc)
			if err := pkgs.Wait(); err != nil {
				log.Warn().Err(err).Msg("fetching all packages")
				break
			}
			if cleanErr != nil {
				log.Warn().Err(cleanErr).Msg("cleaning up packages")
			}
			fmt.Println(repoName, filter)
		}
	},
}

func init() {
	pkgsCmd.AddCommand(cleanSubCmd)
	rootCmd.AddCommand(pkgsCmd)

	pkgsCmd.PersistentFlags().String("owner", "tyk", "PackageCloud repo owner")
	pkgsCmd.PersistentFlags().Float64("rps", 10.0, "Requests per second (see burst also)")
	pkgsCmd.PersistentFlags().Int("burst", 20, "rps burst rate (see rps also)")

	cleanSubCmd.Flags().Int("concurrency", 3, "Cleanup concurrency level")
	cleanSubCmd.Flags().String("savedir", "./backup", "Local directory root to save packages before deleting")
	cleanSubCmd.Flags().Bool("delete", false, "Actually delete the package from the repo")
}
