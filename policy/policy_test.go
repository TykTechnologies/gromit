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

	// master doesn't have any explicit version set, so should be
	// release-5.x (the common branch value)
	assert.EqualValues(t, "release-5.x", repo.Branchvals.RelengVersion)

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-4")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	// release-4 has explicit value release-4.x set.
	assert.EqualValues(t, "release-4.x", repo.Branchvals.RelengVersion)

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-4.3")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	// release-4.3 has sourcebranch set to release-4, and no explicit
	// relengversion set, so should inherit the value of release-4
	assert.EqualValues(t, "release-4.x", repo.Branchvals.RelengVersion)

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-4.3.1")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	// release-4.3.1 has sourcebranch set to release-4.3, and no explicit
	// releng version set, so we should recursively go to the root source
	// branch and inherit its releng version value.
	assert.EqualValues(t, "release-4.x", repo.Branchvals.RelengVersion)

	repo, err = rp.GetRepo("tyk", "https://github.com/tyklabs", "release-3.0.12")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	// release-3.0.12 has release-3.0.11 as source branch, having no explicitly
	// set relengversion, and hence the relengversion should be that of release-3.0.11
	assert.EqualValues(t, "release-3.x", repo.Branchvals.RelengVersion)


}
