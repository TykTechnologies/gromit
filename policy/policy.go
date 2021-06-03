package policy

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type RepoPolicies struct {
	Protected []string
	Repos     map[string]branchPolicies // map of reponames to branchPolicies
	Files     []string
}

type maVars struct {
	Timestamp string
	MAFiles   []string
	SrcBranch string
	Backport  string
	Fwdports  []string
}

var ErrUnknownRepo = errors.New("repo not present in policies")

// getMAVars returns the template vars required to render the sync-automation template
func (rp RepoPolicies) getMAVars(repo, srcBranch string) (maVars, error) {
	bps, found := rp.Repos[repo]
	if !found {
		return maVars{}, ErrUnknownRepo
	}
	var ma = maVars{
		Timestamp: time.Now().UTC().String(),
		MAFiles:   append(rp.Files, bps.Files...),
		SrcBranch: srcBranch,
	}
	destBranch, err := bps.BackportBranch(srcBranch)
	if err == ErrUnknownBranch {
		log.Debug().Msg("no backports")
	} else {
		ma.Backport = destBranch
	}

	destBranches, err := bps.FwdportBranch(srcBranch)
	if err == ErrUnknownBranch {
		log.Debug().Msg("no fwdports")
	} else {
		ma.Fwdports = destBranches
	}

	return ma, nil
}

type prVars struct {
	Files     []string
	RepoName  string
	Backports map[string]string
	Fwdports  map[string]string
	Branch    string
	Remove    bool
}

// getMAVars returns the template vars required to render the sync-automation template
func (rp RepoPolicies) getPRVars(repo, branch string, removal bool) (prVars, error) {
	bps, found := rp.Repos[repo]
	if !found {
		return prVars{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	return prVars{
		RepoName:  repo,
		Files:     append(rp.Files, bps.Files...),
		Backports: bps.Backports,
		Branch:    branch,
		Remove:    removal,
	}, nil
}

// (rp RepoPolicies) SrcBranches returns a list of branches that are sources of commits
func (rp RepoPolicies) SrcBranches(repo string) ([]string, error) {
	bps, found := rp.Repos[repo]
	if !found {
		return []string{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	return bps.SrcBranches(), nil
}

// String representation
func (rp RepoPolicies) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintln(w, `Commits landing on the Source branch are automatically sync'd to the list of Destinations. PRs will be created for the protected branch. Other branches will be updated directly.`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Protected branches: %v\n", rp.Protected)
	fmt.Fprintln(w, "Common Files:")
	for _, file := range rp.Files {
		fmt.Fprintf(w, " - %s\n", file)
	}
	for repo, pols := range rp.Repos {
		fmt.Fprintf(w, "%s\n", repo)
		fmt.Fprintln(w, " Extra files:")
		for _, f := range pols.Files {
			fmt.Fprintf(w, "   - %s\n", f)
		}
		fmt.Fprintln(w, " Deprecations:")
		for version, files := range pols.Deprecations {
			fmt.Fprintf(w, "  Version %s\n", version)
			for _, f := range files {
				fmt.Fprintf(w, "   - %s\n", f)
			}
		}
		fmt.Fprintln(w, " Backports")
		for src, dest := range pols.Backports {
			fmt.Fprintf(w, "   - %s → %s\n", src, dest)
		}
	}
	fmt.Fprintln(w)
	return w.String()
}

// GetPolicyConfig returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *RepoPolicies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
