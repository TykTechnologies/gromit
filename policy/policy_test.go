package policy

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/TykTechnologies/gromit/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestGenTemplate(t *testing.T) {
	repo := SetupRepo(t, "")
	// to set current timestamp - uncomment below line
	// timeStamp = time.Time{}
	//timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"
	testTimeStr := "Tue May 24 08:30:46 UTC 2022"
	timeStamp, err := time.Parse(time.UnixDate, testTimeStr)
	if err != nil {
		t.Fatalf("Can't parse the test timestamp: %v", err)
	}
	// set test timestamp
	repo.SetTimestamp(timeStamp)
	// repo.SetTimestamp(timeStamp)
	err = repo.GenTemplate("sync")
	if err != nil {
		t.Fatalf("Error generating template:  sync-automation: %v", err)
	}
	hash, err := repo.Commit("First commit from test", false)
	if err != nil {
		// Need GH token for
		t.Fatalf("Error commiting after gentemplate:  sync-automation: %v", err)
	}
	t.Logf("Commit made successfully: %s", hash)
	testFile, err := os.ReadFile("../testdata/sync-automation/sync-automation.yml")
	if err != nil {
		t.Fatalf("Error reading sync-automation file from testdata: %v", err)
	}
	// FIXME: Sync bundle generates only one file as of now.
	genFile, err := repo.gitRepo.ReadFile(".github/workflows/sync-automation.yml")
	if err != nil {
		t.Fatalf("Error reading generated sync-automation file from git: %v", err)
	}
	t.Logf("Comparing generated file with the test file..")
	diff := cmp.Diff(testFile, genFile)
	if diff != "" {
		t.Logf("Diff between testfile and generated file: \n%s", diff)
	}
	assert.True(t, bytes.Equal(testFile, genFile), "Comparing generated file, and test file(sync-automation)")
}

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
	// test if global branch policy is set correctly.
	assert.EqualValues(t, repo.Branchvals.GoVersion, "1.15")
}

func SetupRepo(t *testing.T, token string) *RepoPolicy {
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
