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
	"golang.org/x/exp/maps"
)

type Policy struct {
	Protected []string
	Repos     map[string]repoPolicy // map of reponames to branchPolicies
	Files     []string
	Ports     map[string][]string
}

type repoPolicy struct {
	Files     []string
	Ports     map[string][]string
	Protected []string
}

type prVars struct {
	Files        []string
	RepoName     string
	SrcBranch    string
	DestBranches []string
}

type maVars struct {
	Timestamp string
	MAFiles   []string
	SrcBranch string
}

//type port struct {
//	Src string
//	Dst []string
//}
//

var ErrUnknownRepo = errors.New("repo not present in policies")
var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")

//func (r repoPolicy) SrcBranches() []string {
//	var srcs []string
//	for _, p := range r.Ports {
//		srcs = append(srcs, p.Src)
//	}
//	return srcs
//}
//
//func (r repoPolicy) DstBranches(src string) []string {
//	var dst []string
//	for _, p := range r.Ports {
//		if p.Src == src {
//			dst = p.Dst
//			break
//		}
//	}
//	return dst
//}

func (r repoPolicy) SrcBranches() []string {
	return maps.Keys(r.Ports)
}

func (r repoPolicy) DstBranches(src string) []string {
	if p, ok := r.Ports[src]; ok {
		return p
	}
	return nil
}

// getMAVars returns the template vars required to render the sync-automation template
func (rp Policy) getMAVars(repo string, srcBranch string) (maVars, error) {
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

func (rp Policy) getPRVars(repo string, branch string) (prVars, error) {
	r, err := rp.getCombinedRepoPolicy(repo)
	if err != nil {
		return prVars{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	dstBranches := r.DstBranches(branch)
	return prVars{
		RepoName:     repo,
		Files:        r.Files,
		SrcBranch:    branch,
		DestBranches: dstBranches,
	}, nil

}

// getCombinedRepoPolicy returns a repoPolicy objects with all the common policy
// options merged in.
func (rp Policy) getCombinedRepoPolicy(repo string) (repoPolicy, error) {
	r, found := rp.Repos[repo]
	if !found {
		return repoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	return repoPolicy{
		Files: append(rp.Files, r.Files...),
		Ports: r.Ports,
	}, nil
}

// (rp RepoPolicies) IsProtected tells you if a branch can be pushed directly to origin or needs to go via a PR
func (rp Policy) IsProtected(repo, branch string) (bool, error) {
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
func (rp Policy) SrcBranches(repo string) ([]string, error) {
	r, found := rp.Repos[repo]
	if !found {
		return []string{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	return r.SrcBranches(), nil
}

// DestBranches returns the list of destination branches for a given source branch (where commits originate)
func (rp Policy) DestBranches(repo, branch string) ([]string, error) {
	r, found := rp.Repos[repo]
	if !found {
		return []string{}, fmt.Errorf("repo %s unknown among %v", repo, rp.Repos)
	}
	return r.DstBranches(branch), nil
}

// String representation
func (rp Policy) String() string {
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

func (rp Policy) dotGen(cg *cgraph.Graph) error {

	return nil
}

// (rp RepoPolicies) Graph returns a graphviz dot format representation of the policy
func (rp Policy) Graph(w io.Writer) error {
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
func LoadRepoPolicies(policies *Policy) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
