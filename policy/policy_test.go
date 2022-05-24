package policy

import (
	"bytes"
	"os"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// TestPolicyConfig can test a repo with back/forward ports
func TestPolicy(t *testing.T) {
	//timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"

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
	defer func() {
		t.Log("Deleting temporary files..")
		os.RemoveAll(testDir)
	}()
	err = repo.InitGit(1, 0, testDir, "")
	if err != nil {
		t.Fatalf("Could not init: %v", err)
	}

	// Test config merging
	t.Run("config", func(t *testing.T) {
		files := map[string][]string{
			"releng":     {"ci/**", ".github/workflows/release.yml"},
			"sync":       {".github/workflows/sync-automation.yml"},
			"dependabot": {".github/dependabot.yml"},
			"config":     {"tyk.conf.example"},
			"tests":      {".github/workflows/api-tests.yml", ".github/workflows/ui-tests.yml"},
		}
		assert.EqualValues(t, repo.Protected, []string{"master", "release-3-lts"})
		for key := range files {
			assert.ElementsMatch(t, repo.Files[key], files[key])
		}
	})
	// Test template generation
	t.Run("gentemplate", func(t *testing.T) {
		//pwd := os.Getenv("PWD")
		f, err := repo.GenTemplate("sync")
		if err != nil {
			t.Fatalf("Error generating template:  sync-automation: %v", err)
		}
		t.Log("Files generated: ", f)
		hash, err := repo.Commit("First commit from test", false)
		if err != nil {
			t.Fatalf("Error commiting after gentemplate:  sync-automation: %v", err)
		}
		t.Logf("Commit made successfully: %s", hash)
		// Check if the sync-automation file is parsed correctly.
		// NOTE: in the test file, the files tracked by sync-automation should always
		// be given in alphabetical order. - to avoid order related failure of test.
		testFile, err := os.ReadFile("../testdata/sync-automation/sync-automation.yml")
		if err != nil {
			t.Fatalf("Error reading sync-automation file from testdata: %v", err)
		}
		// Sync bundle generates only one file as of now.
		genFile, err := repo.gitRepo.ReadFile(f[0])
		if err != nil {
			t.Fatalf("Error reading generated sync-automation file from git: %v", err)
		}
		t.Logf("Comparing generated file with the test file..")
		diff := cmp.Diff(testFile, genFile)
		if diff != "" {
			t.Logf("Diff before stripping timestamp: \n%s", diff)
		}
		// Hack to make the tests pass. Strip bytes containing the timestamp line.
		// This should be changed if the test file changes, or the template changes.
		tsStart := 32
		tsEnd := 76
		tfTs := testFile[tsStart:tsEnd]
		gfTs := genFile[tsStart:tsEnd]
		// t.Logf("\n\n\n%s", bytes.Replace(testFile, tfTs, []byte(""), 1))
		// t.Logf("\n\n\n%s", bytes.Replace(genFile, gfTs, []byte(""), 1))
		diff = cmp.Diff(bytes.Replace(testFile, tfTs, []byte(""), 1), bytes.Replace(genFile, gfTs, []byte(""), 1))
		if diff != "" {
			t.Fatalf("Diff after stripping timestamp: \n%s", diff)
		}
		// assert.True(t, bytes.Equal(testFile, genFile), "Comparing generated file, and test file(sync-automation)")
	})
}
