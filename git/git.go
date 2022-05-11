package git

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/oauth2"
)

type GitRepo struct {
	Name         string
	commitOpts   *git.CommitOptions
	repo         *git.Repository
	branch       string // local branch
	remoteBranch string // remote branch
	worktree     *git.Worktree
	fs           billy.Filesystem
	auth         transport.AuthMethod
	gh           *github.Client
	prs          []string
	dryRun       bool
}

const defaultRemote = "origin"

// FetchRepo clones a repo into the given dir or an in-memory fs
// pass depth=0 for full clone
// if an authtoken is passed am authenticated github client is enabled
func FetchRepo(fqdnRepo, dir, authToken string, depth int) (*GitRepo, error) {
	log.Debug().Str("repo", fqdnRepo).Str("dir", dir).Int("depth", depth).Msg("fetching repo")
	opts := &git.CloneOptions{
		URL:      fqdnRepo,
		Progress: os.Stdout,
		// FIXME: https://github.com/go-git/go-git/issues/207
		//Depth: depth,
	}
	if depth > 0 {
		opts.Depth = depth
	}
	var gh *github.Client
	if authToken != "" {
		opts.Auth = &http.BasicAuth{
			Username: "abc123", // anything except an empty string
			Password: authToken,
		}
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: authToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		gh = github.NewClient(tc)
	}
	log.Trace().Interface("opts", opts).Msg("git clone options")
	var repo *git.Repository
	var fs billy.Filesystem
	var err error
	if dir == "" {
		log.Info().Msg("using in-memory clone")
		fs = memfs.New()
		repo, err = git.Clone(memory.NewStorage(), fs, opts)
	} else {
		log.Info().Str("dir", dir).Msg("using existing clone")
		fs = osfs.New(dir)
		repo, err = git.PlainOpen(dir)
	}
	if err != nil {
		return nil, err
	}
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return &GitRepo{
		Name:     fqdnRepo,
		auth:     opts.Auth,
		repo:     repo,
		worktree: w,
		fs:       fs,
		gh:       gh,
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

// AddFile adds a file in the worktree to the index and then displays the resulting changeset as a patch.
// The file is assumed to have been updated prior to calling this function.
// It will show the changes commited in the form of a patch to stdout and wait for user confirmation.
// Note that this commit will be lost if it is not pushed to a remote.
func (r *GitRepo) AddFile(path, msg string, confirm bool) (plumbing.Hash, error) {
	origRef, err := r.repo.Head()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting hash for original head: %w", err)
	}
	log.Trace().Str("ref", origRef.String()).Msg("HEAD")
	origHead, err := r.repo.CommitObject(origRef.Hash())
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting original head: %w", err)
	}
	log.Trace().Str("hash", origHead.String()).Msg("HEAD")
	hash, err := r.worktree.Add(path)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("adding to worktree: %w", err)
	}
	log.Trace().Str("hash", hash.String()).Msg("add")
	newCommitHash, err := r.worktree.Commit(msg, r.commitOpts)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("committing to worktree: %w", err)
	}
	log.Trace().Str("hash", newCommitHash.String()).Msg("add to worktree")
	newCommit, err := r.repo.CommitObject(newCommitHash)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting new commit: %w", err)
	}
	log.Trace().Str("hash", newCommit.String()).Msg("new commit")

	patch, err := origHead.Patch(newCommit)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting diff: %w", err)
	}
	err = patch.Encode(os.Stdout)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding diff: %w", err)
	}
	if confirm {
		fmt.Printf("\n----End of diff for %s. Control-C to abort, ⏎/Enter to continue.", r.Name)
		fmt.Scanln()
	}

	return newCommitHash, nil
}

// (r *GitRepo) Checkout fetches the given ref and then checks it out to the worktree
// Any local changes are lost
func (r *GitRepo) Checkout(branch string) error {
	err := r.worktree.Clean(&git.CleanOptions{
		Dir: true,
	})
	if err != nil {
		return fmt.Errorf("cleaning: %w", err)
	}
	err = r.worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("resetting: %w", err)
	}
	refspec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch))
	err = r.repo.Fetch(&git.FetchOptions{
		RemoteName:      "origin",
		RefSpecs:        []config.RefSpec{refspec},
		Auth:            r.auth,
		Progress:        os.Stdout,
		InsecureSkipTLS: false,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Debug().Msg("fetching failed, re-trying")
		err = r.repo.Fetch(&git.FetchOptions{
			RemoteName:      "origin",
			RefSpecs:        []config.RefSpec{refspec},
			Auth:            r.auth,
			Progress:        os.Stdout,
			InsecureSkipTLS: false,
		})
		if err != nil {
			return fmt.Errorf("re-tried fetching: %w", err)
		}
	}
	branchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
	err = r.worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	r.branch = branch
	return nil
}

// Push will push the current worktree to origin
// If remoteBranch is empty, then it pushes to same branch as the local checkout
func (r *GitRepo) Push(branch, remoteBranch string) error {
	if remoteBranch == "" {
		remoteBranch = branch
	}
	if remoteBranch == branch {
		log.Warn().Msg("pushing to same branch as checkout")
	}
	rs := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, remoteBranch)
	log.Trace().Str("refspec", rs).Msg("for push")
	refspec := config.RefSpec(rs)
	err := refspec.Validate()
	if err != nil {
		return fmt.Errorf("refspec %s failed validation", rs)
	}
	if r.dryRun {
		log.Warn().Msg("only dry-run, not really pushing")
	} else {
		err = r.repo.Push(&git.PushOptions{
			RemoteName:      "origin",
			RefSpecs:        []config.RefSpec{refspec},
			Auth:            r.auth,
			Progress:        os.Stdout,
			Force:           false,
			InsecureSkipTLS: false,
		})
		if err != nil {
			return fmt.Errorf("pushing: %w", err)
		}
	}
	r.remoteBranch = remoteBranch
	return nil
}

// (r *GitRepo) Branches will return a list of branches matching the supplied regexp for the repo
func (r *GitRepo) Branches(re string) ([]string, error) {
	remote, err := r.repo.Remote(defaultRemote)
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
func (r *GitRepo) CreateFile(path string) (billy.File, error) {
	op, err := r.fs.Create(path)
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

// (r *GitRepo) PRs returns the URLs of any PRs created so far
func (r *GitRepo) PRs() []string {
	return r.prs
}

// (r *GitRepo) SetDryRun(true) will make this repo not perform any destructive action
func (r *GitRepo) SetDryRun(dryRun bool) {
	r.dryRun = dryRun
}
