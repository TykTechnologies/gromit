package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
)

func TestPolicyConfig(t *testing.T) {
	var rp Policies
	config.LoadConfig("../testdata/policies/repos.yaml")
	err := LoadRepoPolicies(&rp)
	t.Logf("Branches: %+v", rp.Branches)
	t.Logf("Branches.branch: %+v", rp.Branches.Branch)
	if err != nil {
		t.Fatalf("Could not load policy: %v", err)
	}
	repo, err := rp.GetRepo("tyk", "https://github.com/tyklabs", "master")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	assert.EqualValues(t, repo.Protected, []string{"master", "release-3-lts", "release-4"})
	// test if branch policy for master is set correctly.
	assert.EqualValues(t, "3.0.8", repo.Branchvals.UpgradeFromVer)
	t.Logf("Branchvals: %+v", repo.Branchvals)
	// test if  branch policy is set correctly for master
	assert.EqualValues(t, "1.16", repo.Branchvals.GoVersion)

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-4")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-4.3")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}

}
