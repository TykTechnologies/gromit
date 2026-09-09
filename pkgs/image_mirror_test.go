package pkgs

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeImages struct {
	present map[string]bool
	blobs   map[string][]byte
}

func (f *fakeImages) Exists(_ context.Context, _, digest string) (bool, error) {
	return f.present[digest], nil
}

func (f *fakeImages) Fetch(_ context.Context, _, _, digest string) (string, error) {
	b, ok := f.blobs[digest]
	if !ok {
		return "", fmt.Errorf("no blob for %s", digest)
	}
	tmp, err := os.CreateTemp("", "fake-image-*.tar")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), tmp.Close()
}

func (f *fakeImages) Check(path, digest string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	want, ok := f.blobs[digest]
	if !ok || !bytes.Equal(b, want) {
		return fmt.Errorf("archive does not contain digest %s", digest)
	}
	return nil
}

func TestImageArchiveKey(t *testing.T) {
	key := ImageArchiveKey("tykio/tyk-identity-broker", "v1.4.1")
	assert.Equal(t, "images/tykio/tyk-identity-broker/v1.4.1.tar", key)
	image, tag, ok := ParseImageArchiveKey(key)
	assert.True(t, ok)
	assert.Equal(t, "tykio/tyk-identity-broker", image)
	assert.Equal(t, "v1.4.1", tag)

	_, _, ok = ParseImageArchiveKey("tyk-gateway/ubuntu/focal/tyk-gateway_4.2.3_amd64.deb")
	assert.False(t, ok)
	_, _, ok = ParseImageArchiveKey("images/notag.tar")
	assert.False(t, ok)
}

func TestMirrorImagePlanArchivesAndVerifies(t *testing.T) {
	digest := "sha256:abc"
	blob := []byte("oci-bytes")
	src := &fakeImages{
		present: map[string]bool{digest: true},
		blobs:   map[string][]byte{digest: blob},
	}
	plan := ImagePlan{
		Image: "tykio/tyk-identity-broker",
		Tags:  []PlanImage{{Tag: "v1.4.1", Digest: digest}},
	}
	store := newFakeStore()

	res := MirrorImagePlan(context.Background(), plan, src, store, true)
	require.True(t, res.Clean(), "missing: %v failed: %v", res.Missing, res.Failed)
	assert.Equal(t, 1, res.Mirrored)
	assert.Equal(t, 1, res.Verified)

	key := ImageArchiveKey(plan.Image, "v1.4.1")
	assert.Equal(t, blob, store.objects[key])
	assert.Equal(t, digest, store.shas[key])

	res = MirrorImagePlan(context.Background(), plan, src, store, true)
	require.True(t, res.Clean())
	assert.Equal(t, 0, res.Mirrored)
	assert.Equal(t, 1, res.Skipped)
	assert.Equal(t, 1, res.Verified)
}

func TestMirrorImagePlanMissingFromHub(t *testing.T) {
	plan := ImagePlan{
		Image: "tykio/tyk-identity-broker",
		Tags:  []PlanImage{{Tag: "v1.4.1", Digest: "sha256:gone"}},
	}
	res := MirrorImagePlan(context.Background(), plan, &fakeImages{}, newFakeStore(), false)
	assert.False(t, res.Clean())
	require.Len(t, res.Missing, 1)
	assert.Contains(t, res.Missing[0], "v1.4.1")
}

func TestMirrorImagePlanEmptyDigest(t *testing.T) {
	plan := ImagePlan{
		Image: "tykio/tyk-identity-broker",
		Tags:  []PlanImage{{Tag: "v1.4.1"}},
	}
	res := MirrorImagePlan(context.Background(), plan, &fakeImages{}, newFakeStore(), false)
	assert.False(t, res.Clean())
	require.Len(t, res.Failed, 1)
}

func TestMirrorImagePlanNeverMirror(t *testing.T) {
	plan := ImagePlan{
		Image:       "tykio/tyk-gateway-fips",
		NeverMirror: true,
		Tags:        []PlanImage{{Tag: "v5.2.0", Digest: "sha256:fips"}},
	}
	store := newFakeStore()
	res := MirrorImagePlan(context.Background(), plan, &fakeImages{}, store, true)
	require.True(t, res.Clean())
	assert.Equal(t, 1, res.Skipped)
	assert.Equal(t, 0, res.Mirrored)
	assert.Empty(t, store.objects)
}

func TestMirrorImagePlanNeverOverwrites(t *testing.T) {
	digest := "sha256:abc"
	blob := []byte("oci-bytes")
	src := &fakeImages{
		present: map[string]bool{digest: true},
		blobs:   map[string][]byte{digest: blob},
	}
	plan := ImagePlan{
		Image: "tykio/tyk-identity-broker",
		Tags:  []PlanImage{{Tag: "v1.4.1", Digest: digest}},
	}
	store := newFakeStore()
	key := ImageArchiveKey(plan.Image, "v1.4.1")
	other := []byte("other-bytes")
	require.NoError(t, store.Put(context.Background(), key, "sha256:other", bytes.NewReader(other), int64(len(other))))

	res := MirrorImagePlan(context.Background(), plan, src, store, false)
	assert.False(t, res.Clean())
	require.Len(t, res.Failed, 1)
	assert.Equal(t, other, store.objects[key])
}

func TestCheckImageArchiveRoundTrip(t *testing.T) {
	img, err := random.Image(256, 1)
	require.NoError(t, err)
	digest, err := img.Digest()
	require.NoError(t, err)

	dir := t.TempDir()
	p, err := layout.Write(dir, empty.Index)
	require.NoError(t, err)
	require.NoError(t, p.AppendImage(img))

	tmp, err := os.CreateTemp(t.TempDir(), "oci-*.tar")
	require.NoError(t, err)
	require.NoError(t, tarDirectory(dir, tmp))
	require.NoError(t, tmp.Close())

	require.NoError(t, CheckImageArchive(tmp.Name(), digest.String()))
	err = CheckImageArchive(tmp.Name(), "sha256:"+strings.Repeat("0", 64))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain digest")
}

func TestHubClientFetchFromRegistry(t *testing.T) {
	srv := httptest.NewServer(registry.New())
	defer srv.Close()

	img, err := random.Image(256, 1)
	require.NoError(t, err)
	digest, err := img.Digest()
	require.NoError(t, err)

	host := strings.TrimPrefix(srv.URL, "http://")
	ref, err := name.NewTag(host+"/tykio/test:v1.0.0", name.Insecure)
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))

	c := NewHubClient("", 100, 100)
	c.token = ""
	c.username = ""
	c.registry = srv.URL
	ctx := context.Background()

	exists, err := c.Exists(ctx, "tykio/test", digest.String())
	require.NoError(t, err)
	assert.True(t, exists)

	missing, err := c.Exists(ctx, "tykio/test", "sha256:"+strings.Repeat("0", 64))
	require.NoError(t, err)
	assert.False(t, missing)

	path, err := c.Fetch(ctx, "tykio/test", "v1.0.0", digest.String())
	require.NoError(t, err)
	defer os.Remove(path)
	require.NoError(t, c.Check(path, digest.String()))
}
