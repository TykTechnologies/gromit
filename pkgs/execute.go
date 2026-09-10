package pkgs

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	pc "github.com/tyklabs/packagecloud/api/v1"
)

// DeleteFunc removes one package listing from the public repo. It is
// a function so tests (and dry runs) never need a live client.
type DeleteFunc func(pc.PackageDetail) error

// ExecuteConfig carries the per-run switches for ExecutePlan. Delete
// defaults to false: a run is a dry run unless explicitly armed.
type ExecuteConfig struct {
	Delete bool
	Now    time.Time
}

// Executed package actions. Skips are never an error: a skipped
// package is simply not deleted, and the reason is recorded.
const (
	ActionDeleted     = "deleted"
	ActionWouldDelete = "would-delete"
	ActionSkipped     = "skipped"
	ActionFailed      = "failed"
)

// ExecutedPackage records the outcome for one announced package
// listing. Fips marks packages that are deleted without an archive
// copy, per the compliance position.
type ExecutedPackage struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	DistroVersion string `json:"distro_version"`
	Filename      string `json:"filename"`
	Sha256Sum     string `json:"sha256sum"`
	Fips          bool   `json:"fips,omitempty"`
	Action        string `json:"action"`
	Reason        string `json:"reason,omitempty"`
}

// ExecuteResult is the executed.json committed next to the plan: the
// announced-versus-executed audit trail for one repo.
type ExecuteResult struct {
	Repo            string    `json:"repo"`
	PlanGeneratedAt time.Time `json:"plan_generated_at"`
	NotBefore       time.Time `json:"not_before"`
	ExecutedAt      time.Time `json:"executed_at"`
	DryRun          bool      `json:"dry_run"`

	Deleted     int `json:"deleted"`
	WouldDelete int `json:"would_delete"`
	Skipped     int `json:"skipped"`
	Failed      int `json:"failed"`
	// Fips counts packages deleted (or would be) without an archive
	Fips int `json:"fips"`

	Packages []ExecutedPackage `json:"packages,omitempty"`
}

// Clean returns true if no deletion attempt failed. Skips do not
// count: a skipped package is left in place, which is always safe.
func (r ExecuteResult) Clean() bool {
	return r.Failed == 0
}

func (r ExecuteResult) Render() string {
	var b strings.Builder
	mode := "DELETED"
	n := r.Deleted
	if r.DryRun {
		mode = "would delete"
		n = r.WouldDelete
	}
	fmt.Fprintf(&b, "%s: %d %s, %d skipped, %d failed (%d FIPS, no archive)\n",
		r.Repo, n, mode, r.Skipped, r.Failed, r.Fips)
	reasons := make(map[string]int)
	for _, p := range r.Packages {
		if p.Action == ActionSkipped || p.Action == ActionFailed {
			reasons[p.Reason]++
		}
	}
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  %d: %s\n", reasons[k], k)
	}
	return b.String()
}

// isFIPSPackage follows the *-fips naming convention verified by the
// TT-17822 audit; FIPS packages are deleted but never archived.
func isFIPSPackage(name string) bool {
	return strings.HasSuffix(name, "-fips")
}

// listingKey identifies one package listing. The same file (one
// sha256) is listed once per distro version, and each listing is
// deleted separately, so identity is sha+distro+filename.
func listingKey(sha, distro, filename string) string {
	return sha + "|" + distro + "|" + filename
}

// ExecutePlan is the only routine in the retention pipeline that
// destroys anything, so every failsafe lives here. It deletes a
// package listing only when all of these hold:
//
//  1. the announced plan's grace period has elapsed (not_before);
//  2. a freshly derived plan (current config, current registry) still
//     prunes the same listing: exceptions granted during the grace
//     window and tampered plan files both fall out here;
//  3. the listing still exists on packagecloud with the announced
//     sha256;
//  4. the archive holds a copy with the announced sha256 (FIPS
//     packages are exempt: they are deleted, never archived, and the
//     report flags them).
//
// Anything that fails a gate is skipped and reported, never deleted.
// With cfg.Delete false nothing is destroyed and the result records
// what a real run would have done.
func ExecutePlan(ctx context.Context, announced, fresh Plan, items []pc.PackageDetail, store MirrorStore, del DeleteFunc, cfg ExecuteConfig) (ExecuteResult, error) {
	res := ExecuteResult{
		Repo:            announced.Repo,
		PlanGeneratedAt: announced.GeneratedAt,
		NotBefore:       announced.NotBefore,
		ExecutedAt:      cfg.Now,
		DryRun:          !cfg.Delete,
	}
	if cfg.Delete && cfg.Now.Before(announced.NotBefore) {
		return res, fmt.Errorf("%s: plan not_before %s has not elapsed (now %s)",
			announced.Repo, announced.NotBefore.Format(time.RFC3339), cfg.Now.Format(time.RFC3339))
	}
	if !cfg.Delete && cfg.Now.Before(announced.NotBefore) {
		log.Warn().Msgf("%s: grace period runs until %s; this dry run previews a deletion that is not yet allowed",
			announced.Repo, announced.NotBefore.Format("2006-01-02"))
	}

	freshEligible := make(map[string]bool, len(fresh.Packages))
	for _, pp := range fresh.Packages {
		freshEligible[listingKey(pp.Sha256Sum, pp.DistroVersion, pp.Filename)] = true
	}
	live := make(map[string]pc.PackageDetail, len(items))
	for _, item := range items {
		live[listingKey(item.Sha256Sum, item.DistroVersion, item.Filename)] = item
	}

	for _, pp := range announced.Packages {
		ep := ExecutedPackage{
			Name:          pp.Name,
			Version:       pp.Version,
			DistroVersion: pp.DistroVersion,
			Filename:      pp.Filename,
			Sha256Sum:     pp.Sha256Sum,
			Fips:          isFIPSPackage(pp.Name),
		}
		key := listingKey(pp.Sha256Sum, pp.DistroVersion, pp.Filename)

		skip := func(reason string) {
			ep.Action = ActionSkipped
			ep.Reason = reason
			res.Skipped++
			res.Packages = append(res.Packages, ep)
		}

		if !freshEligible[key] {
			skip("no longer prune-eligible in the freshly derived plan")
			continue
		}
		item, found := live[key]
		if !found {
			skip("not on packagecloud with the announced sha256")
			continue
		}
		if !ep.Fips {
			archiveKey := fmt.Sprintf("%s/%s/%s", announced.Repo, item.DistroVersion, item.Filename)
			sha, exists, err := store.Head(ctx, archiveKey)
			if err != nil {
				skip(fmt.Sprintf("archive check failed: %v", err))
				continue
			}
			if !exists {
				skip("not confirmed archived")
				continue
			}
			if sha != pp.Sha256Sum {
				skip("archived copy does not match the announced sha256")
				continue
			}
		}

		if ep.Fips {
			res.Fips++
		}
		if !cfg.Delete {
			ep.Action = ActionWouldDelete
			res.WouldDelete++
			res.Packages = append(res.Packages, ep)
			continue
		}
		if err := del(item); err != nil {
			log.Error().Err(err).Msgf("deleting %s %s %s", item.Name, item.Version, item.DistroVersion)
			ep.Action = ActionFailed
			ep.Reason = err.Error()
			res.Failed++
			res.Packages = append(res.Packages, ep)
			continue
		}
		ep.Action = ActionDeleted
		res.Deleted++
		res.Packages = append(res.Packages, ep)
	}
	return res, nil
}
