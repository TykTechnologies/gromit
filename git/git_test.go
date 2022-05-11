package git

import (
	"os"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

var testRepo = map[string]string{"fqdn": "https://github.com/asutosh/git-tests.git",
	"repo":        "tyk",
	"dir":         "",
	"branch":      "main",
	"filepath":    "testfile.txt",
	"commitmsg":   "Adding test file.",
	"filecontent": "Testing git functions"}

//"token": ""}
var fetchDepth int = 1

func TestGitFunctions(t *testing.T) {
	token := os.Getenv("GH_TOKEN")
	repo, err := FetchRepo(testRepo["fqdn"], testRepo["dir"], token, fetchDepth)
	if err != nil {
		t.Fatalf("Could not fetch repo: %s, with fqdn: %s, with depth: %d to dir %s: (%v)", testRepo["repo"], testRepo["fqdn"], fetchDepth, testRepo["dir"], err)
	}
	err = repo.Checkout(testRepo["branch"])
	if err != nil {
		t.Fatalf("Error checking out branch %s: %v", testRepo["branch"], err)
	}
	tFile, err := repo.CreateFile(testRepo["filepath"])
	tFile.Write([]byte(testRepo["filecontent"]))
	tFile.Close()
	fInfo, err := repo.fs.ReadDir("/")
	if err != nil {
		t.Fatalf("FS error: %v", err)
	}
	t.Logf("fs: %s", fInfo[0].Name())
	if err != nil {
		t.Fatalf("Error creating file %s: %v", testRepo["filepath"], err)
	}
	h, err := repo.worktree.Add(testRepo["filepath"])
	if err != nil {
		t.Fatalf("Error adding file %s to worktree: %v", testRepo["filepath"], err)
	}
	t.Logf("worktree hash: %s", h.String())
	hash, err := repo.AddFile(testRepo["filepath"], testRepo["commitmsg"], false)
	if err != nil {
		t.Fatalf("Unable to commit  file %s: %v", testRepo["filepath"], err)
	}
	t.Logf("Commit hash: %s", hash.String())
	head, _ := repo.repo.Head()
	prevHead, _ := repo.repo.ResolveRevision(plumbing.Revision("HEAD~1"))
	hCommit, _ := repo.repo.CommitObject(head.Hash())
	prevCommit, _ := repo.repo.CommitObject(*prevHead)
	patch, _ := prevCommit.Patch(hCommit)
	_ = patch.Encode(os.Stdout)

	//err = repo.Push(testRepo["branch"], testRepo["branch"])

}
