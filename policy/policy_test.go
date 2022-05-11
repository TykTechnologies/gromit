package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
)

// TestPolicyConfig can test a repo with back/forward ports
func TestPolicy(t *testing.T) {
	//timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"

	var rp Policies
	config.LoadConfig("../testdata/policies/test.yml")
	err := LoadRepoPolicies(&rp)
	if err != nil {
		t.Fatalf("Could not load policy: %v", err)
	}
	repo, err := rp.GetRepo("git-tests", "https://github.com/tyklabs", "main")
	if err != nil {
		t.Fatalf("Could not get a repo: %v", err)
	}
	err = repo.InitGit(1, 0, "/tmp/gt", "")
	if err != nil {
		t.Fatalf("Could not init: %v", err)
	}

	// Test config merging
	t.Run("config", func(t *testing.T) {
		files := map[string][]string{
			"releng":     {"ci/*"},
			"sync":       {".github/workflows/sync-automation.yml"},
			"dependabot": {".github/dependabot.yml"},
			"config":     {"tyk.conf.example"},
		}
		assert.EqualValues(t, repo.Protected, []string{"master", "release-3-lts"})
		for key := range files {
			assert.ElementsMatch(t, repo.Files[key], files[key])
		}
	})
	// Test template generation
	t.Run("gentemplate", func(t *testing.T) {
		repo.GenTemplate("sync-automation", "first commit from test")
	})
}
