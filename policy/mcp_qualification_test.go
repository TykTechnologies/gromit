package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMCPQualificationRelease(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol))

	for _, repository := range []string{"tyk", "tyk-analytics", "tyk-pump"} {
		for _, enabled := range []bool{false, true} {
			name := repository + "/disabled"
			if enabled {
				name = repository + "/enabled"
			}
			t.Run(name, func(t *testing.T) {
				repo, err := pol.GetRepoPolicy(repository)
				require.NoError(t, err)
				require.NoError(t, repo.SetBranch("master"))
				if enabled {
					repo.Branchvals.Features = append(repo.Branchvals.Features, "mcp-qualification")
				}
				bundle, err := NewBundle(repo.Branchvals.Features)
				require.NoError(t, err)
				outputDir := t.TempDir()
				_, err = bundle.Render(repo, outputDir, nil)
				require.NoError(t, err)

				var workflow struct {
					On struct {
						PullRequest struct {
							Types []string
						} `yaml:"pull_request"`
					}
					Jobs map[string]struct {
						Uses        string
						Needs       any
						If          string
						Secrets     any
						Permissions map[string]string
						Steps       []struct {
							Name string
							With map[string]string
						}
					}
				}
				require.NoError(t, yaml.Unmarshal([]byte(readRenderedFile(t, outputDir,
					".github/workflows/release.yml")), &workflow))
				// Leaving draft must start existing CI even when MCP is disabled.
				assert.Equal(t, []string{"opened", "synchronize", "reopened", "ready_for_review", "labeled"},
					workflow.On.PullRequest.Types)
				qualification, exists := workflow.Jobs["mcp-qualification"]
				assert.Equal(t, enabled, exists)
				aggregate, exists := workflow.Jobs["aggregator-ci-test"]
				require.True(t, exists)
				dependencies, ok := aggregate.Needs.([]any)
				require.True(t, ok)
				// The serial suite must supplement all existing required jobs.
				for _, dependency := range []string{"dep-guard", "goreleaser", "api-tests"} {
					assert.Contains(t, dependencies, dependency)
				}
				if repository == "tyk-analytics" {
					assert.Contains(t, dependencies, "ui-tests")
				}
				api, exists := workflow.Jobs["api-tests"]
				require.True(t, exists)
				foundAPI := false
				for _, step := range api.Steps {
					if step.Name != "Run API tests" {
						continue
					}
					foundAPI = true
					markers, filtered := step.With["api_markers"]
					assert.Equal(t, enabled, filtered,
						"MCP may only leave the parallel suite when serial qualification is required")
					if enabled {
						assert.Equal(t, "(${{ matrix.envfiles.apimarkers }}) and not mcp", markers)
					}
				}
				require.True(t, foundAPI)
				if enabled {
					assert.Regexp(t, `^TykTechnologies/github-actions/\.github/workflows/mcp-qualification\.yml@[0-9a-f]{40}$`, qualification.Uses)
					assert.Equal(t, "dep-guard", qualification.Needs)
					assert.Equal(t, "github.event_name == 'pull_request' && github.event.pull_request.draft == false", qualification.If)
					assert.Equal(t, map[string]any{
						"PROBE_APP_ID":          "${{ secrets.PROBE_APP_ID }}",
						"PROBE_APP_PRIVATE_KEY": "${{ secrets.PROBE_APP_PRIVATE_KEY }}",
						"DASH_LICENSE":          "${{ secrets.DASH_LICENSE }}",
					}, qualification.Secrets)
					assert.Equal(t, map[string]string{"contents": "read"}, qualification.Permissions)
					assert.Contains(t, dependencies, "mcp-qualification")
				} else {
					assert.NotContains(t, dependencies, "mcp-qualification")
				}
				if repository == "tyk-analytics" && enabled {
					// Nightly has no serial replacement; its existing MCP selection stays intact.
					nightly := readRenderedFile(t, outputDir, ".github/workflows/nightly-e2e-tests.yml")
					assert.NotContains(t, nightly, "and not mcp")
				}
			})
		}
	}
}

func TestMCPQualificationRejectsUnsupportedPolicy(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol))
	for _, test := range []struct {
		name    string
		mutate  func(*RepoPolicy)
		message string
	}{
		{"unsupported-repository", func(repo *RepoPolicy) { repo.Name = "tyk-sink" }, "supports only tyk"},
		{"missing-api-suite", func(repo *RepoPolicy) { repo.Branchvals.Tests = nil }, "requires the api test job"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, err := pol.GetRepoPolicy("tyk-pump")
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch("master"))
			repo.Branchvals.Features = append(repo.Branchvals.Features, "mcp-qualification")
			test.mutate(&repo)
			bundle, err := NewBundle(repo.Branchvals.Features)
			require.NoError(t, err)
			_, err = bundle.Render(repo, t.TempDir(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}
