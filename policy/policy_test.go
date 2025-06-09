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
	assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, repo0.Branchvals.Features, "testing merging of branchvals")
	assert.EqualValues(t, "repo0.conf", repo0.Branchvals.ConfigFile, "testing repo-level inheritance of branchvals")

	err = repo0.SetBranch("dev")
	if err != nil {
		t.Fatalf("Could not set dev branch: %v", err)
	}
	assert.EqualValues(t, "stillright", repo0.Branchvals.Buildenv, "testing overrides for dev")
	assert.ElementsMatch(t, []string{"a", "b", "e", "f"}, repo0.Branchvals.Features, "testing merging")

	repo1, err := pol.GetRepoPolicy("repo1")
	if err != nil {
		t.Fatalf("Could not get repo1: %v", err)
	}
	r1b := repo1.GetAllBranches()
	assert.EqualValues(t, []string{"main"}, r1b, "testing branches")
	err = repo1.SetBranch("main")
	if err != nil {
		t.Fatalf("Could not set main branch: %v", err)
	}
	assert.EqualValues(t, []string{"flagstd1", "flagstd2"}, repo1.Branchvals.Builds["std"].Flags, "testing explicit merge")
	assert.EqualValues(t, "repo1-std2", repo1.Branchvals.Builds["std2"].BuildPackageName, "testing implicit merges at branch")
	assert.EqualValues(t, []string{"flag2"}, repo1.Branchvals.Builds["std2"].Flags, "testing implicit merge from repo")
	assert.EqualValues(t, build{Flags: []string{"flagstd1", "flagstd2"},
		BuildPackageName: "repo1-pkg",
		DHRepo:           "repo1-doc-right",
		Archs: []struct {
			Docker string
			Deb    string
			Go     string
		}{
			{"doc1", "deb1", "go1"},
			{"doc2", "deb2", "go2"}},
	}, *repo1.Branchvals.Builds["std"], "testing full merge")
	build := repo1.Branchvals.Builds["std"]
	assert.EqualValues(t, []string{"repo1-doc-right"}, build.GetImages("DHRepo"), "testing getImages()")
	assert.EqualValues(t, []string{"doc1", "doc2"}, build.GetDockerPlatforms(), "testing getDockerPlatforms()")
}
