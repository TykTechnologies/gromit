package policy

import (
	"fmt"
	"time"

	"bytes"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

// Policies models the config file structure. Each element here _has_
// to match the name of the key used in the config yaml. There are
// three levels at which a particular value can be set: top-level,
// repo, branch. The top level is applicable for all the repos. The
// recursive map[string]Policies element embeds this type inside
// itself, allowing each repo to override any of the values at upper
// levels
type Policies struct {
	Description    string
	PCRepo         string
	DHRepo         string
	CSRepo         string
	PCPrivate      bool
	PackageName    string
	Reviewers      []string
	ExposePorts    string
	Binary         string
	Buildenv       string
	Cgo            bool
	ConfigFile     string
	VersionPackage string
	VersionKeys    versionKeys
	UpgradeFromVer string
	Features       []string
	Branches       map[string]branchVals `copier:"-"`
	Repos          map[string]Policies   `copier:"-"`
}

// branchVals contains only the parameters that can be overriden at
// the branch level. Some elements are overriden, some elements are
// concatenated. See policy.GetRepo to see how definitions are
// processed at each level
type branchVals struct {
	Buildenv       string
	Cgo            bool
	ConfigFile     string
	VersionPackage string
	VersionKeys    versionKeys
	UpgradeFromVer string
	Convos         bool
	ReviewCount    int
	Tests          []string
	SourceBranch   string
	Features       []string
}

// versionKeys holds the information about all the version
// related variables set during the binary build.
type versionKeys struct {
	Version   string
	Commit    string
	BuildDate string
	BuiltBy   string
}

// RepoPolicy is used to render templates. It provides an abstraction
// between config.yaml and the templates. It is instantiated from
// Policies for a particular repo and branch and the constructor
// implements all the overriding/merging logic between the various
// levels of the Policies type.
type RepoPolicy struct {
	Name           string
	Description    string
	Default        string
	PCPrivate      bool
	PCRepo         string
	DHRepo         string
	CSRepo         string
	Binary         string
	PackageName    string
	Reviewers      []string
	ExposePorts    string
	Cgo            bool
	ConfigFile     string
	VersionPackage string
	VersionKeys    versionKeys
	UpgradeFromVer string
	Branch         string
	Branchvals     branchVals
	Branches       map[string]branchVals
	prBranch       string
	Timestamp      string
	Visibility     string
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
func (rp *RepoPolicy) GetTimeStamp() (time.Time, error) {
	var ts time.Time
	var err error
	ts, err = time.Parse(time.UnixDate, rp.Timestamp)
	return ts, err
}

// SetBranch sets the Branch and Branchvals properties so that templates can simply access them instead of looking them up in the Branches map
func (rp *RepoPolicy) SetBranch(branch string) error {
	bv, found := rp.Branches[branch]
	if !found {
		return fmt.Errorf("branch %s unknown among %v", branch, rp.Branches)
	}
	rp.Branch = branch
	rp.Branchvals = bv

	return nil
}

// GetAllBranches returns all the branches that are managed for this repo
func (rp *RepoPolicy) GetAllBranches() []string {
	return maps.Keys(rp.Branches)
}

// GetRepoPolicy will fetch the RepoPolicy for the supplied repo with
// all overrides processed. This is the constructor for RepoPolicy.
func (p *Policies) GetRepoPolicy(repo string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}

	var rp RepoPolicy
	rp.Name = repo
	// Copy policy level elements
	err := copier.CopyWithOption(&rp, &p, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return rp, err
	}
	// Override policy level elements with repo level
	err = copier.CopyWithOption(&rp, &r, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return rp, err
	}

	allBranches := make(map[string]branchVals)
	for b, bbv := range r.Branches {
		var rbv branchVals // repo level branchvals
		// copy top-level options
		err := copier.CopyWithOption(&rbv, &p, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// override with repo-level options
		err = copier.CopyWithOption(&rbv, &r, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// override with branch level
		err = copier.CopyWithOption(&rbv, &bbv, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// add features from top-level and repo level
		rbv.Features = append(p.Features, r.Features...)
		// add features from branch
		rbv.Features = append(rbv.Features, bbv.Features...)
		log.Trace().Interface("bv", rbv).Str("branch", b).Msg("computed")
		allBranches[b] = rbv
	}
	rp.Branches = allBranches
	return rp, nil
}

// ProcessBranch will render the templates into a git worktree for the supplied branch, commit and push the changes upstream
// The upstream branch name is the supplied branch name prefixed with releng/ and is returned
func (rp *RepoPolicy) ProcessBranch(opDir, branch, msg string, repo *GitRepo) (string, error) {
	log.Info().Msgf("processing branch %s", branch)
	err := repo.FetchBranch(branch)
	if err != nil {
		return "", fmt.Errorf("git checkout %s/%s:%s: %v", repo.Owner, repo.Name, branch, err)
	}
	err = rp.SetBranch(branch)
	if err != nil {
		return "", err
	}
	b, err := NewBundle(rp.Branchvals.Features)
	if err != nil {
		return "", fmt.Errorf("bundle %v: %v", rp.Branchvals.Features, err)
	}
	files, err := b.Render(&rp, opDir, nil)
	log.Info().Strs("files", files).Msg("Rendered files")
	if err != nil {
		return "", fmt.Errorf("bundle gen %v: %v", rp.Branchvals.Features, err)
	}
	dfs, err := NonTrivialDiff(opDir)
	if err != nil {
		return "", fmt.Errorf("computing diff in %s: %v", opDir, err)
	}
	remoteBranch := fmt.Sprintf("releng/%s", branch)
	if len(dfs) == 0 {
		log.Info().Msgf("trivial changes for repo %s branch %s, stopping here", repo.Name, repo.Branch())
		return remoteBranch, nil
	}
	// Add rendered files to git staging.
	for _, f := range files {
		_, err := repo.AddFile(f)
		if err != nil {
			return "", fmt.Errorf("staging file to git worktree: %v", err)
		}
	}
	err = repo.Commit(msg)
	if err != nil {
		return "", fmt.Errorf("git commit %s: %v", repo.Name, err)
	}
	err = repo.Push(remoteBranch)
	if err != nil {
		return remoteBranch, fmt.Errorf("git push %s %s:%s: %v", repo.Name, repo.Branch(), remoteBranch, err)
	}
	log.Info().Msgf("pushed %s to %s", remoteBranch, rp.Name)

	return remoteBranch, nil
}

// Stringer implementation for Policies
func (p Policies) String() string {
	w := new(bytes.Buffer)
	for repo, crPol := range p.Repos {
		fmt.Fprintf(w, "%s: package %s, image %s", repo, crPol.PackageName, crPol.DHRepo)
		rp, err := p.GetRepoPolicy(repo)
		if err != nil {
			log.Fatal().Str("repo", repo).Err(err).Msg("failed to get policy, this should not happen")
		}
		fmt.Fprintf(w, " %s\n", rp)
	}
	return w.String()
}

// Stringer implementation for RepoPolicy
func (rp RepoPolicy) String() string {
	w := new(bytes.Buffer)
	for b, bv := range rp.Branches {
		fmt.Fprintf(w, " %s: package %s, image %s, features %v", b, rp.PackageName, rp.DHRepo, bv.Features)
		if len(bv.Buildenv) > 0 {
			fmt.Fprintf(w, " built on %s", bv.Buildenv)
		} else {
			fmt.Fprintf(w, " not built")
		}
	}
	return w.String()
}

// LoadRepoPolicies populates the supplied policies with the policy key from a the config file
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	return viper.UnmarshalKey("policy", policies)
}
