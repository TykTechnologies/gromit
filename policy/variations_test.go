package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariations(t *testing.T) {
	tv, err := loadVariations(testConfig)
	if err != nil {
		t.Logf("Failed to load saved test variation state from %s: %v", testConfig, err)
		t.Fail()
	}
	assert.ElementsMatch(t, []string{"br0", "br1"}, tv["repo0"].Branches(), "branches for repo0")
	assert.ElementsMatch(t, []string{"tr0"}, tv["repo0"].Triggers("br0"), "triggers for repo0-br0")
	assert.ElementsMatch(t, []string{"ts0", "ts1"}, tv["repo0"].Testsuites("br0", "tr0"), "testsuites for repo0-br0-tr0")

	assert.ElementsMatch(t, []string{"br0"}, tv["repo1"].Branches(), "branches for repo1")
	assert.ElementsMatch(t, []string{"tr0", "tr1"}, tv["repo1"].Triggers("br0"), "triggers for repo1-br0")
	assert.ElementsMatch(t, []string{"ts0", "ts1"}, tv["repo1"].Testsuites("br0", "tr1"), "testsuites for repo1-br0-tr1")
}
