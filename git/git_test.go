package git

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

var testRepo = map[string]string{"fqdn": "https://github.com/tyklabs/git-tests.git",
	"repo":        "tyk",
	"dir":         "/tmp/gt",
	"vdir":        "/tmp/gt-v",
	"branch":      "main",
	"newbranch":   "testbranch",
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

	// Get the checksums of all files after fresh clone, before making any changes.
	startCsums, err := GetDirChecksums(testRepo["dir"])
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", testRepo["dir"], err)
	}

	// Create a new branch, switch and do the test commit there.
	head, err := repo.repo.Head()
	if err != nil {
		t.Fatalf("Can not get head ref: %v", err)
	}
	nbrefName := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", testRepo["newbranch"]))
	nbRef := plumbing.NewHashReference(nbrefName, head.Hash())
	err = repo.repo.Storer.SetReference(nbRef)
	if err != nil {
		t.Fatalf("Can't set reference: %v", err)
	}
	err = repo.worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(nbrefName),
		Force:  true,
	})

	// Create a test file, and add and commit it.
	tFile, err := repo.CreateFile(testRepo["filepath"])
	if err != nil {
		t.Fatalf("Error creating file %s: %v", testRepo["filepath"], err)
	}
	tFile.Write([]byte(testRepo["filecontent"]))
	tFile.Close()

	// Add the checksum of the newly created file to our inital checksumn list.
	path := testRepo["dir"] + "/" + testRepo["filepath"]
	tfh, err := os.Open(path)
	if err != nil {
		t.Fatalf("can't open test file: %s, %v", testRepo["filepath"], err)
	}
	sh := sha1.New()
	if _, err = io.Copy(sh, tfh); err != nil {
		t.Fatalf("Can't calculate sha1 for new file: %v", err)
	}
	tfh.Close()
	startCsums[testRepo["filepath"]] = hex.EncodeToString(sh.Sum(nil))

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

	committedCsums, err := GetDirChecksums(testRepo["dir"])
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", testRepo["dir"], err)
	}

	err = repo.Push(testRepo["newbranch"], testRepo["newbranch"])
	if err != nil {
		t.Fatalf("error in pushig to remote: %v", err)
	}
	t.Logf("Pushed our test chenges to remote")

	t.Logf("Now verifying by pulling the changes..")
	vRepo, err := FetchRepo(testRepo["fqdn"], testRepo["vdir"], token, fetchDepth)
	if err != nil {
		t.Fatalf("Could not fetch repo: %s, with fqdn: %s, with depth: %d to dir %s: (%v)", testRepo["repo"], testRepo["fqdn"], fetchDepth, testRepo["vdir"], err)
	}
	err = vRepo.Checkout(testRepo["newbranch"])
	if err != nil {
		t.Fatalf("Error checking out branch %s: %v", testRepo["newbranch"], err)
	}
	head, err = vRepo.repo.Head()
	if err != nil {
		t.Fatalf("Can not get head ref: %v", err)
	}

	// prevHead, _ := vRepo.repo.ResolveRevision(plumbing.Revision("HEAD~1"))
	// hCommit, _ := vRepo.repo.CommitObject(head.Hash())
	// prevCommit, _ := vRepo.repo.CommitObject(*prevHead)
	// patch, _ := prevCommit.Patch(hCommit)
	// _ = patch.Encode(os.Stdout)

	pulledCsums, err := GetDirChecksums(testRepo["vdir"])
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", testRepo["vdir"], err)
	}

	t.Log("Csum start: ", startCsums)
	t.Log("Csum post commit:  ", committedCsums)
	t.Log("Csum after pulling the changes ", pulledCsums)

	err = vRepo.DeleteRemoteBranch(testRepo["newbranch"])
	if err != nil {
		t.Fatalf("error in deleting  remote branch: %v", err)
	}
	t.Logf("Deleting test directories..")
	if testRepo["dir"] != "" {
		os.RemoveAll(testRepo["dir"])
	}
	if testRepo["vdir"] != "" {
		os.RemoveAll(testRepo["vdir"])
	}
	assert.EqualValues(t, startCsums, committedCsums)
	assert.EqualValues(t, startCsums, pulledCsums)
}

func GetDirChecksums(dir string) (map[string]string, error) {

	csumList := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .git dir tree
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() && info.Mode().IsRegular() {
			fh, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fh.Close()
			h := sha1.New()
			_, err = io.Copy(h, fh)
			if err != nil {
				return err
			}
			csumList[info.Name()] = hex.EncodeToString(h.Sum(nil))
		}
		return nil
	})
	return csumList, err
}
