package policy

import (
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// GetRepoPolicy will fetch the RepoPolicy with all overrides processed
func GetRepoPolicy(repo string, branch string) (RepoPolicy, error) {
	var configPolicies Policies
	err := LoadRepoPolicies(&configPolicies)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse repo policies")
	}
	return configPolicies.GetRepo(repo, viper.GetString("prefix"), branch)
}

// branchVals contains the parameters that are specific to a particular branch in a repo
type branchVals struct {
	GoVersion      string
	Cgo            bool
	ConfigFile     string
	VersionPackage string                // The package containing version.go
	UpgradeFromVer string                // Versions to test package upgrades from
	PCPrivate      bool                  // indicates whether package cloud repo is private
	Branch         map[string]branchVals `copier:"-"`
	// RelengVersion specifies which version of releng bundle to choose for
	// this branch. The conditions for which version to choose where, is always
	// within the templates.
	RelengVersion string
	Active        bool
	ReviewCount   string
	Convos        bool
	Tests         []string
	SourceBranch  string
}

// Policies models the config file structure. The config file may contain one or more repos.
type Policies struct {
	Description         string
	PCRepo              string
	DHRepo              string
	CSRepo              string
	PackageName         string
	Reviewers           []string
	ExposePorts         string
	Binary              string
	Goversion           string
	Default             string              // The default git branch(master/main/anything else)
	Repos               map[string]Policies // map of reponames to branchPolicies
	Ports               map[string][]string
	Branches            branchVals
	Wiki                bool
	Topics              []string `copier:"-"`
	VulnerabilityAlerts bool
	SquashMsg           string
	SquashTitle         string
	Visibility          string
}

// RepoPolicies aggregates RepoPolicy, indexed by repo name.
type RepoPolicies map[string]RepoPolicy

// GetAllRepos returns a map of reponame->repopolicy for all the
// repos in the policy config.
func (p *Policies) GetAllRepos(prefix string) (RepoPolicies, error) {
	var rp RepoPolicies
	for repoName, repoVals := range p.Repos {
		repo, err := repoVals.GetRepo(repoName, prefix, "master")
		if err != nil {
			return RepoPolicies{}, err
		}
		rp[repoName] = repo
	}

	return rp, nil
}

func (b *branchVals) getRelengVersion(r Policies, repo string) (string, error) {
	// Update inner branch with the correct releng version.
	// The precedence is: explicit releng version >> source branch version >> common branch version
	if b.RelengVersion == "" && b.SourceBranch != "" {
		var sb branchVals
		for sbName := b.SourceBranch; sbName != ""; sbName = sb.SourceBranch {
			var exists bool
			if sb, exists = r.Branches.Branch[sbName]; !exists {
				return "", fmt.Errorf("policy error: source branch: %s, for repo: %s doesn't exist", b.SourceBranch, repo)
			}
			if sb.RelengVersion == "" && sb.SourceBranch == "" {
				b.RelengVersion = r.Branches.RelengVersion
				break
			}
			if sb.RelengVersion != "" {
				b.RelengVersion = sb.RelengVersion
			}
		}

	}
	return b.RelengVersion, nil
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
	// Override policy values
	copier.CopyWithOption(&p, &r, copier.Option{IgnoreEmpty: true})

	// Check if the branch has a branch specific policy in the config and override the
	// common branch values with the branch specific ones.
	if ib, found := r.Branches.Branch[branch]; found {
		relengVer, err := ib.getRelengVersion(r, repo)
		if err != nil {
			return RepoPolicy{}, err
		}
		log.Debug().Str("Releng version", relengVer).Msg("parsed releng version to use.")
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
		Name:                repo,
		Default:             p.Default,
		Ports:               r.Ports,
		Branch:              branch,
		prefix:              prefix,
		Branchvals:          b,
		ReleaseBranches:     releaseBranches,
		Reviewers:           r.Reviewers,
		DHRepo:              r.DHRepo,
		PCRepo:              r.PCRepo,
		CSRepo:              r.CSRepo,
		ExposePorts:         r.ExposePorts,
		Binary:              r.Binary,
		Description:         r.Description,
		PackageName:         r.PackageName,
		Topics:              append(p.Topics, r.Topics...),
		VulnerabilityAlerts: p.VulnerabilityAlerts,
		SquashMsg:           p.SquashMsg,
		SquashTitle:         p.SquashTitle,
		Wiki:                p.Wiki,
		Visibility:          p.Visibility,
	}, nil
}

// LoadRepoPolicies returns the policies as a map of repos to policies
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
