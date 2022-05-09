package policy

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"

	"github.com/TykTechnologies/gromit/git"
	"github.com/TykTechnologies/gromit/util"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

var ErrUnknownRepo = errors.New("repo not present in policies")
var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")

// Policies models the config file structure. The config file may contain one or more repos.
type Policies struct {
	Protected []string
	Repos     map[string]Policies // map of reponames to branchPolicies
	Files     map[string][]string
	Ports     map[string][]string
}

// GetRepo will give you a repoVars type for a repo which can be used to feed templates
// Though Ports can be defined at the global level they are not practically used and if defined will be ignored.
func (p *Policies) GetRepo(repo, prefix, branch string) (RepoPolicy, error) {
	r, found := p.Repos[repo]
	if !found {
		return RepoPolicy{}, fmt.Errorf("repo %s unknown among %v", repo, p.Repos)
	}
	var combinedFiles = make(map[string][]string)
	var temp = make(map[string][]string)
	// Merge two maps, avoiding duplication and merging the values
	for bundle, rFiles := range r.Files {
		if pFiles, found := p.Files[bundle]; found {
			temp[bundle] = append(pFiles, rFiles...)
		} else {
			temp[bundle] = rFiles
		}
	}
	maps.Copy(combinedFiles, p.Files)
	maps.Copy(combinedFiles, temp)
	fmt.Println(combinedFiles, temp)
	return RepoPolicy{
		Name:      repo,
		Protected: append(p.Protected, r.Protected...),
		Files:     combinedFiles,
		Ports:     r.Ports,
		branch:    branch,
		prefix:    prefix,
	}, nil
}

// RepoPolicy extracts information from the Policies type for one repo. If you add fields here, the Policies type might have to be updated, and vice versa.
type RepoPolicy struct {
	Name      string
	Protected []string
	Files     map[string][]string
	Ports     map[string][]string
	gitRepo   *git.GitRepo
	branch    string
	prefix    string
}

// Returns the destination branches for a given source branch
func (r RepoPolicy) DestBranches(srcBranch string) ([]string, error) {
	b, found := r.Ports[srcBranch]
	if !found {
		return []string{}, fmt.Errorf("branch %s unknown among %v", srcBranch, r.Ports)
	}
	return b, nil
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

// InitGit initialises the corresponding git repo by fetching it
func (r RepoPolicy) InitGit(depth int, signingKeyid uint64, dir, ghToken string) error {
	log.Logger = log.With().Str("repo", r.Name).Str("branch", r.branch).Logger()
	fqdnRepo := fmt.Sprintf("%s/%s", r.prefix, r.Name)

	var err error
	r.gitRepo, err = git.FetchRepo(fqdnRepo, dir, ghToken, depth)
	if err != nil {
		return err
	}
	if signingKeyid != 0 {
		signer, err := util.GetSigningEntity(signingKeyid)
		if err != nil {
			return err
		}
		err = r.gitRepo.EnableSigning(signer)
		if err != nil {
			log.Warn().Err(err).Msg("commits will not be signed")
		}
	}
	return nil
}

//go:embed templates
var templates embed.FS

// GenTemplate will render a template bundle from a directory tree rooted at name.
func (r *RepoPolicy) GenTemplate(bundle, commitMsg string) error {
	log.Logger = log.With().Str("bundle", bundle).Interface("repo", r.Name).Logger()
	log.Info().Msg("rendering")

	for _, f := range r.Files[bundle] {
		op, err := r.gitRepo.CreateFile(f)
		if err != nil {
			return err
		}
		defer op.Close()

		t := template.Must(template.
			New(bundle).
			Option("missingkey=error").
			ParseFS(templates, f))
		if err != nil {
			return err
		}
		log.Trace().Interface("vars", r).Msg("template vars")
		err = t.Execute(op, r)

		log.Debug().Str("path", f).Msg("wrote")
		hash, err := r.gitRepo.AddFile(f, commitMsg, true)
		log.Debug().Str("hash", hash.String()).Str("path", f).Msg("committed")
	}
	return nil
}

// Push will push the current state of the repo to github
// If the branch is protected, it will be pushed to a branch prefixed with releng/
func (r RepoPolicy) Push() (string, error) {
	var remoteBranch string
	if r.IsProtected(r.branch) {
		remoteBranch = fmt.Sprintf("releng/%s", r.branch)
	} else {
		remoteBranch = r.branch
	}
	return remoteBranch, r.gitRepo.Push(r.branch, remoteBranch)
}

func (r *RepoPolicy) CreatePR(bundle, title, remoteBranch string, dryRun bool) error {
	if r.branch == "" {
		return fmt.Errorf("unknown local branch on repo %s when creating PR", r.Name)
	}
	t := template.Must(template.
		New(bundle).
		Option("missingkey=error").
		ParseFS(templates, fmt.Sprintf("pr-templates/%s/pr.tmpl", bundle)))
	var b bytes.Buffer
	err := t.Execute(&b, r)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	// prOpts := &github.NewPullRequest{
	// 	Title: github.String(title),
	// 	Head:  github.String(remoteBranch),
	// 	Base:  github.String(r.branch),
	// 	Body:  github.String(b.String()),
	// }
	// if dryRun {
	// 	log.Warn().Msg("only dry-run, not really creating PR")
	// } else {
	// 	pr, _, err := r.gitRepo.gh.PullRequests.Create(context.Background(), ghOrg, r.Name, prOpts)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	r.prs = append(r.prs, pr.GetHTMLURL())
	// }
	return nil
}

// String representation
func (rp Policies) String() string {
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

func (rp Policies) dotGen(cg *cgraph.Graph) error {
	return nil
}

// (rp RepoPolicies) Graph returns a graphviz dot format representation of the policy
func (rp Policies) Graph(w io.Writer) error {
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
func LoadRepoPolicies(policies *Policies) error {
	log.Info().Msg("loading repo policies")
	return viper.UnmarshalKey("policy", policies)
}
