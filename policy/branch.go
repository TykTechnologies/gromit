package policy

import "errors"

type branchPolicies struct {
	Protected    []string            `mapstructure:",omitempty"`
	Deprecations map[string][]string `mapstructure:",omitempty"`
	Backports    map[string]string   `mapstructure:",omitempty"`
	Fwdports     map[string][]string `mapstructure:",omitempty"`
	Files        []string            `mapstructure:",omitempty"`
}

var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")

// (bp branchPolicies) SrcBranches returns the list of branches for which automatic backport sync is implemented
// i.e. commits landing on these branches will be sync'd to the backport branch
func (bp branchPolicies) SrcBranches() []string {
	keys := make([]string, 0, len(bp.Backports))
	for k := range bp.Backports {
		keys = append(keys, k)
	}
	for k := range bp.Fwdports {
		keys = append(keys, k)
	}
	return keys
}

// (bp branchPolicies) BackportBranch returns the backport branch for srcBranch which will be the source of commits
func (bp branchPolicies) BackportBranch(srcBranch string) (string, error) {
	destBranch, found := bp.Backports[srcBranch]
	if !found {
		return "", ErrUnknownBranch
	}
	return destBranch, nil
}

// (bp branchPolicies) FwdportBranch returns the branches to which commits from srcBranch should be sync'd to
func (bp branchPolicies) FwdportBranch(srcBranch string) ([]string, error) {
	destBranches, found := bp.Fwdports[srcBranch]
	if !found {
		return []string{}, ErrUnknownBranch
	}
	return destBranches, nil
}
