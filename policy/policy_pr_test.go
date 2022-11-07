//go:build githubtests
// +build githubtests

package policy

import (
	"os"
	"testing"
)

func TestCreatePR(t *testing.T) {
	// Need to have github token for PR test
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken == "" {
		t.Skip("set GH_TOKEN to a valid gitgub PAT to  run this test.")
	}
	repo := SetupRepo(t, ghToken)
	// Create ans switch to a new branch for the PR.
	newBranch := "pr-test"
	err := repo.gitRepo.SwitchBranch(newBranch)
	if err != nil {
		t.Fatalf("Error checking out a new branch: %s : %v", newBranch, err)
	}
	// Create a test file and commit it for PR.
	testFileName := "pr.test"
	testFile, err := repo.gitRepo.CreateFile(testFileName)
	if err != nil {
		t.Fatalf("Error creating test file: %s : %v", testFileName, err)
	}
	testFile.Write([]byte("test content"))
	testFile.Close()

	_, err = repo.gitRepo.AddFile(testFileName)
	if err != nil {
		t.Fatalf("Error when adding file to worktree: %v", err)
	}
	// Create a commit
	hash, err := repo.Commit("Commit from CreatePR test.", false)
	if err != nil {
		t.Fatalf("Error commiting testfile in createPR: %v", err)
	}
	t.Logf("Commit made successfully: %s", hash)

	bundle := "sync"
	title := "Testing sync-automation"
	base := "master"
	// Test dry run first.
	_, err = repo.CreatePR(bundle, title, base, true, false)
	if err != nil {
		t.Fatalf("Error running CreatePR in dryrun mode: (bundle-%s): %v", bundle, err)
	}
	// Push the current changes and create a PR.
	url, err := repo.CreatePR(bundle, title, base, false, false)
	if err != nil {
		t.Fatalf("PR actual run failed: %v", err)
	}
	t.Logf("PR URL: %s", url)

	// Delete the remote branch(which also closes the PR)
	err = repo.gitRepo.DeleteRemoteBranch(newBranch)
	if err != nil {
		t.Fatalf("Unable to delete test banch on remote: %v", err)
	}
}

func TestCreatePRAutomerge(t *testing.T) {
	// Need to have github token for PR test
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken == "" {
		t.Skip("set GH_TOKEN to a valid gitgub PAT to  run this test.")
	}
	repo := SetupRepo(t, ghToken)
	// Create ans switch to a new branch for the PR.
	newBranch := "pr-test-automerge"
	err := repo.gitRepo.SwitchBranch(newBranch)
	if err != nil {
		t.Fatalf("Error checking out a new branch: %s : %v", newBranch, err)
	}
	// Create a test file and commit it for PR.
	testFileName := "pr.test"
	testFile, err := repo.gitRepo.CreateFile(testFileName)
	if err != nil {
		t.Fatalf("Error creating test file: %s : %v", testFileName, err)
	}
	testFile.Write([]byte("test content"))
	testFile.Close()

	_, err = repo.gitRepo.AddFile(testFileName)
	if err != nil {
		t.Fatalf("Error when adding file to worktree: %v", err)
	}
	// Create a commit
	hash, err := repo.Commit("Commit from CreatePR test.", false)
	if err != nil {
		t.Fatalf("Error commiting testfile in createPR: %v", err)
	}
	t.Logf("Commit made successfully: %s", hash)

	bundle := "sync"
	title := "Testing sync-automation"
	base := "master"
	// Test dry run first.
	_, err = repo.CreatePR(bundle, title, base, true, true)
	if err != nil {
		t.Fatalf("Error running CreatePR in dryrun mode: (bundle-%s): %v", bundle, err)
	}
	// Push the current changes and create a PR with automerge enabled.
	url, err := repo.CreatePR(bundle, title, base, false, true)
	if err != nil {
		t.Fatalf("PR actual run failed: %v", err)
	}
	t.Logf("PR URL: %s", url)

	// Delete the remote branch(which also closes the PR)
	err = repo.gitRepo.DeleteRemoteBranch(newBranch)
	if err != nil {
		t.Fatalf("Unable to delete test banch on remote: %v", err)
	}
}
