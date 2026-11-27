package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestUpgradeTestsRunOutsidePullRequests pins the upgrade-tests job condition
// in every rendered release workflow. It reads the golden snapshot, which
// TestGoldenRender keeps identical to what the templates render.
//
// dep-guard only runs on pull requests, so on push, tag and schedule events it
// is skipped. A job without an explicit status check gets an implicit
// success(), which is false when any ancestor was skipped, so upgrade-tests
// was silently skipped outside pull requests (TT-19375). The job must guard
// itself with !cancelled() and depend on its direct need succeeding instead.
func TestUpgradeTestsRunOutsidePullRequests(t *testing.T) {
	type job struct {
		If    string    `yaml:"if"`
		Needs yaml.Node `yaml:"needs"`
	}
	type workflow struct {
		Jobs map[string]job `yaml:"jobs"`
	}

	files, err := filepath.Glob(filepath.Join("testdata", "golden", "*", "*", ".github", "workflows", "release.yml"))
	require.NoError(t, err)

	checked := 0
	for _, f := range files {
		rel, err := filepath.Rel(filepath.Join("testdata", "golden"), f)
		require.NoError(t, err)
		repoName := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(rel))))
		branch := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(rel))))

		t.Run(repoName+"/"+branch, func(t *testing.T) {
			data, err := os.ReadFile(f)
			require.NoError(t, err)

			var wf workflow
			require.NoError(t, yaml.Unmarshal(data, &wf))
			ut, ok := wf.Jobs["upgrade-tests"]
			if !ok {
				t.Skip("no upgrade-tests job")
			}
			checked++

			var needs []string
			require.NoError(t, ut.Needs.Decode(&needs))
			assert.Equal(t, []string{"test-controller-distros"}, needs)
			assert.Contains(t, ut.If, "!cancelled()")
			assert.Contains(t, ut.If, "needs.test-controller-distros.result == 'success'")
			if repoName == "tyk-analytics" {
				assert.Contains(t, ut.If, "github.event_name != 'pull_request'")
			} else {
				assert.NotContains(t, ut.If, "pull_request")
			}
		})
	}
	require.NotZero(t, checked, "no golden release workflow had an upgrade-tests job")
}
