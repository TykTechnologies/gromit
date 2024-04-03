package policy

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
)

var testRepo = map[string]string{
	"url":         "https://github.com/tyklabs/git-tests",
	"owner":       "tyklabs",
	"name":        "git-tests",
	"branch":      "main",
	"newbranch":   "testbranch",
	"filepath":    "testfile.txt",
	"commitmsg":   "Adding test file.",
	"filecontent": "Testing git functions",
}

// Fetch github.com/tyklabs/git-tests in tmpDir and,
// create new branch testbranch
// create file, commit and push
// fetch new branch in tmpVDir
// compare tmpDir and tmpVDir
// mock create a PR if GITHUB_TOKEN is set
func TestGitFunctions(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Requires GITHUB_TOKEN be set to a valid gihub PAT to run this test.")
	}
	tmpDir, err := os.MkdirTemp("", testRepo["name"])
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	src, err := InitGit(testRepo["url"], testRepo["branch"], tmpDir, token)
	if err != nil {
		t.Fatalf("init %s/%s at %s: (%v)", testRepo["owner"], testRepo["name"], tmpDir, err)
	}

	// pristine checksums
	startCsums, err := GetDirChecksums(tmpDir)
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", tmpDir, err)
	}

	// Create a new branch, switch and do the test commit there.
	head, err := src.repo.Head()
	if err != nil {
		t.Fatalf("Can not get head ref: %v", err)
	}
	nbrefName := plumbing.ReferenceName("refs/heads/testbranch")
	nbRef := plumbing.NewHashReference(nbrefName, head.Hash())
	err = src.repo.Storer.SetReference(nbRef)
	if err != nil {
		t.Fatalf("Can't set reference: %v", err)
	}
	err = src.worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(nbrefName),
		Force:  true,
	})

	// Create a test file
	tFile, err := src.CreateFile(testRepo["filepath"])
	if err != nil {
		t.Fatalf("Error creating file %s: %v", testRepo["filepath"], err)
	}
	tFile.Write([]byte(testRepo["filecontent"]))
	tFile.Close()

	// Add the checksum of the newly created file to our inital checksumn list.
	tfh, err := os.Open(filepath.Join(tmpDir, testRepo["filepath"]))
	if err != nil {
		t.Fatalf("can't open test file: %s, %v", testRepo["filepath"], err)
	}
	sh := sha1.New()
	if _, err = io.Copy(sh, tfh); err != nil {
		t.Fatalf("Can't calculate sha1 for new file: %v", err)
	}
	tfh.Close()
	startCsums[testRepo["filepath"]] = hex.EncodeToString(sh.Sum(nil))

	// git add
	h, err := src.AddFile(testRepo["filepath"])
	if err != nil {
		t.Fatalf("Unable to add  file %s: %v", testRepo["filepath"], err)
	}
	t.Logf("worktree hash: %s", h.String())
	err = src.Commit(testRepo["commitmsg"])
	if err != nil {
		t.Fatalf("Unable to commit  file %s: %v", testRepo["filepath"], err)
	}

	// new checksums
	committedCsums, err := GetDirChecksums(tmpDir)
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", testRepo["dir"], err)
	}

	err = src.Push(testRepo["newbranch"])
	if err != nil {
		t.Fatalf("error in pushig to remote: %v", err)
	}
	t.Logf("Pushed our test chenges to remote")

	t.Logf("Now verifying by pulling the changes..")
	tmpVDir, err := os.MkdirTemp("", testRepo["name"])
	if err != nil {
		t.Fatalf("Error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpVDir)
	vSrc, err := InitGit(testRepo["url"], "main", tmpVDir, token)
	if err != nil {
		t.Fatalf("init %s/%s at %s: (%v)", testRepo["owner"], testRepo["name"], tmpVDir, err)
	}
	err = vSrc.FetchBranch(testRepo["newbranch"])
	if err != nil {
		t.Fatalf("Error checking out branch %s: %v", testRepo["newbranch"], err)
	}
	head, err = vSrc.repo.Head()
	if err != nil {
		t.Fatalf("Can not get head ref: %v", err)
	}

	// prevHead, _ := vRepo.repo.ResolveRevision(plumbing.Revision("HEAD~1"))
	// hCommit, _ := vRepo.repo.CommitObject(head.Hash())
	// prevCommit, _ := vRepo.repo.CommitObject(*prevHead)
	// patch, _ := prevCommit.Patch(hCommit)
	// _ = patch.Encode(os.Stdout)

	pulledCsums, err := GetDirChecksums(tmpVDir)
	if err != nil {
		t.Fatalf("Can't get checksums for dir: %s. %v", tmpVDir, err)
	}

	t.Log("Csum start: ", startCsums)
	t.Log("Csum post commit:  ", committedCsums)
	t.Log("Csum after pulling the changes ", pulledCsums)

	juser := os.Getenv("JIRA_USER")
	jtoken := os.Getenv("JIRA_TOKEN")
	if jtoken == "" || juser == "" {
		t.Skip("Requires JIRA_USER and JIRA_TOKEN be set to run this test.")
	}
	j := NewJiraClient(juser, jtoken)
	i, err := j.GetIssue("SYSE-1")
	var pol Policies
	config.LoadConfig("../testdata/config-test.yaml")
	err = LoadRepoPolicies(&pol)
	if err != nil {
		t.Fatalf("Could not load policy from testdata/config-test.yaml: %v", err)
	}
	r0, err := pol.GetRepoPolicy("repo0")
	if err != nil {
		t.Fatalf("Could not get repo0: %v", err)
	}
	gh := NewGithubClient(token)
	prOpts := &PullRequest{
		Jira:       i,
		BaseBranch: testRepo["branch"],
		PrBranch:   testRepo["newbranch"],
		Owner:      testRepo["owner"],
		Repo:       testRepo["name"],
	}
	_, err = gh.CreatePR(r0, prOpts)
	if err != nil {
		t.Fatalf("mock PR: %v", err)
	}

	err = vSrc.DeleteRemoteBranch(testRepo["newbranch"])
	if err != nil {
		t.Fatalf("error in deleting  remote branch: %v", err)
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
