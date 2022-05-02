package policy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type RepoPolicies struct {
	Protected []string
	Repos     map[string]RepoPolicies // map of reponames to branchPolicies
	Files     []string
	Ports     map[string][]string
}

type maVars struct {
	Timestamp string
	MAFiles   []string
	SrcBranch string
}

var ErrUnknownRepo = errors.New("repo not present in policies")
var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")

// getMAVars returns the template vars required to render the sync-automation template
func (rp RepoPolicies) getMAVars(repo, srcBranch string) (maVars, error) {
	bps, found := rp.Repos[repo]
	if !found {
		return maVars{}, ErrUnknownRepo
	}
	return maVars{
		Timestamp: time.Now().UTC().String(),
		MAFiles:   append(rp.Files, bps.Files...),
		SrcBranch: srcBranch,
	}, nil
}

type prVars struct {
	Files        []string
	RepoName     string
	SrcBranch    string
	DestBranches []string
	Remove       bool
}

// getMAVars returns the template vars required to render the sync-automation template
func (rp RepoPolicies) getPRVars(repo, branch string, removal bool) (prVars, error) {
	r, found := rp.Repos[repo]
	if !found {
		return prVars{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	_, err := rp.SrcBranches(repo)
	if err != nil {
		return prVars{}, fmt.Errorf("could not get source branches for repo %s: %v", repo, err)
	}
	destBranches, err := rp.DestBranches(repo, branch)
	if err != nil {
		return prVars{}, fmt.Errorf("could not get dest branches for repo %s: %v", repo, err)
	}
	return prVars{
		RepoName:     repo,
		Files:        append(rp.Files, r.Files...),
		SrcBranch:    branch,
		DestBranches: destBranches,
		Remove:       removal,
	}, nil
}

// (rp RepoPolicies) IsProtected tells you if a branch can be pushed directly to origin or needs to go via a PR
func (rp RepoPolicies) IsProtected(repo, branch string) (bool, error) {
	bps, found := rp.Repos[repo]
	if !found {
		return false, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	for _, pb := range append(bps.Protected, rp.Protected...) {
		if pb == branch {
			return true, nil
		}
	}
	return false, nil
}

// (rp RepoPolicies) SrcBranches returns a list of branches that are sources of commits
func (rp RepoPolicies) SrcBranches(repo string) ([]string, error) {
	r, found := rp.Repos[repo]
	if !found {
		return []string{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	ports := make([]string, len(r.Ports))
	i := 0
	for k := range rp.Ports {
		ports[i] = k
		i++
	}
	return ports, nil
}

// DestBranches returns the list of destination branches for a given source branch (where commits originate)
func (rp RepoPolicies) DestBranches(repo, branch string) ([]string, error) {
	_, found := rp.Repos[repo]
	if !found {
		return []string{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	destBranches, found := rp.Repos[repo].Ports[branch]
	if !found {
		return []string{}, fmt.Errorf("branch %s unknown for repo %s", branch, repo)
	}
	return destBranches, nil
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
		fmt.Fprintln(w, " Ports")
		for src, dest := range pols.Ports {
			fmt.Fprintf(w, "   - %s → %s\n", src, dest)
		}
	}
	fmt.Fprintln(w)
	return w.String()
}

func (rp RepoPolicies) dotGen(cg *cgraph.Graph) error {

	return nil
}

// (rp RepoPolicies) Graph returns a graphviz dot format representation of the policy
func (rp RepoPolicies) Graph(w io.Writer) error {
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
func LoadRepoPolicies(policies *RepoPolicies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
