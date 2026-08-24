package pkgs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	pc "github.com/tyklabs/packagecloud/api/v1"
	"golang.org/x/mod/semver"
)

// Plan is a dry-run report of what the retention policy would prune
// from a repo, without deleting anything
type Plan struct {
	Repo        string    `json:"repo"`
	GeneratedAt time.Time `json:"generated_at"`
	// NotBefore is the earliest time a deletion step may execute
	// this plan; the grace period is carried by the plan itself
	NotBefore time.Time `json:"not_before"`
	Track     string    `json:"track,omitempty"`
	Editions  []string  `json:"editions,omitempty"`
	// Anchor is the series the retention window counts down from,
	// Cutoff the oldest retained series, Series every minor series
	// with released packages
	Anchor string   `json:"anchor,omitempty"`
	Cutoff string   `json:"cutoff,omitempty"`
	Series []string `json:"series"`

	Retained    int   `json:"retained"`
	Pruned      int   `json:"pruned"`
	PrunedBytes int64 `json:"pruned_bytes"`

	PrunedSeries map[string]int `json:"pruned_series,omitempty"`
	Protected    map[string]int `json:"protected,omitempty"`
	NonSemver    int            `json:"non_semver"`

	Packages []PlanPackage `json:"packages,omitempty"`
}

// PlanPackage identifies one prune-eligible package; the checksum is
// its tamper-evident identity
type PlanPackage struct {
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Arch          string    `json:"arch"`
	DistroVersion string    `json:"distro_version"`
	Filename      string    `json:"filename"`
	Sha256Sum     string    `json:"sha256sum"`
	CreateTime    time.Time `json:"created_at"`
}

// ListPackages fetches every package in a repo, unfiltered. Read-only.
func (c *Client) ListPackages(repo string) ([]pc.PackageDetail, error) {
	var all []pc.PackageDetail
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/packages.json", pcPrefix, c.owner, repo)
	for {
		resp, err, next := c.get(url)
		if err != nil {
			return nil, fmt.Errorf("http get err: %v", err)
		}
		var items []pc.PackageDetail
		err = json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("json parse err: %v", err)
		}
		all = append(all, items...)
		if next == "" {
			break
		}
		url = next
	}
	return all, nil
}

// BuildPlan classifies every package in a repo against the retention
// policy. Repos without a track use their static cutoffs, making the
// plan a preview of what `pkgs clean` would do today.
func BuildPlan(repoName string, cfg pkgConfig, tracks Tracks, items []pc.PackageDetail, now time.Time, grace time.Duration) (Plan, error) {
	p := Plan{
		Repo:         repoName,
		GeneratedAt:  now,
		NotBefore:    now.Add(grace),
		Track:        cfg.Track,
		Editions:     cfg.Editions,
		PrunedSeries: make(map[string]int),
		Protected:    make(map[string]int),
	}
	vtrans := strings.NewReplacer("~", "-")

	versions := make([]string, 0, len(items))
	for _, item := range items {
		versions = append(versions, "v"+vtrans.Replace(item.Version))
	}
	p.Series = MinorSeries(versions)

	// A track cutoff is compared by minor series; a static cutoff
	// keeps Filter.Satisfies' full-semver comparison.
	cutoff := semver.Canonical(cfg.VersionCutoff)
	bySeries := false
	if cfg.Track != "" {
		track, found := tracks[cfg.Track]
		if !found {
			return p, fmt.Errorf("track %q is not in the tracks config", cfg.Track)
		}
		anchor, depth, err := track.Anchor(cfg.Editions)
		if err != nil {
			return p, fmt.Errorf("track %q: %w", cfg.Track, err)
		}
		p.Anchor = anchor
		cutoff, err = DeriveCutoff(p.Series, anchor, depth)
		if err != nil {
			return p, err
		}
		bySeries = true
	}
	p.Cutoff = cutoff

	exceptions := make(map[string]bool)
	for _, e := range cfg.Exceptions {
		exceptions[e] = true
	}

	for _, item := range items {
		v := "v" + vtrans.Replace(item.Version)
		if exceptions[v] {
			p.Protected[v]++
			p.Retained++
			continue
		}
		if !semver.IsValid(v) {
			p.NonSemver++
			p.Retained++
			continue
		}
		prune := false
		if cutoff != "" {
			if bySeries {
				prune = semver.Compare(semver.MajorMinor(v), cutoff) < 0
			} else {
				prune = semver.Compare(v, cutoff) < 0
			}
		}
		if !prune && cfg.AgeCutoff != 0 && now.Sub(item.CreateTime) > cfg.AgeCutoff {
			prune = true
		}
		if prune {
			p.Pruned++
			p.PrunedSeries[semver.MajorMinor(v)]++
			if sz, err := strconv.ParseInt(item.Size, 10, 64); err == nil {
				p.PrunedBytes += sz
			}
			p.Packages = append(p.Packages, PlanPackage{
				Name:          item.Name,
				Version:       item.Version,
				Arch:          item.Arch,
				DistroVersion: item.DistroVersion,
				Filename:      item.Filename,
				Sha256Sum:     item.Sha256Sum,
				CreateTime:    item.CreateTime,
			})
		} else {
			p.Retained++
		}
	}
	return p, nil
}

// Render returns a human-readable summary of the plan
func (p Plan) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d packages, %d retained, %d pruned (%.1f GiB)\n",
		p.Repo, p.Retained+p.Pruned, p.Retained, p.Pruned, float64(p.PrunedBytes)/(1<<30))
	fmt.Fprintf(&b, "  no deletion before %s\n", p.NotBefore.Format("2006-01-02"))
	if p.Track != "" {
		fmt.Fprintf(&b, "  track %s editions %v, anchor %s -> cutoff %s (oldest retained series)\n",
			p.Track, p.Editions, p.Anchor, p.Cutoff)
	} else if p.Cutoff != "" {
		fmt.Fprintf(&b, "  static cutoff %s\n", p.Cutoff)
	}
	if len(p.PrunedSeries) > 0 {
		series := make([]string, 0, len(p.PrunedSeries))
		for s := range p.PrunedSeries {
			series = append(series, s)
		}
		semver.Sort(series)
		fmt.Fprintf(&b, "  pruned series:")
		for _, s := range series {
			fmt.Fprintf(&b, " %s(%d)", s, p.PrunedSeries[s])
		}
		fmt.Fprintln(&b)
	}
	if len(p.Protected) > 0 {
		keys := make([]string, 0, len(p.Protected))
		for k := range p.Protected {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "  exceptions held:")
		for _, k := range keys {
			fmt.Fprintf(&b, " %s(%d)", k, p.Protected[k])
		}
		fmt.Fprintln(&b)
	}
	if p.NonSemver > 0 {
		fmt.Fprintf(&b, "  %d non-semver packages always retained\n", p.NonSemver)
	}
	return b.String()
}
