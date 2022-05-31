package policy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/TykTechnologies/gromit/git"
	"github.com/TykTechnologies/gromit/util"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var ErrUnknownRepo = errors.New("repo not present in policies")
var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")
var ErrUnKnownBundle = errors.New("bundle not present in loaded policy")

// branchVals contains the parameters that are specific to a particular branch in a repo
type branchVals struct {
	Name               string // Branch name
	GoVersion          string
	Cgo                bool
	ConfigFile         string
	UpgradeFromVer     string // Versions to test package upgrades from
	SyncExcludeBundles []string
}

// Policies models the config file structure. The config file may contain one or more repos.
type Policies struct {
	Description string
	PCRepo      string
	DHRepo      string
	ExposePorts string
	Protected   []string
	Goversion   string
	Master      string              // The equivalent of the master branch
	Repos       map[string]Policies // map of reponames to branchPolicies
	Ports       map[string][]string
	Branches    []branchVals
}

// RepoPolicy extracts information from the Policies type for one repo. If you add fields here, the Policies type might have to be updated, and vice versa.
type RepoPolicy struct {
	Name       string
	Protected  []string
	Files      map[string][]string
	Ports      map[string][]string
	gitRepo    *git.GitRepo
	Branch     string
	prBranch   string
	branchvals branchVals
	prefix     string
	Timestamp  string
}

// GetRepo will give you a RepoPolicy struct for a repo which can be used to feed templates
// Though Ports can be defined at the global level they are not practically used and if defined will be ignored.
func (p *Policies) GetRepo(repo, prefix, branch string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}
	found = false
	var b branchVals
	for _, b = range r.Branches {
		if b.Name == branch {
			found = true
			break
		}
	}
	if !found {
		return RepoPolicy{}, fmt.Errorf("branch %s unknown for repo %s", branch, repo)
	}
	return RepoPolicy{
		Name:       repo,
		Protected:  append(p.Protected, r.Protected...),
		Ports:      r.Ports,
		Branch:     branch,
		prefix:     prefix,
		branchvals: b,
	}, nil
}

// SwitchBranch calls the SwitchBranch method of gitRepo and creates a new
// branch and switches the underlying git repo to the given branch - also
// sets prBranch to the newly checked out branch.
func (rp *RepoPolicy) SwitchBranch(branch string) error {
	err := rp.gitRepo.SwitchBranch(branch)
	if err != nil {
		return err
	}
	rp.prBranch = branch
	return nil
}

// SetTimestamp Sets the given time as the repopolicy timestamp. If called with zero time
// sets the current time in UTC
func (rp *RepoPolicy) SetTimestamp(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	rp.Timestamp = ts.Format(time.UnixDate)

}

// GetTimeStamp returns the timestamp currently set for the given repopolicy.
func (rp RepoPolicy) GetTimeStamp() (time.Time, error) {
	var ts time.Time
	var err error
	ts, err = time.Parse(time.UnixDate, rp.Timestamp)
	return ts, err
}

// Returns the destination branches for a given source branch
func (r RepoPolicy) DestBranches(srcBranch string) []string {
	b, found := r.Ports[srcBranch]
	if !found {
		return []string{}
	}
	return b
}

// IsProtected tells you if a branch can be pushed directly to origin or needs to go via a PR
func (rp RepoPolicy) IsProtected(branch string) bool {
	for _, pb := range rp.Protected {
		if pb == branch {
			return true
		}
	}
	return false
}

// InitGit initialises the corresponding git repo by fetching it
func (r *RepoPolicy) InitGit(depth int, signingKeyid uint64, dir, ghToken string) error {
	log.Logger = log.With().Str("repo", r.Name).Str("branch", r.Branch).Logger()
	fqdnRepo := fmt.Sprintf("%s/%s", r.prefix, r.Name)

	var err error
	r.gitRepo, err = git.FetchRepo(fqdnRepo, dir, ghToken, depth)
	if err != nil {
		return err
	}
	if signingKeyid != 0 {
		signer, err := util.GetSigningEntity(signingKeyid)
		if err != nil {
			return err
		}
		err = r.gitRepo.EnableSigning(signer)
		if err != nil {
			log.Warn().Err(err).Msg("commits will not be signed")
		}
	}
	return nil
}

// Commit commits the current worktree and then displays the resulting change as a patch,
// and returns the hash of the commit object that was committed.
// It will show the changes commited in the form of a patch to stdout and wait for user confirmation.
func (r RepoPolicy) Commit(msg string, confirm bool) (plumbing.Hash, error) {
	origHead, err := r.gitRepo.Head()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting hash for original head: %w", err)
	}
	newCommit, err := r.gitRepo.Commit(msg)
	if err != nil {
		return plumbing.ZeroHash, err
	}
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
	return newCommit.Hash, nil
}

// Push will push the current state of the repo to github
// If the branch is protected, it will be pushed to a branch prefixed with releng/
// Push should ideally be called only from CreatePR for pushing the changes
// before creating a PR from the current branch against a base branch.
func (r RepoPolicy) Push() error {
	// Never push directly to the base branch. r.Branch has the
	// base branch for which the policy is applicable for(eg: master, release-4
	// etc.) Check if current branch is not base branch and it's not protected
	// before pushing.
	remoteBranch := r.gitRepo.CurrentBranch()
	if remoteBranch == r.Branch {
		return fmt.Errorf("Pushing to the same branch as base branch not supported, remote: %s, base: %s", remoteBranch, r.Branch)
	}
	if r.IsProtected(remoteBranch) {
		return fmt.Errorf("given remote: %s is a protected branch", remoteBranch)
	}
	return r.gitRepo.Push(remoteBranch, remoteBranch)
}

// CreatePR creates a PR on the given github repo for the specified bundle, against the
// gien baseBranch and title. If dryRun is enabled, it prints out the parameters with
// which the PR will be generated to stdout. It returns the URL of the PR on success.
// Returns an empty string and no error on a successful dry run.
func (r *RepoPolicy) CreatePR(bundle, title, baseBranch string, dryRun bool) (string, error) {
	prURL := ""
	if r.Branch == "" {
		return prURL, fmt.Errorf("unknown local branch on repo %s when creating PR", r.Name)
	}
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}

	// Check if bundle templates are rendered, and get the contents.
	body, err := r.renderPR(bundle)
	if err != nil {
		return prURL, fmt.Errorf("Error rendering PR for the bundle: %s: %v", bundle, err)
	}
	log.Info().Msg("successfully rendered pr template")

	owner, err := r.GetOwner()
	if err != nil {
		return prURL, err
	}
	if dryRun {
		log.Warn().Msg("only dry-run, not really creating PR")
		fmt.Println("Only dry-run, not creating actual PR")
		fmt.Printf("\nPR will be created in \n\tOrg: %s\n\tWith branch: %s\n\tAgainst base Branch: %s\n\tWith title: %s\n", owner, r.gitRepo.CurrentBranch(), baseBranch, title)
		fmt.Printf("\tWith PR Body: \n%s\n", string(body))
	} else {
		// Push and then create PR.
		err = r.Push()
		if err != nil {
			return "", err
		}
		log.Info().Str("baseBranch", baseBranch).
			Str("title", title).Msg("calling CreatePR on github")
		prURL, err = r.gitRepo.CreatePR(baseBranch, title, string(body))
		if err != nil {
			return "", err
		}
	}
	return prURL, nil
}

func (r RepoPolicy) GetOwner() (string, error) {
	u, err := url.Parse(r.prefix)
	if err != nil {
		return "", err
	}
	if u.Hostname() != "github.com" {
		return "", fmt.Errorf("not github prefix: %s", u.Hostname())
	}
	owner := strings.TrimPrefix(u.Path, "/")
	return owner, nil
}

// String representation
func (rp Policies) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintln(w, `Commits landing on the Source branch are automatically sync'd to the list of Destinations. PRs will be created for the protected branch. Other branches will be updated directly.`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Protected branches: %v\n", rp.Protected)
	fmt.Fprintln(w, "Common Files:")
	for repo, pols := range rp.Repos {
		fmt.Fprintf(w, "%s\n", repo)
		fmt.Fprintln(w, " Extra files:")
		fmt.Fprintln(w, " Ports")
		for src, dest := range pols.Ports {
			fmt.Fprintf(w, "   - %s → %s\n", src, dest)
		}
	}
	fmt.Fprintln(w)
	return w.String()
}

func (rp Policies) dotGen(cg *cgraph.Graph) error {
	return nil
}

// (rp RepoPolicies) Graph returns a graphviz dot format representation of the policy
func (rp Policies) Graph(w io.Writer) error {
	g := graphviz.New()
	relgraph, err := g.Graph()
	if err != nil {
		return err
	}
	defer func() {
		if err := relgraph.Close(); err != nil {
			log.Fatal().Err(err).Msg("could not close graphviz")
		}
		g.Close()
	}()

	err = rp.dotGen(relgraph)
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyConfig returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
