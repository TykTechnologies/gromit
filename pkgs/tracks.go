package pkgs

import (
	"fmt"
	"slices"

	"github.com/spf13/viper"
	"golang.org/x/mod/semver"
)

// Track models one release train; retention thresholds are derived
// from these three reference points.
type Track struct {
	CurrentFeature string `mapstructure:"current_feature"`
	CurrentLTS     string `mapstructure:"current_lts"`
	LTSMinus1      string `mapstructure:"lts_minus_1"`
}

// Tracks maps a product (e.g. "gateway") to its release track
type Tracks map[string]Track

// CE retains the feature series and two released series before it,
// EE retains down to three released series below the previous LTS.
const (
	ceAnchorDepth = 2
	eeAnchorDepth = 3
)

// LoadTracks returns the validated tracks section of the config file
func LoadTracks() (Tracks, error) {
	tracks := make(Tracks)
	if err := viper.UnmarshalKey("tracks", &tracks); err != nil {
		return nil, err
	}
	for name, t := range tracks {
		if err := t.Validate(); err != nil {
			return nil, fmt.Errorf("tracks.%s: %w", name, err)
		}
	}
	return tracks, nil
}

// Validate checks that a track is complete, parseable and ordered
func (t Track) Validate() error {
	fields := map[string]string{
		"current_feature": t.CurrentFeature,
		"current_lts":     t.CurrentLTS,
		"lts_minus_1":     t.LTSMinus1,
	}
	for name, val := range fields {
		if val == "" {
			return fmt.Errorf("%s is not set", name)
		}
		if !semver.IsValid("v" + val) {
			return fmt.Errorf("%s: %q is not a valid version", name, val)
		}
	}
	if semver.Compare("v"+t.LTSMinus1, "v"+t.CurrentLTS) >= 0 {
		return fmt.Errorf("lts_minus_1 (%s) must be older than current_lts (%s)", t.LTSMinus1, t.CurrentLTS)
	}
	if semver.Compare("v"+t.CurrentLTS, "v"+t.CurrentFeature) >= 0 {
		return fmt.Errorf("current_lts (%s) must be older than current_feature (%s)", t.CurrentLTS, t.CurrentFeature)
	}
	return nil
}

// Anchor returns the series the retention window counts down from and
// the depth to count. With several editions the longest window wins.
func (t Track) Anchor(editions []string) (string, int, error) {
	if len(editions) == 0 {
		return "", 0, fmt.Errorf("no editions configured")
	}
	longest := ""
	for _, e := range editions {
		switch e {
		case "ce", "ee":
			if longest == "" || e == "ee" {
				longest = e
			}
		default:
			return "", 0, fmt.Errorf("unknown edition %q", e)
		}
	}
	if longest == "ee" {
		return t.LTSMinus1, eeAnchorDepth, nil
	}
	return t.CurrentFeature, ceAnchorDepth, nil
}

// MinorSeries returns the sorted, de-duplicated minor series
// (vMAJOR.MINOR) covered by the supplied versions
func MinorSeries(versions []string) []string {
	set := make(map[string]bool)
	for _, v := range versions {
		if !semver.IsValid(v) {
			continue
		}
		set[semver.MajorMinor(v)] = true
	}
	series := make([]string, 0, len(set))
	for s := range set {
		series = append(series, s)
	}
	semver.Sort(series)
	return series
}

// DeriveCutoff walks the released minor series instead of doing
// version arithmetic: series can be skipped and windows can cross
// major boundaries, so "anchor minus N" only means something against
// the series that actually shipped. Returns the oldest retained
// series.
func DeriveCutoff(series []string, anchor string, depth int) (string, error) {
	a := semver.MajorMinor("v" + anchor)
	if !semver.IsValid(a) {
		return "", fmt.Errorf("track anchor %q is not a valid version", anchor)
	}
	idx := slices.Index(series, a)
	if idx < 0 {
		return "", fmt.Errorf("track anchor %s has no released packages; released series: %v", a, series)
	}
	idx -= depth
	if idx < 0 {
		idx = 0
	}
	return series[idx], nil
}
