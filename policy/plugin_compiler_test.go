package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginCompilerNGReleaseSupplyChainMetadata(t *testing.T) {
	var pol Policies
	config.LoadConfig("")
	require.NoError(t, LoadRepoPolicies(&pol))

	repo, err := pol.GetRepoPolicy("tyk")
	require.NoError(t, err)
	require.NoError(t, repo.SetBranch("master"))

	bundle, err := NewBundle([]string{"plugin-compiler-ng"})
	require.NoError(t, err)

	outputDir := t.TempDir()
	_, err = bundle.Render(repo, outputDir, nil)
	require.NoError(t, err)

	buildWorkflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-build.yml")
	assert.Contains(t, buildWorkflow,
		"tykio/tyk-plugin-compiler,enable=${{ startsWith(github.ref, 'refs/tags/') }}")
	assert.Contains(t, buildWorkflow, "id: build-ng-base")
	assert.Contains(t, buildWorkflow,
		`base="tykio/tyk-plugin-compiler@${{ steps.build-ng-base.outputs.digest }}"`)
	assert.Contains(t, buildWorkflow,
		`base="${{ steps.login-ecr.outputs.registry }}/tyk-plugin-compiler@${{ steps.build-ng-base.outputs.digest }}"`)
	assert.Contains(t, buildWorkflow, `BASE_IMAGE=${{ steps.source-base.outputs.ref }}`)
	assert.Equal(t, 1, strings.Count(buildWorkflow, "docker/setup-buildx-action@"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "latest=false"))
	assert.Equal(t, 4, strings.Count(buildWorkflow, "sbom: true"))
	assert.Equal(t, 4, strings.Count(buildWorkflow, "provenance: mode=max"))
	assert.Equal(t, 2, strings.Count(buildWorkflow, "contents: read"))

	baseWorkflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-base.yml")
	assert.Contains(t, baseWorkflow, `BASE_IMAGE=${{ steps.source-base.outputs.ref }}`)
	assert.Equal(t, 1, strings.Count(baseWorkflow, "docker/setup-buildx-action@"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "sbom: true"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "provenance: mode=max"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "contents: read"))
}

func TestGatewayReleaseSupplyChainMetadata(t *testing.T) {
	var pol Policies
	config.LoadConfig("")
	require.NoError(t, LoadRepoPolicies(&pol))

	repo, err := pol.GetRepoPolicy("tyk")
	require.NoError(t, err)
	require.NoError(t, repo.SetBranch("master"))

	bundle, err := NewBundle([]string{"releng"})
	require.NoError(t, err)

	outputDir := t.TempDir()
	_, err = bundle.Render(repo, outputDir, nil)
	require.NoError(t, err)

	releaseWorkflow := readRenderedFile(t, outputDir, ".github/workflows/release.yml")
	assert.Equal(t, 6, strings.Count(releaseWorkflow, "provenance: mode=max"))
	assert.Equal(t, 6, strings.Count(releaseWorkflow, "sbom: true"))
	assert.NotContains(t, releaseWorkflow, "Attach base image VEX")
	assert.NotContains(t, releaseWorkflow, "cosign attest --yes --type openvex")
	assert.NotContains(t, releaseWorkflow, "sigstore/cosign-installer")
}

func readRenderedFile(t *testing.T, outputDir, name string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(contents)
}
