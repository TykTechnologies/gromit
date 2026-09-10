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
		if cmd.Name() == "retirement" {
			return
		}
		var err error
		repos, err = pkgs.LoadConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Could not load repo config")
		}
		if cmd.Name() == "images" || (cmd.Parent() != nil && cmd.Parent().Name() == "images") {
			return
		}
		pcToken := os.Getenv("PACKAGECLOUD_TOKEN")
		if pcToken == "" {
			log.Fatal().Msg("Working with packagecloud.io requires PACKAGECLOUD_TOKEN")
		}
		owner, _ := cmd.Flags().GetString("owner")
		rps, _ := cmd.Flags().GetFloat64("rps")
		burst, _ := cmd.Flags().GetInt("burst")
		pkgClient = pkgs.NewClient(pcToken, owner, rps, burst)
	},
}

// pkgsCmd represents the pkgs command
var cleanSubCmd = &cobra.Command{
	Use:   "clean <repo>",
	Args:  cobra.ArbitraryArgs,
	Short: "Cleanup packages from the repository",
	Long: `The packages are removed from the repository. The removed pacakges are downloaded before being removed.
Each repo is processed sequentially, Deletions within a repo are processed concurrently, limited by the rps and burst parameters. The concurrency level affects the run time by controlling the number of concurrent downloads. 4 downloads

With --plan, clean becomes the gated execution step of the retention
pipeline (TT-17825) and ignores the static filter config entirely: see
'pkgs clean --plan --help' notes below. Production deletion runs are
workflow-driven from a plan committed to the plans repo; the deletion
workflow ships deliberately inert until the rollout is signed off.

Gates enforced with --plan:
  - the plan's not_before must have elapsed (when --delete is set)
  - the repo must set allow_delete in the pkgs config (when --delete is set)
  - a freshly derived plan must still prune each listing
  - the live listing must carry the announced sha256
  - the archive must hold a verified copy (FIPS: deleted, never archived)

Anything failing a gate is skipped and reported, never deleted.
--executed writes the announced-versus-executed audit trail; commit it
next to the plan. Positional args filter the plan to those repos.`,
	Run: func(cmd *cobra.Command, args []string) {
		if planFile, _ := cmd.Flags().GetString("plan"); planFile != "" {
			if err := gatedClean(cmd, args, planFile); err != nil {
				log.Fatal().Err(err).Msg("gated clean")
			}
			return
		}
		if len(args) < 1 {
			log.Fatal().Msg("clean without --plan needs at least one repo argument")
		}
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

func gatedClean(cmd *cobra.Command, args []string, planFile string) error {
	doDelete, _ := cmd.Flags().GetBool("delete")
	bucket, _ := cmd.Flags().GetString("bucket")
	executedFile, _ := cmd.Flags().GetString("executed")

	data, err := os.ReadFile(planFile)
	if err != nil {
		return err
	}
	var plans []pkgs.Plan
	if err := json.Unmarshal(data, &plans); err != nil {
		return fmt.Errorf("parsing %s: %w", planFile, err)
	}
	only := make(map[string]bool, len(args))
	for _, a := range args {
		only[a] = true
	}
	tracks, err := pkgs.LoadTracks()
	if err != nil {
		return fmt.Errorf("loading tracks config: %w", err)
	}
	store, err := pkgs.NewS3Store(cmd.Context(), bucket)
	if err != nil {
		return err
	}

	now := time.Now()
	var results []pkgs.ExecuteResult
	for _, plan := range plans {
		if len(only) > 0 && !only[plan.Repo] {
			continue
		}
		cfg, found := (*repos)[plan.Repo]
		if !found {
			return fmt.Errorf("%s is in the plan but not in the pkgs config", plan.Repo)
		}
		if doDelete && !cfg.AllowDelete {
			return fmt.Errorf("%s is not enabled for deletion; the rollout is repo by repo, set pkgs.%s.allow_delete in a reviewed PR first", plan.Repo, plan.Repo)
		}
		items, err := pkgClient.ListPackages(plan.Repo)
		if err != nil {
			return fmt.Errorf("listing %s: %w", plan.Repo, err)
		}
		fresh, err := pkgs.BuildPlan(plan.Repo, cfg, tracks, items, now, 0)
		if err != nil {
			return fmt.Errorf("re-deriving the plan for %s: %w", plan.Repo, err)
		}
		res, err := pkgs.ExecutePlan(cmd.Context(), plan, fresh, items, store, pkgClient.Delete,
			pkgs.ExecuteConfig{Delete: doDelete, Now: now})
		if err != nil {
			return err
		}
		results = append(results, res)
		fmt.Fprint(cmd.OutOrStdout(), res.Render())
	}
	if len(results) == 0 {
		return fmt.Errorf("no plans in %s matched %v", planFile, args)
	}
	if executedFile != "" {
		out, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(executedFile, out, 0o644); err != nil {
			return err
		}
	}
	for _, res := range results {
		if !res.Clean() {
			return fmt.Errorf("some deletions failed; see the execution report")
		}
	}
	return nil
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

var retirementSubCmd = &cobra.Command{
	Use:   "retirement",
	Short: "Render the public retired-versions table from a plan",
	Long: `Reads a plan (the JSON from 'pkgs plan --json') and emits the
retired-versions snippet published on the tyk.io retention policy
page. Only track-driven repos appear in the table.

Rendering from the committed plan rather than recomputing keeps the
published table identical to what the pruning run will enforce.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile, _ := cmd.Flags().GetString("plan")
		data, err := os.ReadFile(planFile)
		if err != nil {
			return err
		}
		var plans []pkgs.Plan
		if err := json.Unmarshal(data, &plans); err != nil {
			return fmt.Errorf("parsing %s: %w", planFile, err)
		}
		fmt.Fprint(cmd.OutOrStdout(), pkgs.RenderRetiredVersions(plans))
		return nil
	},
}

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Docker Hub image retention",
}

var imagesPlanCmd = &cobra.Command{
	Use:   "plan <repo>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Dry-run report of which Hub tags the retention policy would prune",
	Long: `Lists tags on each product's Hub images (CE, EE, FIPS) and classifies
them with the same cutoffs as 'pkgs plan'. Only real release tags
(vMAJOR.MINOR.PATCH) can be pruned; aliases, RCs and moving tags are
retained. Arguments are a pkgs key (tyk-gateway) or a Hub name
(tyk-gateway-ee).

Read-only: nothing is copied or deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		graceDays, _ := cmd.Flags().GetInt("grace-days")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		grace := time.Duration(graceDays) * 24 * time.Hour
		tracks, err := pkgs.LoadTracks()
		if err != nil {
			return fmt.Errorf("loading tracks config: %w", err)
		}
		rps, _ := cmd.Flags().GetFloat64("rps")
		burst, _ := cmd.Flags().GetInt("burst")
		hub := pkgs.NewHubClient("", rps, burst)
		var plans []pkgs.ImagePlan
		for _, arg := range args {
			repoName, cfg, images, err := repos.ResolveImageArg(arg)
			if err != nil {
				return err
			}
			for _, image := range images {
				tags, err := hub.ListTags(image)
				if err != nil {
					return err
				}
				plan, err := pkgs.BuildImagePlan(repoName, image, cfg, tracks, tags, time.Now(), grace)
				if err != nil {
					return fmt.Errorf("planning %s: %w", image, err)
				}
				if err := hub.FillPlanDigests(image, &plan, concurrency); err != nil {
					return err
				}
				plans = append(plans, plan)
				if !asJSON {
					fmt.Fprint(cmd.OutOrStdout(), plan.Render())
				}
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

var imagesMirrorCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Copy the prune-eligible Hub images from a plan to the S3 archive",
	Long: `Reads a plan (the JSON from 'pkgs images plan --json'), matches each
prune-eligible tag by digest against the live registry, and uploads an
OCI-layout tarball to the archive bucket. Images already archived are
skipped, so reruns are idempotent. FIPS images (never_mirror) are not
copied. Nothing is ever deleted.

With --verify, every archived object is read back and checked for the
plan digest, proving the copy is restorable.

Exits non-zero if any image could not be confirmed archived; such a
plan must not proceed to deletion.

Restore: extract the tarball and push the OCI layout, e.g.
  crane push ./layout docker.io/<image>:<tag>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile, _ := cmd.Flags().GetString("plan")
		bucket, _ := cmd.Flags().GetString("bucket")
		verify, _ := cmd.Flags().GetBool("verify")

		data, err := os.ReadFile(planFile)
		if err != nil {
			return err
		}
		var plans []pkgs.ImagePlan
		if err := json.Unmarshal(data, &plans); err != nil {
			return fmt.Errorf("parsing %s: %w", planFile, err)
		}
		store, err := pkgs.NewS3Store(cmd.Context(), bucket)
		if err != nil {
			return err
		}
		rps, _ := cmd.Flags().GetFloat64("rps")
		burst, _ := cmd.Flags().GetInt("burst")
		hub := pkgs.NewHubClient("", rps, burst)
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		results := make([]pkgs.MirrorResult, len(plans))
		g := new(errgroup.Group)
		g.SetLimit(concurrency)
		for i, plan := range plans {
			g.Go(func() error {
				results[i] = pkgs.MirrorImagePlan(cmd.Context(), plan, hub, store, verify)
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
			return fmt.Errorf("some images are not confirmed archived")
		}
		return nil
	},
}

var imagesCheckCmd = &cobra.Command{
	Use:   "check <file>",
	Args:  cobra.ExactArgs(1),
	Short: "Confirm an OCI-layout tarball contains a plan digest",
	Long: `Used by the restore workflow after downloading an archived image.
Exits non-zero if the tarball is not an OCI layout or does not contain
the given digest.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		digest, _ := cmd.Flags().GetString("digest")
		return pkgs.CheckImageArchive(args[0], digest)
	},
}

func init() {
	pkgsCmd.AddCommand(cleanSubCmd)
	pkgsCmd.AddCommand(planSubCmd)
	pkgsCmd.AddCommand(mirrorSubCmd)
	pkgsCmd.AddCommand(retirementSubCmd)
	rootCmd.AddCommand(pkgsCmd)

	pkgsCmd.PersistentFlags().String("owner", "tyk", "PackageCloud repo owner")
	pkgsCmd.PersistentFlags().Float64("rps", 10.0, "Requests per second (see burst also)")
	pkgsCmd.PersistentFlags().Int("burst", 20, "rps burst rate (see rps also)")

	cleanSubCmd.Flags().Int("concurrency", 3, "Cleanup concurrency level")
	cleanSubCmd.Flags().String("savedir", "./backup", "Local directory root to save packages before deleting")
	cleanSubCmd.Flags().Bool("delete", false, "Actually delete the package from the repo")
	cleanSubCmd.Flags().String("plan", "", "Committed plan (JSON from 'pkgs plan --json'): execute it gated instead of the static filters")
	cleanSubCmd.Flags().String("executed", "", "Write the announced-versus-executed report (executed.json) here")
	cleanSubCmd.Flags().String("bucket", "tyk-artifact-archive", "S3 archive bucket that must hold each package before it is deleted")

	planSubCmd.Flags().Bool("json", false, "Emit the plan as JSON, including the prune-eligible package list")
	planSubCmd.Flags().Int("grace-days", 30, "Days until the plan's not_before deadline; override for the 90-day launch notice")
	planSubCmd.Flags().Int("concurrency", 8, "Concurrent size lookups, bounded overall by rps/burst")

	mirrorSubCmd.Flags().String("plan", "", "Plan file from 'pkgs plan --json'")
	mirrorSubCmd.MarkFlagRequired("plan")
	mirrorSubCmd.Flags().String("bucket", "tyk-artifact-archive", "S3 bucket to archive to")
	mirrorSubCmd.Flags().Bool("verify", false, "Read every archived object back and check its hash")
	mirrorSubCmd.Flags().Int("concurrency", 3, "Repos to mirror in parallel")

	retirementSubCmd.Flags().String("plan", "", "Plan file from 'pkgs plan --json'")
	retirementSubCmd.MarkFlagRequired("plan")

	imagesCmd.AddCommand(imagesPlanCmd)
	imagesCmd.AddCommand(imagesMirrorCmd)
	imagesCmd.AddCommand(imagesCheckCmd)
	pkgsCmd.AddCommand(imagesCmd)
	imagesPlanCmd.Flags().Bool("json", false, "Emit the plan as JSON, including the prune-eligible tag list")
	imagesPlanCmd.Flags().Int("grace-days", 30, "Days until the plan's not_before deadline")
	imagesPlanCmd.Flags().Int("concurrency", 8, "Concurrent digest lookups for tags the list endpoint omitted")

	imagesMirrorCmd.Flags().String("plan", "", "Plan file from 'pkgs images plan --json'")
	imagesMirrorCmd.MarkFlagRequired("plan")
	imagesMirrorCmd.Flags().String("bucket", "tyk-artifact-archive", "S3 bucket to archive to")
	imagesMirrorCmd.Flags().Bool("verify", false, "Read every archived object back and check its digest")
	imagesMirrorCmd.Flags().Int("concurrency", 2, "Hub images to mirror in parallel")

	imagesCheckCmd.Flags().String("digest", "", "Image digest the archive must contain")
	imagesCheckCmd.MarkFlagRequired("digest")
}
