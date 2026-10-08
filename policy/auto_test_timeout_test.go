package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUITestTimeoutBudget(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol))

	for _, test := range []struct {
		name       string
		repository string
		branch     string
		workflow   string
		apiTimeout int
	}{
		{"dashboard-release", "tyk-analytics", "master", "release.yml", 25},
		{"dashboard-nightly", "tyk-analytics", "master", "nightly-e2e-tests.yml", 30},
		{"automation", "tyk-pro", "main", "test-square.yml", 25},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, err := pol.GetRepoPolicy(test.repository)
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch(test.branch))
			bundle, err := NewBundle(repo.Branchvals.Features)
			require.NoError(t, err)
			outputDir := t.TempDir()
			_, err = bundle.Render(repo, outputDir, nil)
			require.NoError(t, err)

			var workflow struct {
				Jobs map[string]struct {
					Timeout int `yaml:"timeout-minutes"`
					Steps   []struct {
						Name    string
						Timeout int `yaml:"timeout-minutes"`
					}
				}
			}
			require.NoError(t, yaml.Unmarshal([]byte(readRenderedFile(t, outputDir,
				".github/workflows/"+test.workflow)), &workflow))

			ui, exists := workflow.Jobs["ui-tests"]
			require.True(t, exists)
			timeouts := make(map[string]int)
			for _, step := range ui.Steps {
				timeouts[step.Name] = step.Timeout
			}
			// Dashboard's Playwright suite permits 40 minutes. The composite
			// action also installs npm dependencies and Chromium before testing.
			assert.GreaterOrEqual(t, timeouts["Execute UI tests"], 40+5)
			require.Positive(t, timeouts["Set up test environment"])
			require.Positive(t, timeouts["Generate test reports and collect logs"])
			// The job must also leave room for environment setup, reporting,
			// and the token/checkout steps outside those bounded actions.
			assert.GreaterOrEqual(t, ui.Timeout,
				timeouts["Set up test environment"]+
					timeouts["Execute UI tests"]+
					timeouts["Generate test reports and collect logs"]+2)
			assert.LessOrEqual(t, ui.Timeout, 60, "retain the UI runner cost cap")

			api, exists := workflow.Jobs["api-tests"]
			require.True(t, exists)
			assert.Equal(t, test.apiTimeout, api.Timeout, "preserve API job limits")
			foundAPI := false
			for _, step := range api.Steps {
				if step.Name == "Run API tests" {
					foundAPI = true
					assert.Equal(t, test.apiTimeout, step.Timeout, "preserve API execution limits")
				}
			}
			require.True(t, foundAPI)
		})
	}
}
