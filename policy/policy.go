package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"maps"

	"github.com/TykTechnologies/gromit/util"

	"dario.cat/mergo"
	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// repoConfig contains all the attributes of a repo. Each element here
// _has_ to match the name of the key used in the config yaml. The
// recursive map[string]repoConfig element embeds this type inside
// itself, allowing each repo to override any of the values at upper
// levels
type repoConfig struct {
	Owner               string
	ExposePorts         string
	PackageName         string
	Binary              string
	Buildenv            string
	Builds              buildMap
	BaseImage           string
	DistrolessBaseImage string
	Cgo                 bool
	ConfigFile          string
	VersionPackage      string
	UpgradeFromVer      string
	Tests               []string
	Features            []string
	DeletedFiles        []string
	Branches            map[string]branchVals `copier:"-"`
	Repos               map[string]repoConfig `copier:"-"`
}

// build models the variations in build and their corresponding packages
type build struct {
	Dir              string `yaml:"dir,omitempty"`
	Main             string `yaml:"main,omitempty"`
	Binary           string `yaml:"binary,omitempty"`
	VersionPackage   string `yaml:"versionpackage,omitempty"`
	Flags            []string
	BuildPackageName string
	Description      string
	ImageTitle       string
	PCRepo           string
	UpgradeRepo      string
	DHRepo           string
	CSRepo           string
	CIRepo           string
	Env              []string
	Archs            []struct {
		Docker     string
		Deb        string
		Go         string
		SkipDocker bool
	}
}

type buildMap map[string]*build

// Policies models the config file structure. There are three levels
// at which a particular value can be set: group-level, repo, branch.
// The group level is applicable for all the repos in that group.
// Repeating the same repo in multiple groups is UB
type Policies struct {
	Owner        string
	DeletedFiles []string
	Groups       map[string]repoConfig
}

// branchVals contains only the parameters that can be overriden at
// the branch level. Some elements are overriden, some elements are
// concatenated. See policy.GetRepo to see how definitions are
// processed at each level
type branchVals struct {
	Buildenv            string
	BaseImage           string
	DistrolessBaseImage string
	Cgo                 bool
	ConfigFile          string
	VersionPackage      string
	UpgradeFromVer      string
	Tests               []string
	Features            []string
	Builds              buildMap
	DeletedFiles        []string
}

// RepoPolicy is used to render templates. It provides an abstraction
// between config.yaml and the templates. It is instantiated from
// Policies for a particular repo and branch and the constructor
// implements all the overriding/merging logic between the various
// levels of the Policies type.
type RepoPolicy struct {
	Owner          string
	Name           string
	Default        string
	PackageName    string
	Binary         string
	Builds         buildMap
	Reviewers      []string
	ExposePorts    string
	Cgo            bool
	ConfigFile     string
	VersionPackage string
	UpgradeFromVer string
	Branch         string
	Branchvals     branchVals
	Branches       map[string]branchVals
	Timestamp      string
}

// PushOptions collects the input required to update templates for a
// branch in git and push changes upstream
type PushOptions struct {
	OpDir        string
	Branch       string
	RemoteBranch string
	CommitMsg    string
	Repo         *GitRepo
}

// SetTimestamp Sets the given time as the repopolicy timestamp. If called with zero time
// sets the current time in UTC
func (rp *RepoPolicy) SetTimestamp(ts time.Time) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	rp.Timestamp = ts.Format(time.UnixDate)
}

// GetTimeStamp returns the timestamp currently set for the given repopolicy.
func (rp *RepoPolicy) GetTimeStamp() (time.Time, error) {
	var ts time.Time
	var err error
	ts, err = time.Parse(time.UnixDate, rp.Timestamp)
	return ts, err
}

// SetBranch sets the Branch and Branchvals properties so that templates can simply access them instead of looking them up in the Branches map. This must be called before calling Render()
func (rp *RepoPolicy) SetBranch(branch string) error {
	// First try exact match
	bv, found := rp.Branches[branch]
	if !found {
		// Try case-insensitive match
		lowerBranch := strings.ToLower(branch)
		for k, v := range rp.Branches {
			if strings.ToLower(k) == lowerBranch {
				bv = v
				found = true
				branch = k // Use the actual branch name from config
				break
			}
		}
	}
	if !found {
		return fmt.Errorf("branch %s unknown among %v", branch, rp.Branches)
	}
	rp.Branch = branch
	rp.Branchvals = bv

	return nil
}

// GetAllBranches returns all the branches that are managed for this repo
func (rp *RepoPolicy) GetAllBranches() []string {
	return slices.Sorted(maps.Keys(rp.Branches))
}

// GetRepoPolicy will fetch the RepoPolicy for the supplied repo with
// all overrides (group, repo, branch levels) processed. This is the
// constructor for RepoPolicy.
func (p *Policies) GetRepoPolicy(repo string) (RepoPolicy, error) {
	var group, r repoConfig
	found := false
	for grpName, grp := range p.Groups {
		log.Trace().Msgf("looking in group %s", grpName)
		r, found = grp.Repos[repo]
		if found {
			log.Debug().Msgf("found %s in group %s", repo, grpName)
			group = grp
			break
		}
	}
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown", repo)
	}
	var rp RepoPolicy
	rp.Name = repo
	// Copy policy level elements
	err := copier.CopyWithOption(&rp, &p, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return rp, err
	}
	// Override policy level elements with repo level
	err = copier.CopyWithOption(&rp, &r, copier.Option{IgnoreEmpty: true})
	if err != nil {
		return rp, err
	}
	log.Trace().Interface("rp", rp).Msg("computed repo vals")

	allBranches := make(map[string]branchVals)
	for b, bbv := range r.Branches {
		var rbv branchVals // repo level branchvals
		// copy group-level options
		err := copier.CopyWithOption(&rbv, &group, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// override with repo-level options
		err = copier.CopyWithOption(&rbv, &r, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// override with branch level
		err = copier.CopyWithOption(&rbv, &bbv, copier.Option{IgnoreEmpty: true})
		if err != nil {
			return rp, err
		}
		// builds are merged
		log.Debug().Msgf("Merging builds for %s/%s", rp.Name, b)
		rbv.Builds = mergeBuilds(r.Builds, bbv.Builds)
		// attributes that are unions
		rbv.Features = util.NewSetFromSlices(group.Features, r.Features, bbv.Features).Members()
		rbv.DeletedFiles = util.NewSetFromSlices(p.DeletedFiles, group.DeletedFiles, r.DeletedFiles, bbv.DeletedFiles).Members()

		log.Trace().Interface("bv", rbv).Str("branch", b).Msg("computed branch vals")
		allBranches[b] = rbv
	}
	rp.Branches = allBranches
	return rp, nil
}

// mergeBuilds returns a merged build map from _r_epo and _b_ranch level
func mergeBuilds(r, b buildMap) buildMap {
	merged := make(buildMap)
	maps.Copy(merged, r)
	if err := mergo.Merge(&merged, b, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
		log.Fatal().Interface("dst", merged).Interface("src", b).Msgf("could not merge branch-level build definitions for: %v", err)
	}
	return merged
}

// ProcessBranch will render the templates into a git worktree for the supplied branch, commit and push the changes upstream
// The upstream branch name is the supplied branch name prefixed with releng/ and is returned
func (rp *RepoPolicy) ProcessBranch(pushOpts *PushOptions) error {
	log.Debug().Msgf("processing branch %s", pushOpts.Branch)
	err := pushOpts.Repo.FetchBranch(pushOpts.Branch)
	if err != nil {
		return fmt.Errorf("git checkout %s:%s: %v", pushOpts.Repo.url, pushOpts.Branch, err)
	}
	err = rp.SetBranch(pushOpts.Branch)
	if err != nil {
		return err
	}
	b, err := NewBundle(rp.Branchvals.Features)
	if err != nil {
		return fmt.Errorf("bundle %v: %v", rp.Branchvals.Features, err)
	}
	files, err := b.Render(&rp, pushOpts.OpDir, nil)
	log.Debug().Strs("files", files).Msg("rendered files")
	if err != nil {
		return fmt.Errorf("bundle gen %v: %v", rp.Branchvals.Features, err)
	}
	for _, f := range rp.Branchvals.DeletedFiles {
		fname := filepath.Join(pushOpts.OpDir, f)
		fi, err := os.Stat(fname)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			log.Warn().Err(err).Msgf("stat %s", fname)
		}
		glob := f
		if fi.IsDir() {
			log.Debug().Msgf("recursively deleting %s", fname)
			glob += "/*"
		}
		if err := pushOpts.Repo.RemoveAll(glob); err != nil {
			log.Warn().Err(err).Msgf("removing %s from the index", f)
		}
	}
	// Add rendered files to git staging.
	for _, f := range files {
		_, err := pushOpts.Repo.AddFile(f)
		if err != nil {
			return fmt.Errorf("staging file to git worktree: %v", err)
		}
	}
	err = pushOpts.Repo.Commit(pushOpts.CommitMsg)
	if err != nil {
		return fmt.Errorf("git commit %s: %v", pushOpts.Repo.url, err)
	}

	// Incorporate changes that were pushed outside the templates
	// err = pushOpts.Repo.PullBranch(pushOpts.RemoteBranch)
	// if err != nil && err != git.NoErrAlreadyUpToDate && err != git.ErrBranchNotFound {
	// 	return fmt.Errorf("pulling changes into %s: %v", pushOpts.RemoteBranch, err)
	// }

	err = pushOpts.Repo.Push(pushOpts.RemoteBranch)
	if err != nil {
		return fmt.Errorf("git push %s %s:%s: %v", pushOpts.Repo.url, pushOpts.Repo.Branch(), pushOpts.RemoteBranch, err)
	}
	log.Info().Msgf("pushed %s to %s", pushOpts.RemoteBranch, rp.Name)

	return nil
}

// LoadRepoPolicies populates the supplied policies with the policy key from a the config file
// This will panic if the type assertions fail
func LoadRepoPolicies(policies *Policies) error {
	return viper.UnmarshalKey("policy", policies)
}
