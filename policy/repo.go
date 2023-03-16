package policy

import (
	"time"

	"github.com/TykTechnologies/gromit/git"
)

// RepoPolicy extracts information from the Policies type for one repo. If you add fields here, the Policies type might have to be updated, and vice versa.
type RepoPolicy struct {
	Name                string
	Description         string
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

// SetTimestamp Sets the given time as the repopolicy timestamp. If called with zero time
// sets the current time in UTC
func (r *RepoPolicy) SetTimestamp(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	r.Timestamp = ts.Format(time.UnixDate)

}
