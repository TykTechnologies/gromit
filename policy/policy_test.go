package policy

import (
	"os"
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

func SetupTestRepo(t *testing.T, token string) *RepoPolicy {
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
	testDir := "/tmp/pt-" + repo.Name
	// delete the temp dir as soon as the tests finish.
	t.Cleanup(func() {
		t.Log("Deleting temporary files..")
		os.RemoveAll(testDir)
	})
	err = repo.InitGit(1, 0, testDir, token)
	if err != nil {
		t.Fatalf("Could not init: %v", err)
	}
	return &repo
}
