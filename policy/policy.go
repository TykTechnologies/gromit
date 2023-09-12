package policy

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"bytes"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Policies models the config file structure. Each element here _has_
// to match the name of the key used in the config yaml. There are
// three levels at which a particular value can be set: top-level,
// repo, branch. The top level is applicable for all the repos. The
// recursive map[string]Policies element embeds this type inside
// itself, allowing each repo to override any of the values at upper
// levels
type Policies struct {
	Description string
	PCRepo      string
	DHRepo      string
	CSRepo      string
	PCPrivate   bool
	PackageName string
	Reviewers   []string
	ExposePorts string
	Binary      string
	Features    []string
	Buildenv    string
	Branches    map[string]branchVals
	Repos       map[string]Policies
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
	UpgradeFromVer string
	Convos         bool
	ReviewCount    int
	Tests          []string
	SourceBranch   string
	Features       []string
}

// RepoPolicy is used to render templates. It provides an abstraction
// between config.yaml and the templates. It is instantiated from
// Policies for a particular repo and branch and the constructor
// implements all the overriding/merging logic between the various
// levels of the Policies type.
type RepoPolicy struct {
	Name        string
	Description string
	Default     string
	PCPrivate   bool
	PCRepo      string
	DHRepo      string
	CSRepo      string
	Binary      string
	PackageName string
	Reviewers   []string
	ExposePorts string
	Branch      string
	Branchvals  branchVals
	Branches    map[string]branchVals
	prBranch    string
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

// GetRepoPolicy will fetch the RepoPolicy for the supplied repo with
// all overrides processed. This is the constructor for RepoPolicy.
// The supplied branch populates the Branch and Branchvals properties
// which are used in the templates
func (p *Policies) GetRepoPolicy(repo, branch string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}
	_, found = r.Branches[branch]
	if !found {
		return RepoPolicy{}, fmt.Errorf("branch %s unknown among %v", branch, r.Branches)
	}

	allBranches := make(map[string]branchVals)
	for b, bbv := range r.Branches {
		var rbv branchVals // repo level branchvals
		// copy top-level options
		copier.CopyWithOption(&rbv, &p, copier.Option{IgnoreEmpty: true})
		// override with branch level
		copier.CopyWithOption(&rbv, &bbv, copier.Option{IgnoreEmpty: true})
		// add features from top-level and repo level
		rbv.Features = append(p.Features, r.Features...)
		// add features from branch
		rbv.Features = append(rbv.Features, bbv.Features...)
		log.Trace().Interface("bv", rbv).Str("branch", b).Msg("computed")
		allBranches[b] = rbv
	}
	return RepoPolicy{
		Name:        repo,
		Branch:      branch,
		Branchvals:  allBranches[branch],
		Branches:    allBranches,
		Reviewers:   r.Reviewers,
		DHRepo:      r.DHRepo,
		PCRepo:      r.PCRepo,
		PCPrivate:   r.PCPrivate,
		CSRepo:      r.CSRepo,
		ExposePorts: r.ExposePorts,
		Binary:      r.Binary,
		Description: r.Description,
		PackageName: r.PackageName,
	}, nil
}

// Stringer implementation
func (p Policies) String() string {
	w := new(bytes.Buffer)
	for repo, crPol := range p.Repos {
		fmt.Fprintf(w, "%s: package %s, image %s", repo, crPol.PackageName, crPol.DHRepo)
		for b := range crPol.Branches {
			rp, err := p.GetRepoPolicy(repo, b)
			if err != nil {
				log.Fatal().Str("repo", repo).Err(err).Msg("failed to get policy, this should not happen")
			}
			fmt.Fprintf(w, " %s\n", rp)
		}
	}
	return w.String()
}

// Stringer implementation
func (rp RepoPolicy) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintf(w, " %s: package %s, image %s, features %v", rp.Branch, rp.PackageName, rp.DHRepo, rp.Branchvals.Features)
	if len(rp.Branchvals.Buildenv) > 0 {
		fmt.Fprintf(w, " built on %s", rp.Branchvals.Buildenv)
	} else {
		fmt.Fprintf(w, " not built")
	}
	return w.String()
}

// LoadRepoPolicies populates the supplied policies with the policy key from a the config file
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	return viper.UnmarshalKey("policy", policies)
}
