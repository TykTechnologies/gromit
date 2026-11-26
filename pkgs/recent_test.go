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

var recentSince = time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)

func recentClient(t *testing.T, mux *http.ServeMux) *HubClient {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := NewHubClient("", 100, 100)
	c.token, c.username = "", ""
	c.base, c.auth, c.registry = srv.URL, srv.URL, srv.URL
	return c
}

func encode(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }

func TestRecentImages(t *testing.T) {
	repoPages := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/namespaces/tykio/repositories", func(w http.ResponseWriter, r *http.Request) {
		repoPages++
		switch r.URL.Query().Get("page") {
		case "":
			assert.Equal(t, "last_updated", r.URL.Query().Get("ordering"))
			encode(w, hubRepoPage{
				Next: "http://" + r.Host + "/v2/namespaces/tykio/repositories?page=2",
				Results: []hubRepoInfo{
					{Name: "tyk-gateway", LastUpdated: "2026-11-25T19:00:00Z"},
					{Name: "internal", LastUpdated: "2026-11-25T18:00:00Z", IsPrivate: true},
					{Name: "hand-pushed", LastUpdated: ""},
				},
			})
		case "2":
			// Older than the cutoff: listing must stop here.
			encode(w, hubRepoPage{
				Next:    "http://" + r.Host + "/v2/namespaces/tykio/repositories?page=3",
				Results: []hubRepoInfo{{Name: "stale", LastUpdated: "2026-11-01T00:00:00Z"}},
			})
		default:
			t.Errorf("paged past the cutoff: %s", r.URL)
		}
	})
	mux.HandleFunc("/v2/repositories/tykio/internal/tags", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("listed a private repository")
	})
	mux.HandleFunc("/v2/repositories/tykio/tyk-gateway/tags", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			encode(w, hubTagPage{Results: []hubTagEntry{
				{Name: "v5.13.7", Digest: "sha256:old", TagLastPushed: "2026-11-20T00:00:00Z", LastUpdated: "2026-11-20T00:00:00Z"},
			}})
			return
		}
		assert.Equal(t, "last_updated", r.URL.Query().Get("ordering"))
		encode(w, hubTagPage{
			Next: "http://" + r.Host + "/v2/repositories/tykio/tyk-gateway/tags?page=2",
			Results: []hubTagEntry{
				{Name: "v5.13.8", Digest: "sha256:new", TagLastPushed: "2026-11-25T19:50:14Z", LastUpdated: "2026-11-25T19:50:14Z", LastUpdater: "ci-bot"},
				{Name: "v5.13", Digest: "sha256:new", TagLastPushed: "2026-11-25T19:50:12Z", LastUpdated: "2026-11-25T19:50:12Z", LastUpdater: "ci-bot"},
				// No times at all: counts as recent and must not end the listing.
				{Name: "odd", Digest: "sha256:odd"},
				// Updated recently but pushed before the cutoff: skipped, listing goes on.
				{Name: "retagged-meta", Digest: "sha256:meta", TagLastPushed: "2026-11-10T00:00:00Z", LastUpdated: "2026-11-25T12:00:00Z"},
				{Name: "v5.13.8-rc3", Digest: "sha256:rc", TagLastPushed: "2026-11-25T15:13:22Z", LastUpdated: "2026-11-25T15:13:22Z", LastUpdater: "someone"},
			},
		})
	})
	mux.HandleFunc("/v2/repositories/tykio/hand-pushed/tags", func(w http.ResponseWriter, r *http.Request) {
		encode(w, hubTagPage{Results: []hubTagEntry{
			{Name: "latest", TagLastPushed: "2026-11-25T08:00:00Z", LastUpdated: "2026-11-25T08:00:00Z", LastUpdater: "someone"},
		}})
	})
	mux.HandleFunc("/v2/repositories/tykio/hand-pushed/tags/latest", func(w http.ResponseWriter, r *http.Request) {
		encode(w, hubTagEntry{Name: "latest"})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		encode(w, map[string]string{"token": "reg-token"})
	})
	mux.HandleFunc("/v2/tykio/hand-pushed/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	res, err := recentClient(t, mux).RecentImages("tykio", recentSince, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, repoPages)
	assert.Empty(t, res.Failed)

	var refs []string
	for _, img := range res.Images {
		refs = append(refs, img.Ref)
	}
	assert.Equal(t, []string{
		"docker.io/tykio/tyk-gateway@sha256:new",
		"docker.io/tykio/tyk-gateway@sha256:odd",
		"docker.io/tykio/tyk-gateway@sha256:rc",
		// No digest anywhere: scanned by tag instead of dropped.
		"docker.io/tykio/hand-pushed:latest",
	}, refs)

	gw := res.Images[0]
	assert.Equal(t, []string{"v5.13", "v5.13.8"}, gw.Tags)
	assert.Equal(t, []string{"ci-bot"}, gw.PushedBy)
	assert.Equal(t, time.Date(2026, 11, 25, 19, 50, 14, 0, time.UTC), gw.LastPushed)
	assert.Equal(t, []string{"someone"}, res.Images[2].PushedBy)
}

func TestRecentImagesKeepsGoingWhenOneRepoFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/namespaces/tykio/repositories", func(w http.ResponseWriter, r *http.Request) {
		encode(w, hubRepoPage{Results: []hubRepoInfo{
			{Name: "gone", LastUpdated: "2026-11-25T19:00:00Z"},
			{Name: "ok", LastUpdated: "2026-11-25T18:00:00Z"},
		}})
	})
	mux.HandleFunc("/v2/repositories/tykio/gone/tags", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	mux.HandleFunc("/v2/repositories/tykio/ok/tags", func(w http.ResponseWriter, r *http.Request) {
		encode(w, hubTagPage{Results: []hubTagEntry{
			{Name: "v1", Digest: "sha256:1", TagLastPushed: "2026-11-25T18:00:00Z", LastUpdated: "2026-11-25T18:00:00Z"},
		}})
	})

	res, err := recentClient(t, mux).RecentImages("tykio", recentSince, 2)
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Contains(t, res.Failed[0], "tykio/gone")
	require.Len(t, res.Images, 1)
	assert.Equal(t, "docker.io/tykio/ok@sha256:1", res.Images[0].Ref)
}

func TestRecentImagesRepoListingError(t *testing.T) {
	res, err := recentClient(t, http.NewServeMux()).RecentImages("tykio", recentSince, 2)
	assert.ErrorContains(t, err, "listing tykio repositories")
	assert.NotNil(t, res.Images)
}

func TestRecentImagesRejectsBadInput(t *testing.T) {
	c := NewHubClient("", 100, 100)
	for _, org := range []string{"", "tykio/tyk-gateway"} {
		_, err := c.RecentImages(org, recentSince, 2)
		assert.Error(t, err, org)
	}
	_, err := c.RecentImages("tykio", recentSince, 0)
	assert.ErrorContains(t, err, "concurrency")
}
