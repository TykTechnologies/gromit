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
	if err != nil {
		t.Fatalf("Could not load policy: %v", err)
	}
	repo, err := rp.GetRepo("tyk", "https://github.com/tyklabs", "master")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	assert.EqualValues(t, repo.Protected, []string{"master", "release-3-lts", "release-4"})
	// test if branch policy for master is set correctly.
	assert.EqualValues(t, repo.Branchvals.UpgradeFromVer, "3.0.8")
	t.Logf("Branchvals: %+v", repo.Branchvals)
	// test if  branch policy is set correctly for master
	assert.EqualValues(t, repo.Branchvals.GoVersion, "1.16")
}
