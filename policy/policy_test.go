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

// TestPolicyConfig can test a repo with back/forward ports
func TestPolicy(t *testing.T) {

	var rp Policies

	//timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"
	testTimeStr := "Tue May 24 08:30:46 UTC 2022"
	timeStamp, err := time.Parse(time.UnixDate, testTimeStr)
	if err != nil {
		t.Fatalf("Can't parse the test timestamp: %v", err)
	}

	// Need to have github token for PR test
	ghToken := os.Getenv("GH_TOKEN")

	config.LoadConfig("../testdata/policies/repos.yaml")
	err = LoadRepoPolicies(&rp)
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
	err = repo.InitGit(1, 0, testDir, ghToken)
	if err != nil {
		t.Fatalf("Could not init: %v", err)
	}
	newBranch := "pr-test"
	err = repo.gitRepo.SwitchBranch(newBranch)
	if err != nil {
		t.Fatalf("Error checking out a new branch: %s : %v", newBranch, err)
	}

	// Test config parsing
	t.Run("config", func(t *testing.T) {
		assert.EqualValues(t, repo.Protected, []string{"master", "release-3-lts"})
	})
	// Test template generation
	t.Run("gentemplate", func(t *testing.T) {
		// to set current timestamp - uncomment below line
		// timeStamp = time.Time{}
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
			t.Logf("Diff before stripping timestamp: \n%s", diff)
		}
		assert.True(t, bytes.Equal(testFile, genFile), "Comparing generated file, and test file(sync-automation)")
	})
	// Test push to GH and creating PR.
	t.Run("createpr", func(t *testing.T) {
		bundle := "sync"
		title := "Testing sync-automation"
		base := "master"
		// Test dry run first.
		_, err := repo.CreatePR(bundle, title, base, true)
		if err != nil {
			t.Fatalf("Error running CreatePR in dryrun mode: (bundle-%s): %v", bundle, err)
		}
		// Push the current changes and create a PR.
		url, err := repo.CreatePR(bundle, title, base, false)
		if err != nil {
			t.Fatalf("PR actual run failed: %v", err)
		}
		t.Logf("PR URL: %s", url)

	})
}
