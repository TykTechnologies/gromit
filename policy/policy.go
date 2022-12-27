package policy

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// branchVals contains the parameters that are specific to a particular branch in a repo
type branchVals struct {
	GoVersion      string
	Cgo            bool
	ConfigFile     string
	VersionPackage string                // The package containing version.go
	UpgradeFromVer string                // Versions to test package upgrades from
	PCPrivate      bool                  // indicates whether package cloud repo is private
	Branch         map[string]branchVals `copier:"-"`
	Active         bool
	ReviewCount    string
	Convos         bool
	Tests          []string
	SourceBranch   string
}

// Policies models the config file structure. The config file may contain one or more repos.
type Policies struct {
	Description string
	PCRepo      string
	DHRepo      string
	CSRepo      string
	PackageName string
	Reviewers   []string
	ExposePorts string
	Binary      string
	Protected   []string
	Goversion   string
	Default     string              // The default git branch(master/main/anything else)
	Repos       map[string]Policies // map of reponames to branchPolicies
	Ports       map[string][]string
	Branches    branchVals
}

// RepoPolies aggregates RepoPolicy, indexed by repo name.
type RepoPolicies map[string]RepoPolicy

// BundleVars is an interface that all datatypes that will be passed to a bundle renderer must satisfy
type BundleVars interface {
	renderTemplate(*template.Template, string) error
}

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
// Though Ports can be defined at the global level they are not practically used and if defined will be ignored.
func (p *Policies) GetRepo(repo, prefix, branch string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}

	var b branchVals

	copier.Copy(&b, r.Branches)

	if ib, found := r.Branches.Branch[branch]; found {
		copier.CopyWithOption(&b, &ib, copier.Option{IgnoreEmpty: true})
	}

	// Build release branches map by iterating over each branch values
	releaseBranches := make(map[string]branchVals)

	for branch, releaseBranch := range r.Branches.Branch {
		if releaseBranch.Active {
			var aux branchVals
			copier.Copy(&aux, r.Branches)
			if iaux, found := r.Branches.Branch[branch]; found {
				copier.CopyWithOption(&aux, &iaux, copier.Option{IgnoreEmpty: true})
			}
			releaseBranches[branch] = aux
		}
	}

	return RepoPolicy{
		Name:            repo,
		Protected:       append(p.Protected, r.Protected...),
		Default:         p.Default,
		Ports:           r.Ports,
		Branch:          branch,
		prefix:          prefix,
		Branchvals:      b,
		ReleaseBranches: releaseBranches,
		Reviewers:       r.Reviewers,
		DHRepo:          r.DHRepo,
		PCRepo:          r.PCRepo,
		CSRepo:          r.CSRepo,
		ExposePorts:     r.ExposePorts,
		Binary:          r.Binary,
		Description:     r.Description,
		PackageName:     r.PackageName,
	}, nil
}

// String representation
func (p Policies) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintln(w, `Commits landing on the Source branch are automatically sync'd to the list of Destinations. PRs will be created for the protected branch. Other branches will be updated directly.`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Protected branches: %v\n", p.Protected)
	fmt.Fprintln(w, "Common Files:")
	for repo, pols := range p.Repos {
		fmt.Fprintf(w, "%s\n", repo)
		fmt.Fprintln(w, " Extra files:")
		fmt.Fprintln(w, " Ports")
		for src, dest := range pols.Ports {
			fmt.Fprintf(w, "   - %s → %s\n", src, dest)
		}
	}
	fmt.Fprintln(w)
	return w.String()
}

// LoadRepoPolicies returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
