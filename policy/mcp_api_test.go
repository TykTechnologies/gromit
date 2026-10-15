package policy

import (
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMCPAPIRelease(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol))
	for _, repository := range []string{"tyk", "tyk-analytics", "tyk-pump", "tyk-sink"} {
		t.Run(repository, func(t *testing.T) {
			repo, err := pol.GetRepoPolicy(repository)
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch("master"))
			bundle, err := NewBundle(repo.Branchvals.Features)
			require.NoError(t, err)
			dir := t.TempDir()
			_, err = bundle.Render(repo, dir, nil)
			require.NoError(t, err)
			var workflow struct {
				Jobs map[string]struct {
					Needs    any
					If       string
					Outputs  map[string]string
					Strategy struct{ Matrix map[string]any }
					Steps    []struct {
						Name            string
						Uses            string
						With            map[string]any
						Run             string
						ContinueOnError any `yaml:"continue-on-error"`
					}
				}
			}
			require.NoError(t, yaml.Unmarshal([]byte(readRenderedFile(t, dir, ".github/workflows/release.yml")), &workflow))
			_, separate := workflow.Jobs["mcp-qualification"]
			assert.False(t, separate, "MCP must execute inside the regular API matrix")
			controller := workflow.Jobs["test-controller-api"]
			assert.Equal(t, "${{ steps.params.outputs.matrix }}", controller.Outputs["matrix"])
			assert.Equal(t, "${{ steps.mcp_images.outputs.manifest }}", controller.Outputs["mcp_manifest"])
			foundImages := false
			for _, step := range controller.Steps {
				if step.Name == "Set test parameters" {
					assert.Equal(t, "true", step.With["expand_api_matrix"])
					assert.Equal(t, "${{ github.event_name == 'pull_request' && 'true' || 'false' }}", step.With["include_mcp"])
				}
				if step.Name == "Resolve coordinated MCP API images" {
					foundImages = true
					topology := "direct"
					if repository == "tyk-sink" {
						topology = "mdcb"
					}
					assert.Equal(t, topology, step.With["topology"])
				}
			}
			assert.True(t, foundImages)
			api := workflow.Jobs["api-tests"]
			assert.Contains(t, api.If, "needs.test-controller-api.result == 'success'", "A failed image resolver must not launch unpinned shards")
			assert.Equal(t, map[string]any{"include": "${{ fromJson(needs.test-controller-api.outputs.matrix) }}"}, api.Strategy.Matrix)
			choose, setup, execute, report := -1, -1, -1, -1
			for i, step := range api.Steps {
				assert.NotContains(t, step.Run, "run_mcp_v2_stack.py")
				switch step.Name {
				case "Choose test code branch":
					choose = i
				case "Set up test environment":
					setup = i
					assert.Equal(t, "${{ matrix.envfiles.suite }}", step.With["suite"])
				case "Run API tests":
					execute = i
					assert.Equal(t, "${{ matrix.envfiles.apimarkers }}", step.With["api_markers"])
					assert.Equal(t, "${{ matrix.envfiles.shard_count || 1 }}", step.With["shard_count"])
				case "Generate test reports and collect logs":
					report = i
					assert.Equal(t, "${{ matrix.envfiles.suite != 'mcp' }}", step.ContinueOnError)
				}
				if strings.Contains(step.Uses, "github-actions/.github/actions/tests/") {
					assert.NotContains(t, step.Uses, "@production", "New API interfaces require the same immutable action revision")
				}
			}
			assert.True(t, choose >= 0 && choose < setup && setup < execute && execute < report)
			aggregate := workflow.Jobs["aggregator-ci-test"]
			dependencies, ok := aggregate.Needs.([]any)
			require.True(t, ok)
			assert.Contains(t, dependencies, "api-tests")
			assert.Contains(t, dependencies, "test-controller-api")
			assert.NotContains(t, dependencies, "mcp-qualification")
			assert.Equal(t, "Require complete MCP API evidence", aggregate.Steps[1].Name)
			assert.Equal(t, "Aggregate results", aggregate.Steps[2].Name)
		})
	}
}

func TestMCPAPIRejectsUnsupportedPolicy(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol))
	for _, name := range []string{"unsupported-repository", "missing-api", "duplicate-feature"} {
		t.Run(name, func(t *testing.T) {
			repo, err := pol.GetRepoPolicy("tyk-pump")
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch("master"))
			switch name {
			case "unsupported-repository":
				repo.Name = "tyk-sync"
			case "missing-api":
				repo.Branchvals.Tests = nil
			case "duplicate-feature":
				repo.Branchvals.Features = append(repo.Branchvals.Features, "mcp-qualification")
			}
			bundle, err := NewBundle(repo.Branchvals.Features)
			require.NoError(t, err)
			_, err = bundle.Render(repo, t.TempDir(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "mcp-api-tests")
		})
	}
}
