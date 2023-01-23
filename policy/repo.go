package policy

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/TykTechnologies/gromit/git"
)

// RepoPolicy extracts information from the Policies type for one repo. If you add fields here, the Policies type might have to be updated, and vice versa.
type RepoPolicy struct {
	Name                string
	Description         string
	Protected           []string `copier:"-"`
	Default             string
	PCRepo              string
	DHRepo              string
	CSRepo              string
	Binary              string
	PackageName         string
	Reviewers           []string
	ExposePorts         string
	Files               map[string][]string
	Ports               map[string][]string
	gitRepo             *git.GitRepo
	Branch              string
	ReleaseBranches     map[string]branchVals
	prBranch            string
	Branchvals          branchVals
	prefix              string
	Timestamp           string
	Wiki                bool
	Topics              []string `copier:"-"`
	VulnerabilityAlerts bool
	SquashMsg           string
	SquashTitle         string
	Visibility          string
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
func (r RepoPolicy) IsProtected(branch string) bool {
	for _, pb := range r.Protected {
		if pb == branch {
			return true
		}
	}
	return false
}

// GetOwner returns the owner part of a given github oprg prefix fqdn, returns
// error if not a valid github fqdn.
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

// SetTimestamp Sets the given time as the repopolicy timestamp. If called with zero time
// sets the current time in UTC
func (r *RepoPolicy) SetTimestamp(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	r.Timestamp = ts.Format(time.UnixDate)

}

// GetTimeStamp returns the timestamp currently set for the given repopolicy.
func (r RepoPolicy) GetTimeStamp() (time.Time, error) {
	var ts time.Time
	var err error
	ts, err = time.Parse(time.UnixDate, r.Timestamp)
	return ts, err
}
