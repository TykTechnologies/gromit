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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/TykTechnologies/gromit/pkgs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
			Progress:    true,
		}
		ll := zerolog.GlobalLevel()
		if ll < zerolog.InfoLevel {
			cc.Progress = false
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true, PartsExclude: []string{zerolog.TimestampFieldName}})
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

var planSubCmd = &cobra.Command{
	Use:   "plan <repo>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Dry-run report of what the retention policy would prune",
	Long: `Computes the retention cutoff for each repo and classifies every package
against it, without deleting or downloading anything.

Repos with a track in the config derive their cutoff from the tracks
section; other repos use their static versioncutoff/agecutoff, making
the plan a preview of what 'pkgs clean' would do.

The JSON output carries the full prune-eligible package list with
checksums, which later execution stages verify before deleting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		graceDays, _ := cmd.Flags().GetInt("grace-days")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		grace := time.Duration(graceDays) * 24 * time.Hour
		tracks, err := pkgs.LoadTracks()
		if err != nil {
			return fmt.Errorf("loading tracks config: %w", err)
		}
		var plans []pkgs.Plan
		for _, repoName := range args {
			cfg, found := (*repos)[repoName]
			if !found {
				return fmt.Errorf("%s not present in pkgs config", repoName)
			}
			items, err := pkgClient.ListPackages(repoName)
			if err != nil {
				return fmt.Errorf("listing %s: %w", repoName, err)
			}
			plan, err := pkgs.BuildPlan(repoName, cfg, tracks, items, time.Now(), grace)
			if err != nil {
				return fmt.Errorf("planning %s: %w", repoName, err)
			}
			pkgClient.FillPrunedBytes(&plan, items, concurrency)
			plans = append(plans, plan)
			if !asJSON {
				fmt.Fprint(cmd.OutOrStdout(), plan.Render())
			}
		}
		if asJSON {
			out, err := json.MarshalIndent(plans, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		}
		return nil
	},
}

var mirrorSubCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Copy the prune-eligible packages from a plan to the S3 archive",
	Long: `Reads a plan (the JSON from 'pkgs plan --json'), matches each
prune-eligible package by sha256 against the live repo, and uploads it
to the archive bucket. Packages already archived are skipped, so
reruns are idempotent. Nothing is ever deleted.

With --verify, every archived object is read back and its hash checked
against the plan, proving the copy is restorable.

Exits non-zero if any package could not be confirmed archived; such a
plan must not proceed to deletion.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile, _ := cmd.Flags().GetString("plan")
		bucket, _ := cmd.Flags().GetString("bucket")
		verify, _ := cmd.Flags().GetBool("verify")

		data, err := os.ReadFile(planFile)
		if err != nil {
			return err
		}
		var plans []pkgs.Plan
		if err := json.Unmarshal(data, &plans); err != nil {
			return fmt.Errorf("parsing %s: %w", planFile, err)
		}
		store, err := pkgs.NewS3Store(cmd.Context(), bucket)
		if err != nil {
			return err
		}
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		results := make([]pkgs.MirrorResult, len(plans))
		g := new(errgroup.Group)
		g.SetLimit(concurrency)
		for i, plan := range plans {
			g.Go(func() error {
				items, err := pkgClient.ListPackages(plan.Repo)
				if err != nil {
					return fmt.Errorf("listing %s: %w", plan.Repo, err)
				}
				results[i] = pkgs.MirrorPlan(cmd.Context(), plan, items, store, verify)
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		clean := true
		for _, res := range results {
			fmt.Fprint(cmd.OutOrStdout(), res.Render())
			clean = clean && res.Clean()
		}
		if !clean {
			return fmt.Errorf("some packages are not confirmed archived")
		}
		return nil
	},
}

func init() {
	pkgsCmd.AddCommand(cleanSubCmd)
	pkgsCmd.AddCommand(planSubCmd)
	pkgsCmd.AddCommand(mirrorSubCmd)
	rootCmd.AddCommand(pkgsCmd)

	pkgsCmd.PersistentFlags().String("owner", "tyk", "PackageCloud repo owner")
	pkgsCmd.PersistentFlags().Float64("rps", 10.0, "Requests per second (see burst also)")
	pkgsCmd.PersistentFlags().Int("burst", 20, "rps burst rate (see rps also)")

	cleanSubCmd.Flags().Int("concurrency", 3, "Cleanup concurrency level")
	cleanSubCmd.Flags().String("savedir", "./backup", "Local directory root to save packages before deleting")
	cleanSubCmd.Flags().Bool("delete", false, "Actually delete the package from the repo")

	planSubCmd.Flags().Bool("json", false, "Emit the plan as JSON, including the prune-eligible package list")
	planSubCmd.Flags().Int("grace-days", 30, "Days until the plan's not_before deadline; override for the 90-day launch notice")
	planSubCmd.Flags().Int("concurrency", 8, "Concurrent size lookups, bounded overall by rps/burst")

	mirrorSubCmd.Flags().String("plan", "", "Plan file from 'pkgs plan --json'")
	mirrorSubCmd.MarkFlagRequired("plan")
	mirrorSubCmd.Flags().String("bucket", "tyk-artifact-archive", "S3 bucket to archive to")
	mirrorSubCmd.Flags().Bool("verify", false, "Read every archived object back and check its hash")
	mirrorSubCmd.Flags().Int("concurrency", 3, "Repos to mirror in parallel")
}
