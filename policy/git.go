package policy

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"time"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v47/github"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// GitRepo models a local git worktree with the authentication and
// enough metadata to allow it to be pushed it to github
type GitRepo struct {
	Name       string
	Owner      string
	commitOpts *git.CommitOptions
	repo       *git.Repository
	RepoPolicy RepoPolicy
	worktree   *git.Worktree
	dir        string
	auth       transport.AuthMethod
	gh         *github.Client
	ghV4       *githubv4.Client
	prs        []string
	dryRun     bool
}

const defaultRemote = "origin"

// InitGit is a constructor for the GitRepo type
// private repos will need ghToken
func Init(repoName, owner, branch string, depth int, dir, ghToken string) (*GitRepo, error) {
	log.Logger = log.With().Str("repo", repoName).Str("branch", branch).Str("owner", owner).Logger()

	fi, err := os.Stat(dir)
	if os.IsNotExist(err) || !fi.IsDir() {
		log.Debug().Str("dir", dir).Msg("does not exist")
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}

	var gh *github.Client
	var ghV4 *githubv4.Client

	fqrn := fmt.Sprintf("https://github.com/%s/%s", owner, repoName)
	cloneOpts := &git.CloneOptions{
		URL:      fqrn,
		Progress: os.Stdout,
		// FIXME: Make a shallow clone https://github.com/go-git/go-git/issues/207
		Depth:         depth,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if ghToken != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "abc123", // anything except an empty string
			Password: ghToken,
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghToken},
		)
		tc := oauth2.NewClient(context.Background(), ts)

		gh = github.NewClient(tc)
		ghV4 = githubv4.NewClient(tc)
	}

	var repo *git.Repository
	repo, err = git.PlainOpen(dir)
	if err == git.ErrRepositoryNotExists {
		repo, err = git.PlainClone(dir, false, cloneOpts)
	}
	// Load repo policy for the given repo.
	var cp Policies
	err = LoadRepoPolicies(&cp)
	if err != nil {
		log.Fatal().Msg("Could not load config policies")
	}
	rp, err := cp.GetRepoPolicy(repoName, branch)
	if err != nil {
		return nil, err
	}
	w, err := repo.Worktree()
	if err != nil {
		log.Error().Err(err).Msg("Error getting worktree")
		return nil, err
	}
	/*  FIXME: Re enable pulling once issue #305(https://github.com/go-git/go-git/issues/305) is fixed.
	err = w.Pull(&git.PullOptions{
		SingleBranch:  true,
		Progress:      os.Stdout,
		Auth:          cloneOpts.Auth,
		ReferenceName: cloneOpts.ReferenceName,
	})
	if err == plumbing.ErrReferenceNotFound {
		log.Debug().Err(err).Str("branch", branch).Str("remote", fqrn).Msg("does not exist, the branch will get created")
		err = nil
	}
	if err == git.NoErrAlreadyUpToDate {
		log.Debug().Err(err).Str("branch", branch).Str("remote", fqrn).Msg("brnach already up-to-date")
		err = nil
	}
	// to mitigate https://github.com/go-git/go-git/issues/328 temporarily until it gets fixed.
	if err == transport.ErrEmptyUploadPackRequest {
		log.Debug().Err(err).Str("branch", branch).Str("remote", fqrn).Msg("empty upload pack request- https://github.com/go-git/go-git/issues/328")
		err = nil
	}*/

	return &GitRepo{
		Name:       repoName,
		Owner:      owner,
		auth:       cloneOpts.Auth,
		repo:       repo,
		worktree:   w,
		dir:        dir,
		gh:         gh,
		ghV4:       ghV4,
		RepoPolicy: rp,
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

// SwitchBranch will create a new branch and switch the
// worktree to it.
func (r *GitRepo) SwitchBranch(branch string) error {
	head, err := r.repo.Head()
	if err != nil {
		return err
	}
	nbrefName := plumbing.NewBranchReferenceName(branch)
	nbRef := plumbing.NewHashReference(nbrefName, head.Hash())
	err = r.repo.Storer.SetReference(nbRef)
	if err != nil {
		return err
	}
	err = r.worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(nbrefName),
		Force:  true,
	})
	if err != nil {
		return err
	}
	return err
}

// (r *GitRepo) Checkout fetches the given ref and then checks it out to the worktree
// Any local changes are lost. If the branch does not exist, it is not created.
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
	localRef := plumbing.NewBranchReferenceName(branch)
	remoteRef := plumbing.NewRemoteReferenceName("origin", branch)
	err = r.repo.Storer.SetReference(plumbing.NewSymbolicReference(localRef, remoteRef))
	if err != nil {
		return err
	}
	err = r.worktree.Checkout(&git.CheckoutOptions{
		Branch: localRef,
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	return nil
}

// Push will push the current worktree to origin
func (r *GitRepo) Push(remoteBranch string) error {
	if remoteBranch == r.Branch() {
		log.Warn().Msg("pushing to same branch as checkout")
	}
	rs := fmt.Sprintf("+refs/heads/%s:refs/heads/%s", r.Branch(), remoteBranch)
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
		if err == git.NoErrAlreadyUpToDate {
			log.Debug().Err(err).Msg("push -already up to date remote")
			err = nil
		}
		if err != nil {
			return fmt.Errorf("pushing: %w", err)
		}
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

// (r *GitRepo) PRs returns the URLs of any PRs created so far
func (r *GitRepo) PRs() []string {
	return r.prs
}

// EnableAutoMergePR uses the graphQL github v4 API with the PR ID
// (not number) to mutate graphQL PR object to enable automerge
func (r *GitRepo) EnableAutoMerge(prID string) error {
	var mutation struct {
		Automerge struct {
			ClientMutationID githubv4.String
			Actor            struct {
				Login githubv4.String
			}
			PullRequest struct {
				BaseRefName githubv4.String
				CreatedAt   githubv4.DateTime
				Number      githubv4.Int
			}
		} `graphql:"enablePullRequestAutoMerge(input: $input)"`
	}

	mergeMethod := githubv4.PullRequestMergeMethodSquash

	amInput := githubv4.EnablePullRequestAutoMergeInput{
		MergeMethod:   &mergeMethod,
		PullRequestID: prID,
	}

	return r.ghV4.Mutate(context.Background(), &mutation, amInput, nil)
}

// RenderPRTemplate will fill in the supplied template body with values from GitRepo
func (r *GitRepo) RenderPRTemplate(body *string) (*bytes.Buffer, error) {
	op := new(bytes.Buffer)
	t := template.Must(
		template.New("prbody").
			Option("missingkey=error").
			Funcs(sprig.FuncMap()).
			Parse(*body))
	err := t.Execute(op, r)
	return op, err
}

//go:embed prs/main.tmpl
var prbody string

// CreatePR will create a PR using the user supplied title and the embedded PR body
// If a PR already exists, it will return that PR
func (r *GitRepo) CreatePR(prtitle, remoteBranch string, draft bool) (*github.PullRequest, error) {
	body, err := r.RenderPRTemplate(&prbody)
	if err != nil {
		return nil, err
	}
	title, err := r.RenderPRTemplate(&prtitle)
	if err != nil {
		return nil, err
	}

	prOpts := &github.NewPullRequest{
		Title: github.String(title.String()),
		Head:  github.String(remoteBranch),
		Base:  github.String(r.Branch()),
		Body:  github.String(body.String()),
		Draft: github.Bool(draft),
	}
	log.Trace().Interface("propts", prOpts).Str("owner", r.Owner).Str("repo", r.Name).Msg("creating PR")
	if r.dryRun {
		return nil, nil
	}
	pr, resp, err := r.gh.PullRequests.Create(context.Background(), r.Owner, r.Name, prOpts)
	// Attempt to detect if a PR already existingPR, complexity due to
	// https://github.com/google/go-github/issues/1441
	existingPR := false
	if e, ok := err.(*github.ErrorResponse); ok {
		for _, ghErr := range e.Errors {
			if strings.HasPrefix(ghErr.Message, "A pull request already exists") {
				log.Debug().Interface("ghErr", ghErr).Interface("resp", resp).Msg("found existing PR")
				existingPR = true
				break
			}
		}
	}
	if !existingPR && err != nil {
		return nil, fmt.Errorf("error creating PR for %s:%s: %v", r.Name, remoteBranch, err)
	} else if existingPR {
		prs, err := r.getPR(remoteBranch)
		if err != nil {
			return nil, fmt.Errorf("PR %s:%s exists but could not be fetched: %v", r.Name, remoteBranch, err)
		}
		// Only one PR for a given head
		pr = prs[0]
	}
	return pr, nil
}

// getPR searches for PRs created for the head ref/branch
func (r *GitRepo) getPR(head string) ([]*github.PullRequest, error) {
	prlOpts := &github.PullRequestListOptions{
		Head: head,
	}
	prs, resp, err := r.gh.PullRequests.List(context.Background(), r.Owner, r.Name, prlOpts)
	log.Trace().Interface("resp", resp).Msg("getting existing PR")
	return prs, err
}

// (r *GitRepo) SetDryRun(true) will make this repo not perform any destructive action
func (r *GitRepo) SetDryRun(dryRun bool) {
	r.dryRun = dryRun
}