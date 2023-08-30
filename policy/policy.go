package policy

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

// Policies models the config file structure. The config file may
// contain one or more repos in the Repos map, keyed by their short
// name. Each repo in Repos can override any of the values in the
// Policies struct just for itself.
type Policies struct {
	Default     string
	Description string
	PCRepo      string
	DHRepo      string
	CSRepo      string
	PackageName string
	Reviewers   []string
	ExposePorts string
	Binary      string
	Features    []string
	Buildenv    string
	Repos       map[string]Policies // recursive for overrides
	Branchvals  branchVals          // this is not present actually at the top-level, only for repos
	Visibility  string
}

// branchVals contains the parameters that are specific to a
// particular branch in a repo. This private type links the Policies
// (which maps to a config file) and the RepoPolicy (which is used to
// render templates).
// Please discuss _before_ adding elements here
type branchVals struct {
	Buildenv       string
	PCPrivate      bool
	Cgo            bool
	ConfigFile     string
	VersionPackage string
	UpgradeFromVer string
	Branches       map[string]branchVals `copier:"-"`
	ReviewCount    string
	Convos         bool
	Tests          []string
	SourceBranch   string
	Features       []string
}

// RepoPolicies aggregates RepoPolicy, indexed by repo name.
type RepoPolicies map[string]RepoPolicy

// RepoPolicy is used to render templates. It provides an abstraction
// between config.yaml and the templates and is used to merge and override values making the template renderer simpler.
// Please discuss _before_ adding elements here.
type RepoPolicy struct {
	Name        string
	Description string
	Default     string
	PCRepo      string
	DHRepo      string
	CSRepo      string
	Binary      string
	PackageName string
	Reviewers   []string
	ExposePorts string
	Branch      string
	prBranch    string
	Branchvals  branchVals
	prefix      string
	Timestamp   string
	Visibility  string
}

// GetOwner returns the owner part of a given github oprg prefix fqdn, returns
// error if not a valid github fqdn.
func (r *RepoPolicy) GetOwner() (string, error) {
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

// SetTimestamp Sets the given time as the repopolicy timestamp. If called with zero time
// sets the current time in UTC
func (r *RepoPolicy) SetTimestamp(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	r.Timestamp = ts.Format(time.UnixDate)

}

// GetTimeStamp returns the timestamp currently set for the given repopolicy.
func (r *RepoPolicy) GetTimeStamp() (time.Time, error) {
	var ts time.Time
	var err error
	ts, err = time.Parse(time.UnixDate, r.Timestamp)
	return ts, err
}

// GetRepoPolicy will fetch the RepoPolicy with all overrides processed
func (p *Policies) GetRepoPolicy(repo string, branch string) (RepoPolicy, error) {
	return p.GetRepo(repo, viper.GetString("prefix"), branch)
}

// GetAllRepos returns a map of reponame->repopolicy for all the
// repos in the policy config.
func (p *Policies) GetAllRepos(prefix string) (RepoPolicies, error) {
	var rp RepoPolicies
	for repoName, repoVals := range p.Repos {
		log.Info().Msgf("Reponame: %s", repoName)
		repo, err := repoVals.GetRepo(repoName, prefix, "master")
		if err != nil {
			return RepoPolicies{}, err
		}
		rp[repoName] = repo
	}

	return rp, nil
}

// GetRepo will give you a RepoPolicy struct for a repo which can be used to feed templates
func (p *Policies) GetRepo(repo, prefix, branch string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}

	var bv branchVals

	copier.Copy(&bv, r.Branchvals)
	// Override policy values
	copier.CopyWithOption(&p, &r, copier.Option{IgnoreEmpty: true})

	// Check if the branch has a branch specific policy in the config and override the
	// common branch values with the branch specific ones.
	if ib, found := r.Branchvals.Branches[branch]; found {
		copier.CopyWithOption(&bv, &ib, copier.Option{IgnoreEmpty: true})
		// features need to be merged
		bv.Features = append(r.Features, ib.Features...)
	}

	return RepoPolicy{
		Name:        repo,
		Default:     p.Default,
		Branch:      branch,
		Branchvals:  bv,
		prefix:      prefix,
		Reviewers:   r.Reviewers,
		DHRepo:      r.DHRepo,
		PCRepo:      r.PCRepo,
		CSRepo:      r.CSRepo,
		ExposePorts: r.ExposePorts,
		Binary:      r.Binary,
		Description: r.Description,
		PackageName: r.PackageName,
		Visibility:  p.Visibility,
	}, nil
}

// String representation
func (p Policies) String() string {
	w := new(bytes.Buffer)
	//fmt.Fprintln(w)
	for repo, crPol := range p.Repos {
		fmt.Fprintf(w, "%s:\n", repo)
		fmt.Fprintf(w, "%v:\n", crPol)
		for _, branch := range maps.Keys(crPol.Branchvals.Branches) {
			rp, err := p.GetRepoPolicy(repo, branch)
			if err != nil {
				log.Fatal().Str("repo", repo).Str("branch", branch).Err(err).Msg("failed to get policy, this should not happen")
			}
			fmt.Fprintf(w, "%v\n", rp)
		}
	}
	return w.String()
}

// LoadRepoPolicies returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	return viper.UnmarshalKey("policy", policies)
}
