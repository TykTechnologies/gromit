package pkgs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pc "github.com/tyklabs/packagecloud/api/v1"
)

var planNow = time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
var planGrace = 30 * 24 * time.Hour

func pkg(version string, age time.Duration) pc.PackageDetail {
	return pc.PackageDetail{
		Name:          "tyk-test",
		Version:       version,
		Arch:          "amd64",
		DistroVersion: "ubuntu/jammy",
		Filename:      "tyk-test_" + version + "_amd64.deb",
		Sha256Sum:     "sha-" + version,
		CreateTime:    planNow.Add(-age),
	}
}

var testTracks = Tracks{
	"gateway": {CurrentFeature: "5.14", CurrentLTS: "5.13", LTSMinus1: "5.8"},
}

func TestBuildPlanTrackDriven(t *testing.T) {
	cfg := pkgConfig{
		Track:      "gateway",
		Editions:   []string{"ce", "ee"},
		Exceptions: []string{"v3.0.9"},
		Name:       "Tyk Test",
	}
	items := []pc.PackageDetail{
		pkg("2.8.3", 8*365*24*time.Hour), // below cutoff: pruned
		pkg("2.9.4", 8*365*24*time.Hour), // below cutoff: pruned
		pkg("3.0.9", 6*365*24*time.Hour), // in cutoff series and protected
		pkg("5.2.0", 3*365*24*time.Hour), // inside the window: retained
		pkg("5.3.0", 3*365*24*time.Hour), // inside the window: retained
		pkg("5.8.1", 365*24*time.Hour),   // anchor series: retained
		pkg("5.14.0", 24*time.Hour),      // current feature: retained
		pkg("1.0nightly", 24*time.Hour),  // non-semver: retained
		pkg("5.14.0~rc1", 24*time.Hour),  // tilde translates, semver: retained
	}
	// EE window: anchor 5.8, retaining three shipped series below it
	// (5.3, 5.2, 3.0), so the cutoff is v3.0
	plan, err := BuildPlan("tyk-test", cfg, testTracks, items, planNow, planGrace)
	require.NoError(t, err)
	assert.Equal(t, "Tyk Test", plan.Product)

	// the plan carries its own grace-period deadline
	assert.Equal(t, planNow.AddDate(0, 0, 30), plan.NotBefore)
	assert.Equal(t, "5.8", plan.Anchor)
	assert.Equal(t, "v3.0", plan.Cutoff)
	assert.Equal(t, 2, plan.Pruned)
	assert.Equal(t, 7, plan.Retained)
	assert.Equal(t, 1, plan.NonSemver)
	assert.Equal(t, map[string]int{"v3.0.9": 1}, plan.Protected)
	assert.Equal(t, map[string]int{"v2.8": 1, "v2.9": 1}, plan.PrunedSeries)

	// the prune list carries the tamper-evident identity
	require.Len(t, plan.Packages, 2)
	for _, pp := range plan.Packages {
		assert.NotEmpty(t, pp.Sha256Sum)
	}
}

// TestBuildPlanStatic proves a repo without a track previews the
// existing clean semantics
func TestBuildPlanStatic(t *testing.T) {
	cfg := pkgConfig{
		VersionCutoff: "v1.7",
		AgeCutoff:     3 * 365 * 24 * time.Hour,
		Exceptions:    []string{"v1.8.2"},
	}
	items := []pc.PackageDetail{
		pkg("1.6.9", 24*time.Hour),       // below cutoff: pruned
		pkg("1.7.0", 24*time.Hour),       // at cutoff: retained
		pkg("1.8.2", 5*365*24*time.Hour), // old but protected
		pkg("1.9.0", 4*365*24*time.Hour), // above cutoff but too old: pruned
		pkg("2.0.0", 24*time.Hour),       // fresh and above cutoff: retained
	}
	plan, err := BuildPlan("tyk-mdcb", cfg, testTracks, items, planNow, planGrace)
	require.NoError(t, err)

	assert.Empty(t, plan.Anchor)
	assert.Equal(t, "v1.7.0", plan.Cutoff)
	assert.Equal(t, 2, plan.Pruned)
	assert.Equal(t, 3, plan.Retained)
	assert.Equal(t, map[string]int{"v1.8.2": 1}, plan.Protected)
}

// TestBuildPlanPrereleaseOnly proves a prereleaseonly repo keeps every
// release, however old, and prunes only old pre-releases
func TestBuildPlanPrereleaseOnly(t *testing.T) {
	cfg := pkgConfig{
		AgeCutoff:      30 * 24 * time.Hour,
		PrereleaseOnly: true,
	}
	items := []pc.PackageDetail{
		pkg("5.13.2~rc5", 60*24*time.Hour),       // old pre-release: pruned
		pkg("2.3.0-unstable.1", 60*24*time.Hour), // old pre-release: pruned
		pkg("5.15.0~rc10", 24*time.Hour),         // fresh pre-release: retained
		pkg("2.0.0", 240*24*time.Hour),           // old release: retained
		pkg("5.8.10", 24*time.Hour),              // fresh release: retained
	}
	plan, err := BuildPlan("tyk-ee-unstable", cfg, testTracks, items, planNow, planGrace)
	require.NoError(t, err)

	assert.Equal(t, 2, plan.Pruned)
	assert.Equal(t, 3, plan.Retained)
	for _, pp := range plan.Packages {
		assert.Contains(t, []string{"5.13.2~rc5", "2.3.0-unstable.1"}, pp.Version)
	}
}

func TestFilterPrereleaseOnly(t *testing.T) {
	f := &Filter{Age: 30 * 24 * time.Hour, PrereleaseOnly: true, vtrans: strings.NewReplacer("~", "-")}
	assert.True(t, f.Satisfies(pkg("5.13.2~rc5", 60*24*time.Hour), planNow))
	assert.False(t, f.Satisfies(pkg("5.15.0~rc10", 24*time.Hour), planNow))
	assert.False(t, f.Satisfies(pkg("2.0.0", 240*24*time.Hour), planNow))
}

func TestFillPrunedBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.Header().Set("Content-Length", "1000")
	}))
	defer srv.Close()

	items := []pc.PackageDetail{
		pkg("1.6.9", 0),
		pkg("1.6.10", 0),
		pkg("2.0.0", 0), // retained: must not be sized
	}
	// same file under a second distro version: counted once
	dup := pkg("1.6.9", 0)
	dup.DistroVersion = "debian/bookworm"
	items = append(items, dup)
	for i := range items {
		items[i].DownloadURL = srv.URL + "/" + items[i].Filename
	}

	plan, err := BuildPlan("tyk-test", pkgConfig{VersionCutoff: "v1.7"}, testTracks, items, planNow, planGrace)
	require.NoError(t, err)
	require.Equal(t, 3, plan.Pruned)

	c := NewClient("test-token", "tyk", 100, 100)
	c.FillPrunedBytes(&plan, items, 4)
	assert.Equal(t, int64(2000), plan.PrunedBytes)
}

func TestBuildPlanBadTrack(t *testing.T) {
	_, err := BuildPlan("x", pkgConfig{Track: "nonesuch", Editions: []string{"ce"}}, testTracks, nil, planNow, planGrace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in the tracks config")

	_, err = BuildPlan("x", pkgConfig{Track: "gateway"}, testTracks, nil, planNow, planGrace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no editions")

	// anchor with no shipped packages: loud failure
	_, err = BuildPlan("x", pkgConfig{Track: "gateway", Editions: []string{"ce"}},
		testTracks, []pc.PackageDetail{pkg("1.0.0", 0)}, planNow, planGrace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no released packages")
}
