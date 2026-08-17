package pkgs

import (
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadTracksFromConfig proves the shipped config parses,
// validates and has no dangling track references
func TestLoadTracksFromConfig(t *testing.T) {
	config.LoadConfig("")
	tracks, err := LoadTracks()
	require.NoError(t, err)
	require.Contains(t, tracks, "gateway")

	repos, err := LoadConfig()
	require.NoError(t, err)
	for name, cfg := range *repos {
		if cfg.Track == "" {
			continue
		}
		assert.Contains(t, tracks, cfg.Track, "%s references an unknown track", name)
		assert.NotEmpty(t, cfg.Editions, "%s has a track but no editions", name)
	}
}

func TestTrackValidate(t *testing.T) {
	valid := Track{CurrentFeature: "5.14", CurrentLTS: "5.13", LTSMinus1: "5.8"}
	require.NoError(t, valid.Validate())

	cases := []struct {
		name  string
		track Track
		want  string
	}{
		{"missing field", Track{CurrentFeature: "5.14", CurrentLTS: "5.13"}, "lts_minus_1 is not set"},
		{"not semver", Track{CurrentFeature: "5.14", CurrentLTS: "5.13", LTSMinus1: "banana"}, "not a valid version"},
		{"lts newer than feature", Track{CurrentFeature: "5.14", CurrentLTS: "5.15", LTSMinus1: "5.8"}, "must be older than current_feature"},
		{"lts-1 newer than lts", Track{CurrentFeature: "5.14", CurrentLTS: "5.13", LTSMinus1: "5.13"}, "must be older than current_lts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.track.Validate()
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), tc.want), "got %q, want substring %q", err, tc.want)
		})
	}
}

func TestTrackAnchor(t *testing.T) {
	track := Track{CurrentFeature: "5.14", CurrentLTS: "5.13", LTSMinus1: "5.8"}

	anchor, depth, err := track.Anchor([]string{"ce"})
	require.NoError(t, err)
	assert.Equal(t, "5.14", anchor)
	assert.Equal(t, ceAnchorDepth, depth)

	anchor, depth, err = track.Anchor([]string{"ee"})
	require.NoError(t, err)
	assert.Equal(t, "5.8", anchor)
	assert.Equal(t, eeAnchorDepth, depth)

	// shared repos get the longest window
	anchor, depth, err = track.Anchor([]string{"ce", "ee"})
	require.NoError(t, err)
	assert.Equal(t, "5.8", anchor)
	assert.Equal(t, eeAnchorDepth, depth)

	_, _, err = track.Anchor(nil)
	assert.Error(t, err)
	_, _, err = track.Anchor([]string{"enterprise"})
	assert.Error(t, err)
}

func TestMinorSeries(t *testing.T) {
	series := MinorSeries([]string{
		"v5.8.1", "v5.8.2", "v4.0.0", "v5.14.0-rc1", "vnot-a-version", "v3.0.9",
	})
	assert.Equal(t, []string{"v3.0", "v4.0", "v5.8", "v5.14"}, series)
}

func TestDeriveCutoff(t *testing.T) {
	// A realistic shipped-series history: gaps (no 5.9..5.11 shipped
	// packages) and a major boundary below 5.0.
	series := []string{"v4.2", "v4.3", "v5.0", "v5.2", "v5.3", "v5.8", "v5.12", "v5.13", "v5.14"}

	// CE: feature anchor, two series retained below it
	cutoff, err := DeriveCutoff(series, "5.14", ceAnchorDepth)
	require.NoError(t, err)
	assert.Equal(t, "v5.12", cutoff, "skipped series must not widen the window")

	// EE: LTS-1 anchor, three series below (5.0, 4.3, 4.2) crosses
	// the major boundary
	cutoff, err = DeriveCutoff(series, "5.2", eeAnchorDepth)
	require.NoError(t, err)
	assert.Equal(t, "v4.2", cutoff, "window must cross major version boundaries by shipped series")

	// thin history: fewer released series than the depth retains everything
	cutoff, err = DeriveCutoff([]string{"v5.13", "v5.14"}, "5.14", eeAnchorDepth)
	require.NoError(t, err)
	assert.Equal(t, "v5.13", cutoff)

	// anchor with no released packages is a loud failure, not a guess
	_, err = DeriveCutoff(series, "5.15", ceAnchorDepth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no released packages")

	// garbage anchor
	_, err = DeriveCutoff(series, "not-a-version", ceAnchorDepth)
	require.Error(t, err)
}
