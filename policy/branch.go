package policy

import (
	"fmt"
)

type branchPolicies struct {
	Protected    []string            `mapstructure:",omitempty"`
	Deprecations map[string][]string `mapstructure:",omitempty"`
	Backports    map[string]string   `mapstructure:",omitempty"`
	Files        []string            `mapstructure:",omitempty"`
}

// (bp branchPolicies) SrcBranches returns the list of branches for which automatic backport sync is implemented
// i.e. commits landing on these branches will be sync'd to the backport branch
func (bp branchPolicies) SrcBranches() []string {
	keys := make([]string, 0, len(bp.Backports))
	for k := range bp.Backports {
		keys = append(keys, k)
	}
	return keys
}

// (bp branchPolicies) BackportBranch returns the backport branch for srcBranch which will be the source of commits
func (bp branchPolicies) BackportBranch(srcBranch string) (string, error) {
	destBranch, found := bp.Backports[srcBranch]
	if !found {
		return "", fmt.Errorf("branch %s unknown among %v", srcBranch, bp.Backports)
	}
	return destBranch, nil
}
