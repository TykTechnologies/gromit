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
	assert.Equal(t, 3, strings.Count(buildWorkflow, "WITH_GATEWAY_SELFTEST=1"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "WITH_GATEWAY_SELFTEST=0"))

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

func TestPluginCompilerNGReleaseBranchConfiguration(t *testing.T) {
	tests := []struct {
		branch           string
		goImage          string
		variantCount     int
		usesBoringCrypto bool
	}{
		{branch: "release-5.3", goImage: "1.23-bullseye", variantCount: 1},
		{branch: "release-5.8", goImage: "1.25-bullseye", variantCount: 3, usesBoringCrypto: true},
		{branch: "release-5.8.15", goImage: "1.25-bullseye", variantCount: 3, usesBoringCrypto: true},
	}

	for _, test := range tests {
		t.Run(test.branch, func(t *testing.T) {
			var pol Policies
			config.LoadConfig("")
			require.NoError(t, LoadRepoPolicies(&pol))

			repo, err := pol.GetRepoPolicy("tyk")
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch(test.branch))

			bundle, err := NewBundle([]string{"plugin-compiler-ng"})
			require.NoError(t, err)

			outputDir := t.TempDir()
			_, err = bundle.Render(repo, outputDir, nil)
			require.NoError(t, err)

			workflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-build.yml")
			assert.Contains(t, workflow, "GOLANG_CROSS: "+test.goImage)
			assert.Equal(t, test.variantCount, strings.Count(workflow, "latest=false"))
			assert.Equal(t, test.variantCount, strings.Count(workflow, "WITH_GATEWAY_SELFTEST=1"))
			assert.Equal(t, test.variantCount, strings.Count(workflow, "WITH_GATEWAY_SELFTEST=0"))
			if test.usesBoringCrypto {
				assert.Contains(t, workflow, "FIPS_GOEXPERIMENT=boringcrypto")
				assert.Contains(t, workflow, "SELFTEST_GOEXPERIMENT=boringcrypto")
			} else {
				assert.NotContains(t, workflow, "boringcrypto")
			}
		})
	}
}

func readRenderedFile(t *testing.T, outputDir, name string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(contents)
}
