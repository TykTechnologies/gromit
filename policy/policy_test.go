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
	repo0, err := pol.GetRepoPolicy("repo0")
	if err != nil {
		t.Fatalf("Could not get repo0: %v", err)
	}
	err = repo0.SetBranch("main")
	if err != nil {
		t.Fatalf("Could not set main branch: %v", err)
	}
	assert.EqualValues(t, "right", repo0.Branchvals.Buildenv, "testing branch-level override for main")
	assert.EqualValues(t, []string{"a", "b", "c", "d"}, repo0.Branchvals.Features, "testing merging of branchvals")
	assert.EqualValues(t, "repo0.conf", repo0.Branchvals.ConfigFile, "testing repo-level inheritance of branchvals")
	assert.EqualValues(t, "Repo Zero", repo0.Description, "testing repo-level value")

	err = repo0.SetBranch("dev")
	if err != nil {
		t.Fatalf("Could not set dev branch: %v", err)
	}
	assert.EqualValues(t, "stillright", repo0.Branchvals.Buildenv, "testing overrides for dev")
	assert.EqualValues(t, []string{"a", "b", "e", "f"}, repo0.Branchvals.Features, "testing merging")
	assert.EqualValues(t, "Repo Zero", repo0.Description, "testing inheritance")

	repo1, err := pol.GetRepoPolicy("repo1")
	if err != nil {
		t.Fatalf("Could not get repo1: %v", err)
	}
	assert.EqualValues(t, "Repo One", repo1.Description, "testing second repo")
	r1b := repo1.GetAllBranches()
	assert.EqualValues(t, []string{"main"}, r1b, "testing branches")
}
