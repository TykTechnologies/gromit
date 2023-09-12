package policy

import (
	"encoding/json"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
)

func TestPolicyConfig(t *testing.T) {
	var pol Policies
	config.LoadConfig("../testdata/config-test.yaml")
	//config.LoadConfig("")
	err := LoadRepoPolicies(&pol)
	if err != nil {
		t.Fatalf("Could not load policy from testdata/config-test.yaml: %v", err)
	}
	prettyPol, _ := json.MarshalIndent(pol, "", " ")
	t.Logf("%s", prettyPol)
	main, err := pol.GetRepoPolicy("repo0", "main")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	assert.EqualValues(t, "right", main.Branchvals.Buildenv, "testing overrides zero")
	assert.EqualValues(t, []string{"a", "b", "c", "d"}, main.Branchvals.Features, "testing merging zero")
	assert.EqualValues(t, "Repo Zero", main.Description, "testing inherited values zero")

	dev, err := pol.GetRepoPolicy("repo0", "dev")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	assert.EqualValues(t, "stillright", dev.Branchvals.Buildenv, "testing overrides one")
	assert.EqualValues(t, []string{"a", "b", "e", "f"}, dev.Branchvals.Features, "testing merging one")
	assert.EqualValues(t, "Repo Zero", dev.Description, "testing inherited values one")
}
