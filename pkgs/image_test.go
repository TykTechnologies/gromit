package pkgs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsReleaseTag(t *testing.T) {
	assert.True(t, IsReleaseTag("v1.8.0"))
	assert.True(t, IsReleaseTag("v0.4.0"))
	assert.False(t, IsReleaseTag("v1.8"))
	assert.False(t, IsReleaseTag("v1"))
	assert.False(t, IsReleaseTag("v1.8.0-rc4"))
	assert.False(t, IsReleaseTag("v1.4.3-amd64"))
	assert.False(t, IsReleaseTag("v1.5rc3"))
	assert.False(t, IsReleaseTag("s1.2.4"))
	assert.False(t, IsReleaseTag("latest"))
}

func tag(name string, age time.Duration) ImageTag {
	return ImageTag{
		Name:       name,
		Digest:     "sha256:" + name,
		LastPushed: planNow.Add(-age),
	}
}

func TestBuildImagePlanAgeCutoff(t *testing.T) {
	cfg := pkgConfig{
		Dhrepo:     "tykio/tyk-identity-broker",
		AgeCutoff:  3 * 365 * 24 * time.Hour,
		Exceptions: []string{"v1.1.0"},
		Name:       "Tyk Identity Broker",
	}
	tags := []ImageTag{
		tag("v1.8.0", 24*time.Hour),
		tag("v1.4.2", 300*24*time.Hour), // ~10 months: kept
		tag("v1.4.1", 4*365*24*time.Hour),
		tag("v1.1.0", 5*365*24*time.Hour), // old, protected
		tag("v1.8", 24*time.Hour),         // alias
		tag("v1.8.0-rc4", 4*365*24*time.Hour),
		tag("latest", 6*365*24*time.Hour),
	}
	plan, err := BuildImagePlan("tyk-identity-broker", cfg.Dhrepo, cfg, testTracks, tags, planNow, planGrace)
	require.NoError(t, err)

	assert.Equal(t, "tykio/tyk-identity-broker", plan.Image)
	assert.Equal(t, "Tyk Identity Broker", plan.Product)
	assert.Equal(t, planNow.AddDate(0, 0, 30), plan.NotBefore)
	assert.Equal(t, 1, plan.Pruned)
	assert.Equal(t, 6, plan.Retained)
	assert.Equal(t, 3, plan.NonRelease)
	assert.Equal(t, map[string]int{"v1.1.0": 1}, plan.Protected)
	assert.Equal(t, map[string]int{"v1.4": 1}, plan.PrunedSeries)
	require.Len(t, plan.Tags, 1)
	assert.Equal(t, "v1.4.1", plan.Tags[0].Tag)
	assert.Equal(t, "sha256:v1.4.1", plan.Tags[0].Digest)
}

func TestBuildImagePlanTrackDriven(t *testing.T) {
	cfg := pkgConfig{
		Dhrepo:   "tykio/tyk-gateway",
		Track:    "gateway",
		Editions: []string{"ce", "ee"},
	}
	tags := []ImageTag{
		tag("v2.8.3", 8*365*24*time.Hour),
		tag("v3.0.8", 6*365*24*time.Hour),
		tag("v5.2.0", 3*365*24*time.Hour),
		tag("v5.3.0", 3*365*24*time.Hour),
		tag("v5.8.1", 365*24*time.Hour),
		tag("v5.14.0", 24*time.Hour),
		tag("v5.14", 24*time.Hour),
	}
	plan, err := BuildImagePlan("tyk-gateway", cfg.Dhrepo, cfg, testTracks, tags, planNow, planGrace)
	require.NoError(t, err)
	assert.False(t, plan.NeverMirror)

	fips, err := BuildImagePlan("tyk-gateway", "tykio/tyk-gateway-fips", cfg, testTracks, tags, planNow, planGrace)
	require.NoError(t, err)
	assert.True(t, fips.NeverMirror)
	assert.Equal(t, "tykio/tyk-gateway-fips", fips.Image)
	assert.Equal(t, "5.8", plan.Anchor)
	assert.Equal(t, "v3.0", plan.Cutoff)
	assert.Equal(t, 1, plan.Pruned)
	assert.Equal(t, "v2.8.3", plan.Tags[0].Tag)
	assert.Equal(t, 1, plan.NonRelease)
}

func TestDigestFromEntry(t *testing.T) {
	assert.Equal(t, "sha256:tag", digestFromEntry(hubTagEntry{Digest: "sha256:tag"}))
	assert.Equal(t, "sha256:one", digestFromEntry(hubTagEntry{
		Images: []hubImage{{Digest: "sha256:one", Architecture: "amd64"}},
	}))
	assert.Empty(t, digestFromEntry(hubTagEntry{
		Images: []hubImage{
			{Digest: "sha256:amd", Architecture: "amd64"},
			{Digest: "sha256:arm", Architecture: "arm64"},
		},
	}))
}

func TestListTagsPaginatesAndFillsDigest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/repositories/tykio/tyk-identity-broker/tags", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "2" {
			_ = json.NewEncoder(w).Encode(hubTagPage{
				Results: []hubTagEntry{{
					Name:          "v1.3.1",
					TagLastPushed: "2022-05-31T00:00:00.000000Z",
				}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(hubTagPage{
			Next: "http://" + r.Host + "/v2/repositories/tykio/tyk-identity-broker/tags?page=2",
			Results: []hubTagEntry{{
				Name:          "v1.8.0",
				Digest:        "sha256:abc",
				TagLastPushed: "2026-08-24T16:22:51.388533Z",
			}},
		})
	})
	mux.HandleFunc("/v2/repositories/tykio/tyk-identity-broker/tags/v1.3.1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(hubTagEntry{
			Name: "v1.3.1",
			Images: []hubImage{
				{Digest: "sha256:amd", Architecture: "amd64"},
				{Digest: "sha256:arm", Architecture: "arm64"},
			},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "reg-token"})
	})
	mux.HandleFunc("/v2/tykio/tyk-identity-broker/manifests/v1.3.1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		assert.Equal(t, "Bearer reg-token", r.Header.Get("Authorization"))
		w.Header().Set("Docker-Content-Digest", "sha256:list")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewHubClient("", 100, 100)
	c.base = srv.URL
	c.auth = srv.URL
	c.registry = srv.URL
	tags, err := c.ListTags("tykio/tyk-identity-broker")
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "sha256:abc", tags[0].Digest)
	assert.Empty(t, tags[1].Digest)

	c.FillMissingDigests("tykio/tyk-identity-broker", tags, 2)
	assert.Equal(t, "sha256:list", tags[1].Digest)

	plan := ImagePlan{Tags: []PlanImage{{Tag: "v1.3.1"}}}
	require.NoError(t, c.FillPlanDigests("tykio/tyk-identity-broker", &plan, 2))
	assert.Equal(t, "sha256:list", plan.Tags[0].Digest)
}

func TestFillPlanDigestsErrorsWhenMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewHubClient("", 100, 100)
	c.base = srv.URL
	c.auth = srv.URL
	c.registry = srv.URL
	plan := ImagePlan{Tags: []PlanImage{{Tag: "v1.0.0"}}}
	err := c.FillPlanDigests("tykio/tyk-identity-broker", &plan, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "v1.0.0")
}

func TestResolveImageArg(t *testing.T) {
	r := Repos{
		"tyk-gateway": {
			Dhrepos: []string{"tykio/tyk-gateway", "tykio/tyk-gateway-ee", "tykio/tyk-gateway-fips"},
		},
		"tyk-identity-broker": {Dhrepo: "tykio/tyk-identity-broker"},
	}
	name, cfg, images, err := r.ResolveImageArg("tyk-gateway")
	require.NoError(t, err)
	assert.Equal(t, "tyk-gateway", name)
	assert.Equal(t, cfg.Dhrepos, images)

	name, _, images, err = r.ResolveImageArg("tyk-gateway-ee")
	require.NoError(t, err)
	assert.Equal(t, "tyk-gateway", name)
	assert.Equal(t, []string{"tykio/tyk-gateway-ee"}, images)

	name, _, images, err = r.ResolveImageArg("tykio/tyk-identity-broker")
	require.NoError(t, err)
	assert.Equal(t, "tyk-identity-broker", name)
	assert.Equal(t, []string{"tykio/tyk-identity-broker"}, images)

	_, _, _, err = r.ResolveImageArg("nonesuch")
	require.Error(t, err)
}
