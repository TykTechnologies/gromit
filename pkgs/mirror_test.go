package pkgs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pc "github.com/tyklabs/packagecloud/api/v1"
)

type fakeStore struct {
	objects map[string][]byte
	shas    map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: make(map[string][]byte), shas: make(map[string]string)}
}

func (f *fakeStore) Head(_ context.Context, key string) (string, bool, error) {
	sha, ok := f.shas[key]
	return sha, ok, nil
}

func (f *fakeStore) Put(_ context.Context, key, sha string, body io.Reader, _ int64) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.objects[key] = b
	f.shas[key] = sha
	return nil
}

func (f *fakeStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := f.objects[key]
	if !ok {
		return nil, fmt.Errorf("%s not in store", key)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func shaOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// mirrorFixture serves one package's content over HTTP and returns the
// plan, live item and content for it
func mirrorFixture(t *testing.T, version string, content []byte) (PlanPackage, pc.PackageDetail) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(content)
	}))
	t.Cleanup(srv.Close)
	filename := "tyk-test_" + version + "_amd64.deb"
	pp := PlanPackage{
		Name:          "tyk-test",
		Version:       version,
		Arch:          "amd64",
		DistroVersion: "ubuntu/jammy",
		Filename:      filename,
		Sha256Sum:     shaOf(content),
	}
	item := pc.PackageDetail{
		Name:          "tyk-test",
		Version:       version,
		Arch:          "amd64",
		DistroVersion: "ubuntu/jammy",
		Filename:      filename,
		Sha256Sum:     shaOf(content),
		DownloadURL:   srv.URL + "/" + filename,
	}
	return pp, item
}

func TestMirrorPlanArchivesAndVerifies(t *testing.T) {
	content := []byte("deb-bytes-1.2.3")
	pp, item := mirrorFixture(t, "1.2.3", content)
	plan := Plan{Repo: "tyk-test-repo", Packages: []PlanPackage{pp}}
	store := newFakeStore()

	res := MirrorPlan(context.Background(), plan, []pc.PackageDetail{item}, store, true)
	require.True(t, res.Clean(), "missing: %v failed: %v", res.Missing, res.Failed)
	assert.Equal(t, 1, res.Mirrored)
	assert.Equal(t, 1, res.Verified)

	key := "tyk-test-repo/ubuntu/jammy/" + pp.Filename
	assert.Equal(t, content, store.objects[key])
	assert.Equal(t, pp.Sha256Sum, store.shas[key])

	res = MirrorPlan(context.Background(), plan, []pc.PackageDetail{item}, store, true)
	require.True(t, res.Clean())
	assert.Equal(t, 0, res.Mirrored)
	assert.Equal(t, 1, res.Skipped)
	assert.Equal(t, 1, res.Verified)
}

func TestMirrorPlanMissingFromRepo(t *testing.T) {
	pp, _ := mirrorFixture(t, "1.2.3", []byte("gone"))
	plan := Plan{Repo: "tyk-test-repo", Packages: []PlanPackage{pp}}

	res := MirrorPlan(context.Background(), plan, nil, newFakeStore(), false)
	assert.False(t, res.Clean())
	require.Len(t, res.Missing, 1)
	assert.Contains(t, res.Missing[0], "1.2.3")
}

func TestMirrorPlanChecksumMismatch(t *testing.T) {
	pp, item := mirrorFixture(t, "1.2.3", []byte("real-bytes"))
	pp.Sha256Sum = shaOf([]byte("expected-bytes"))
	item.Sha256Sum = pp.Sha256Sum // matches the plan, so it is "found"
	plan := Plan{Repo: "tyk-test-repo", Packages: []PlanPackage{pp}}
	store := newFakeStore()

	res := MirrorPlan(context.Background(), plan, []pc.PackageDetail{item}, store, false)
	assert.False(t, res.Clean())
	require.Len(t, res.Failed, 1)
	assert.Empty(t, store.objects, "nothing must be archived on mismatch")
}

func TestMirrorPlanNeverOverwrites(t *testing.T) {
	content := []byte("deb-bytes")
	pp, item := mirrorFixture(t, "1.2.3", content)
	plan := Plan{Repo: "tyk-test-repo", Packages: []PlanPackage{pp}}
	store := newFakeStore()

	key := "tyk-test-repo/ubuntu/jammy/" + pp.Filename
	other := []byte("other-bytes")
	require.NoError(t, store.Put(context.Background(), key, shaOf(other), bytes.NewReader(other), int64(len(other))))

	res := MirrorPlan(context.Background(), plan, []pc.PackageDetail{item}, store, false)
	assert.False(t, res.Clean())
	require.Len(t, res.Failed, 1)
	assert.Equal(t, other, store.objects[key], "existing object must be untouched")
}
