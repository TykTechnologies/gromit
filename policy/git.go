package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"time"

	_ "embed"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/rs/zerolog/log"
)

// GitRepo models a local git worktree with the authentication and
// enough metadata to allow it to be pushed it to github
type GitRepo struct {
	url        string
	commitOpts *git.CommitOptions
	repo       *git.Repository
	worktree   *git.Worktree
	dir        string
	auth       transport.AuthMethod
}

// InitGit is a constructor for the GitRepo type
// private repos will need ghToken
func InitGit(url, branch, dir, ghToken string) (*GitRepo, error) {
	log.Logger = log.With().Str("url", url).Logger()

	fi, err := os.Stat(dir)
	if os.IsNotExist(err) || !fi.IsDir() {
		log.Debug().Str("dir", dir).Msg("does not exist")
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}

	cloneOpts := &git.CloneOptions{
		URL:           url,
		Tags:          git.NoTags,
		Progress:      os.Stdout,
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if ghToken != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "ignored", // anything except an empty string
			Password: ghToken,
		}
	}

	var repo *git.Repository
	repo, err = git.PlainOpen(dir)
	if err == git.ErrRepositoryNotExists {
		repo, err = git.PlainClone(dir, false, cloneOpts)
		if err != nil {
			return nil, fmt.Errorf("could not clone %s: %v", branch, err)
		}
		log.Info().Msgf("created fresh clone in %s", dir)
	}
	w, err := repo.Worktree()
	if err != nil {
		log.Error().Err(err).Msg("Error getting worktree")
		return nil, err
	}

	return &GitRepo{
		url:      url,
		auth:     cloneOpts.Auth,
		repo:     repo,
		worktree: w,
		dir:      dir,
		commitOpts: &git.CommitOptions{
			All: false,
			Author: &object.Signature{
				Name:  "Gromit",
				Email: "policy@gromit",
				When:  time.Now().UTC(),
			},
		},
	}, err
}

// RemoveAll removes all files matching the supplied path from the  worktree.
func (r *GitRepo) RemoveAll(path string) error {
	return r.worktree.RemoveGlob(path)
}

// AddFile adds a file in the worktree to the index.
// The file is assumed to have been updated prior to calling this function.
func (r *GitRepo) AddFile(path string) (plumbing.Hash, error) {
	hash, err := r.worktree.Add(path)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return hash, nil
}

// Branch returns the short name of the ref HEAD is pointing
// to - provided the ref is a branch. Returns empty string
// if ref is not a branch.
func (r *GitRepo) Branch() string {
	h, err := r.repo.Head()
	if err != nil {
		log.Warn().Err(err).Msg("could not get current branch")
		return ""
	}
	if !h.Name().IsBranch() {
		log.Warn().Msg("HEAD is not a branch")
		return ""
	}
	return h.Name().Short()
}

// Commit adds all unstaged changes and commits the current worktree, confirming if asked
// Note that this commit will be lost if it is not pushed to a remote.
func (r *GitRepo) Commit(msg string) error {
	newCommitHash, err := r.worktree.Commit(msg, r.commitOpts)
	if err != nil {
		return err
	}
	log.Trace().Str("hash", newCommitHash.String()).Msg("worktree hash")
	if err != nil {
		return err
	}
	newCommit, err := r.repo.CommitObject(newCommitHash)
	if err != nil {
		return fmt.Errorf("getting new commit: %w", err)
	}
	log.Trace().Str("hash", newCommit.String()).Msg("new commit")
	return nil
}

// (r *GitRepo) FetchBranch fetches the given ref and then checks it out to the worktree
// Any local changes are lost. If the branch does not exist in the `origin` remote, an
// error is returned
func (r *GitRepo) FetchBranch(branch string) error {
	rbSpec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/remotes/origin/%s", branch, branch))
	err := r.repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{rbSpec},
		Depth:      1,
		Auth:       r.auth,
		Progress:   os.Stdout,
		Tags:       git.NoTags,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("could not fetch %s: %v", branch, err)
	}
	rbRef := plumbing.NewRemoteReferenceName("origin", branch)
	lbRef := plumbing.NewBranchReferenceName(branch)
	// err = r.repo.CreateBranch(&config.Branch{
	// 	Name:   branch,
	// 	Remote: "origin",
	// 	Merge:  lbRef,
	// })
	// if err != nil && err != git.ErrBranchExists {
	// 	return fmt.Errorf("could not create local branch %s: %v", branch, err)
	// }

	err = r.repo.Storer.SetReference(plumbing.NewSymbolicReference(lbRef, rbRef))
	if err != nil {
		return fmt.Errorf("could not set storer ref: %v", err)
	}
	return r.worktree.Checkout(&git.CheckoutOptions{
		Branch: lbRef,
		Create: false,
		Force:  true,
	})
}

// (r *GitRepo) PullBranch will incorporate changes from origin.
// Only ff changes can be merged.
func (r *GitRepo) PullBranch(branch string) error {
	rbRef := plumbing.NewBranchReferenceName(branch)
	return r.worktree.Pull(&git.PullOptions{
		SingleBranch:  true,
		ReferenceName: rbRef,
		Auth:          r.auth,
		Progress:      os.Stdout,
	})
}

// Push will push the current worktree to origin
func (r *GitRepo) Push(remoteBranch string) error {
	if remoteBranch == r.Branch() {
		log.Warn().Msgf("pushing to %s which was checked out as %s", remoteBranch, r.Branch())
	}
	rs := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", r.Branch(), remoteBranch)
	log.Trace().Str("refspec", rs).Msg("for push")
	refspec := config.RefSpec(rs)
	err := refspec.Validate()
	if err != nil {
		return fmt.Errorf("refspec %s failed validation", rs)
	}
	err = r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refspec},
		Auth:       r.auth,
		Progress:   os.Stdout,
		// Force:      false,
		// ForceWithLease: &git.ForceWithLease{
		// 	RefName: plumbing.NewBranchReferenceName(remoteBranch),
		// },
		InsecureSkipTLS: false,
	})
	if err == git.NoErrAlreadyUpToDate {
		log.Debug().Err(err).Msg("push -already up to date remote")
		err = nil
	}
	if err != nil {
		return fmt.Errorf("pushing: %w", err)
	}
	return nil
}

// DeleteRemoteBranch deletes the given branch from the remote origin,
// this is mainly used in the test functions to delete the test branches,
// but can also be called from other contexts.
// Please note that it operates only on the origin remote.
func (r *GitRepo) DeleteRemoteBranch(remoteBranch string) error {
	if remoteBranch == "" {
		return git.ErrBranchNotFound
	}
	remote, err := r.repo.Remote("origin")
	if err != nil {
		return err
	}
	rs := fmt.Sprintf(":refs/heads/%s", remoteBranch)
	err = config.RefSpec(rs).Validate()
	if err != nil {
		return err
	}
	err = remote.Push(&git.PushOptions{
		RefSpecs:        []config.RefSpec{config.RefSpec(rs)},
		Auth:            r.auth,
		Progress:        os.Stdout,
		Force:           false,
		InsecureSkipTLS: false,
	})
	if err != nil {
		return err
	}
	return nil
}

// (r *GitRepo) Branches will return a list of branches matching the supplied regexp for the repo
func (r *GitRepo) Branches(re string) ([]string, error) {
	remote, err := r.repo.Remote("origin")
	if err != nil {
		panic(err)
	}
	refList, err := remote.List(&git.ListOptions{
		Auth:            r.auth,
		InsecureSkipTLS: false,
	})
	if err != nil {
		panic(err)
	}
	refPrefix := "refs/heads/"
	regexp := regexp.MustCompile(re)
	var branches []string
	for _, ref := range refList {
		refName := ref.Name().String()
		if !strings.HasPrefix(refName, refPrefix) {
			continue
		}
		branchName := refName[len(refPrefix):]
		if regexp.MatchString(branchName) {
			branches = append(branches, ref.Name().Short())
		}
	}
	return branches, nil
}

// CreateFile will create a file in a directory, truncating it if it already exists with the embedded git worktree.
// Any intermediate directories are also created.
func (r *GitRepo) CreateFile(path string) (*os.File, error) {
	op, err := os.Create(filepath.Join(r.dir, path))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return op, nil
}

// (r *GitRepo) EnableSignging will enable commits to be signed for this repo
func (r *GitRepo) EnableSigning(key *openpgp.Entity) error {
	if key != nil {
		r.commitOpts.SignKey = key
	} else {
		return fmt.Errorf("signing key is nil")
	}
	return nil
}
