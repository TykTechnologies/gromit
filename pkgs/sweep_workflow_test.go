package pkgs

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// GitHub drops a job output that contains any secret's value, and the
// image list holds the CI Docker Hub account name, so the sweep's list job
// may only output numbers. A dropped list leaves the scan matrix empty and
// nothing is scanned.
func TestSweepListJobOutputsOnlyNumbers(t *testing.T) {
	raw, err := os.ReadFile("../.github/workflows/dockerhub-secret-sweep.yml")
	require.NoError(t, err)

	var wf struct {
		Jobs map[string]struct {
			Outputs map[string]string `yaml:"outputs"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &wf))

	list, ok := wf.Jobs["list"]
	require.True(t, ok, "list job missing")
	var keys []string
	for k := range list.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{"count", "failed", "indices", "total"}, keys)
}
