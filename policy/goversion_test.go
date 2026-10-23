package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TykTechnologies/gromit/config"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	assert.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	assert.NoError(t, os.WriteFile(p, []byte(body), 0o644))
}

const releaseYML = `jobs:
  goreleaser:
    strategy:
      matrix:
        golang_cross:
          - 1.27-bullseye
          - 1.27-bullseye-pgo
        include:
          - golang_cross: 1.26-bullseye
`

func TestAdoptBuildenv(t *testing.T) {
	for _, tt := range []struct {
		name   string
		files  map[string]string
		config string
		dir    func(string) string
		want   string
	}{
		{
			name:   "the checkout wins over the config",
			files:  map[string]string{".github/workflows/release.yml": releaseYML},
			config: "1.26-bullseye",
			want:   "1.27-bullseye", // the first entry of the matrix: not the -pgo one, not the include
		},
		{
			name:   "no config at all: the repository carries the version",
			files:  map[string]string{".github/workflows/release.yml": releaseYML},
			config: "",
			want:   "1.27-bullseye",
		},
		{
			name:   "quoted tag",
			files:  map[string]string{".github/workflows/release.yml": "matrix:\n  golang_cross:\n    - '1.28-bookworm'\n"},
			config: "1.26-bookworm",
			want:   "1.28-bookworm",
		},
		{
			name:   "gromit.env when there is no release.yml",
			files:  map[string]string{"ci/gromit.env": "FOO=bar\nGO_IMAGE=tykio/golang-cross:1.27-bookworm\n"},
			config: "1.26-bookworm",
			want:   "1.27-bookworm",
		},
		{
			name:   "release.yml wins over gromit.env",
			files:  map[string]string{".github/workflows/release.yml": releaseYML, "ci/gromit.env": "GO_IMAGE=tykio/golang-cross:1.20-bullseye\n"},
			config: "",
			want:   "1.27-bullseye",
		},
		{
			name:   "a checkout without a tag takes the config value: a new repository",
			files:  map[string]string{"README.md": "x"},
			config: "1.26-bullseye",
			want:   "1.26-bullseye",
		},
		{
			name:   "a release.yml that has no matrix list is not a tag",
			files:  map[string]string{".github/workflows/release.yml": "jobs:\n  a:\n    include:\n      - golang_cross: 1.26-bullseye\n"},
			config: "1.25-bullseye",
			want:   "1.25-bullseye",
		},
		{
			name:   "no tag in the checkout and none in the config: rendered as before, with an empty value",
			files:  map[string]string{"README.md": "x"},
			config: "",
			want:   "",
		},
		{
			name:   "CRLF line endings",
			files:  map[string]string{".github/workflows/release.yml": "matrix:\r\n  golang_cross:\r\n    - 1.27-bullseye\r\n"},
			config: "1.26-bullseye",
			want:   "1.27-bullseye",
		},
		{
			name:   "rendering to stdout has no checkout: the config value",
			files:  map[string]string{".github/workflows/release.yml": releaseYML},
			config: "1.26-bullseye",
			dir:    func(string) string { return "-" },
			want:   "1.26-bullseye",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range tt.files {
				write(t, dir, name, body)
			}
			if tt.dir != nil {
				dir = tt.dir(dir)
			}
			rp := RepoPolicy{Name: "tyk", Branch: "master", Branchvals: branchVals{Buildenv: tt.config}}
			assert.NoError(t, rp.AdoptBuildenv(dir))
			assert.Equal(t, tt.want, rp.Branchvals.Buildenv)
		})
	}
}

// Adopting is idempotent: rendering the adopted value gives the file it came from, so the drift check
// has nothing to report when a pull request changes the Go version in the repository.
func TestAdoptBuildenvTwice(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".github/workflows/release.yml", releaseYML)
	rp := RepoPolicy{Name: "tyk", Branch: "master", Branchvals: branchVals{Buildenv: "1.26-bullseye"}}
	assert.NoError(t, rp.AdoptBuildenv(dir))
	assert.NoError(t, rp.AdoptBuildenv(dir))
	assert.Equal(t, "1.27-bullseye", rp.Branchvals.Buildenv)
}

// Every file gromit renders the golang-cross tag into carries the tag of the checkout: nothing in the
// policy (the config, a plugin compiler setting) sets a Go version of its own.
func TestAdoptedBuildenvReachesEveryRenderedFile(t *testing.T) {
	checkout := t.TempDir()
	write(t, checkout, ".github/workflows/release.yml", releaseYML)

	var pol Policies
	config.LoadConfig("")
	require.NoError(t, LoadRepoPolicies(&pol))
	repo, err := pol.GetRepoPolicy("tyk")
	require.NoError(t, err)
	require.NoError(t, repo.SetBranch("release-5.3"))
	require.NoError(t, repo.AdoptBuildenv(checkout))
	require.Equal(t, "1.27-bullseye", repo.Branchvals.Buildenv)

	bundle, err := NewBundle([]string{"plugin-compiler", "plugin-compiler-ng"})
	require.NoError(t, err)
	out := t.TempDir()
	_, err = bundle.Render(repo, out, nil)
	require.NoError(t, err)

	for file, want := range map[string]string{
		".github/workflows/plugin-compiler-build.yml":     "GOLANG_CROSS: 1.27-bullseye",
		".github/workflows/plugin-compiler-ng-build.yml":  "GOLANG_CROSS: 1.27-bullseye",
		"ci/images/plugin-compiler/Dockerfile":            "tykio/golang-cross:1.27-bullseye",
		"ci/images/plugin-compiler-ng/Dockerfile.release": "tykio/golang-cross:1.27-bullseye",
	} {
		assert.Contains(t, readRenderedFile(t, out, file), want, file)
	}
}
