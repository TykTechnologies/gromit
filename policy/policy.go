package policy

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TykTechnologies/gromit/git"
	"github.com/TykTechnologies/gromit/util"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

var ErrUnknownRepo = errors.New("repo not present in policies")
var ErrUnknownBranch = errors.New("branch not present in branch policies of repo")
var ErrUnKnownBundle = errors.New("bundle not present in loaded policy")

// branchVals contains the parameters that are specific to a particular branch in a repo
type branchVals struct {
	Name           string // Branch name
	GoVersion      string
	Cgo            bool
	ConfigFile     string
	UpgradeFromVer string // Versions to test package upgrades from
}

// Policies models the config file structure. The config file may contain one or more repos.
type Policies struct {
	Description string
	PCRepo      string
	DHRepo      string
	ExposePorts string
	Protected   []string
	Goversion   string
	Master      string              // The equivalent of the master branch
	Repos       map[string]Policies // map of reponames to branchPolicies
	Files       map[string][]string
	Ports       map[string][]string
	Branches    []branchVals
}

// RepoPolicy extracts information from the Policies type for one repo. If you add fields here, the Policies type might have to be updated, and vice versa.
type RepoPolicy struct {
	Name       string
	Protected  []string
	Files      map[string][]string
	Ports      map[string][]string
	gitRepo    *git.GitRepo
	branch     string
	branchvals branchVals
	prefix     string
}

type Template struct {
	fields Fields
}

type Fields map[string]interface{}

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

	found = false
	var b branchVals
	for _, b = range r.Branches {
		if b.Name == branch {
			found = true
			break
		}
	}
	if !found {
		return RepoPolicy{}, fmt.Errorf("branch %s unknown for repo %s", branch, repo)
	}
	return RepoPolicy{
		Name:       repo,
		Protected:  append(p.Protected, r.Protected...),
		Files:      combinedFiles,
		Ports:      r.Ports,
		branch:     branch,
		prefix:     prefix,
		branchvals: b,
	}, nil
}

// Returns the destination branches for a given source branch
func (r RepoPolicy) DestBranches(srcBranch string) ([]string, error) {
	b, found := r.Ports[srcBranch]
	if !found {
		return []string{}, fmt.Errorf("branch %s unknown among %v", srcBranch, r.Ports)
	}
	return b, nil
}

func getSyncTemplate(r *RepoPolicy, bundle string) (*Template, error) {
	t := time.Now().UTC()
	dstBranches, err := r.DestBranches(r.branch)
	if err != nil {
		return nil, err
	}
	var files []string
	// iterate through all bundles, fill in everything except for the
	// sync bundle.
	for b, flist := range r.Files {
		if b == bundle {
			continue
		}
		files = append(files, flist...)
	}
	return &Template{
		fields: Fields{
			"Timestamp":  t.Format(time.UnixDate),
			"SrcBranch":  r.branch,
			"DestBranch": dstBranches,
			"Files":      files,
		},
	}, nil
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
func (r *RepoPolicy) InitGit(depth int, signingKeyid uint64, dir, ghToken string) error {
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

//go:embed templates/*/*
var templates embed.FS

// GenTemplate will render a template bundle from a directory tree rooted at name.
func (r *RepoPolicy) GenTemplate(bundle string) ([]string, error) {
	log.Logger = log.With().Str("bundle", bundle).Interface("repo", r.Name).Logger()
	log.Info().Msg("rendering")
	var fileList []string

	// Check if the given bundle is valid.
	if _, ok := r.Files[bundle]; !ok {
		return fileList, ErrUnKnownBundle
	}
	// bundle to template function mapping.
	tmplFnMap := map[string]func(*RepoPolicy, string) (*Template, error){
		"sync": getSyncTemplate,
	}
	for _, f := range r.Files[bundle] {
		op, err := r.gitRepo.CreateFile(f)
		if err != nil {
			return fileList, err
		}
		defer op.Close()
		// fs.WalkDir(templates, ".", func(path string, d fs.DirEntry, err error) error {
		// 	if err != nil {
		// 		fmt.Println("err: ", err)
		// 	}
		// 	fmt.Println(path)
		// 	return nil
		// })
		t := template.Must(template.
			New(filepath.Base(f)).
			Option("missingkey=error").
			Funcs(template.FuncMap{
				"join":     join,
				"populate": populate,
			}).
			ParseFS(templates, filepath.Join("templates", bundle, f)))
		if err != nil {
			return fileList, err
		}
		log.Trace().Interface("vars", r).Msg("template vars")
		// Call the function corresponding to the given bundle to get the
		// correct Template interface.
		fn := tmplFnMap[bundle]
		tmpl, err := fn(r, bundle)
		if err != nil {
			return fileList, err
		}
		err = t.Execute(op, tmpl.fields)
		if err != nil {
			return fileList, err
		}
		log.Debug().Str("path", f).Msg("wrote")
		hash, err := r.gitRepo.AddFile(f)
		if err != nil {
			return fileList, err
		}
		fileList = append(fileList, f)
		log.Debug().Str("hash", hash.String()).Str("file", f).Msg("added file to worktree")
	}
	return fileList, nil
}

func populate(files []string) string {
	var ret []string
	for _, f := range files {
		f = strings.TrimSuffix(f, "/**")
		f = strings.TrimSuffix(f, "/*")
		f = strings.TrimSuffix(f, "/")
		ret = append(ret, f)
	}
	return strings.Join(ret, " ")
}

func join(files []string) string {
	return strings.Join(files, " ")
}

// Commit commits the current worktree and then displays the resulting change as a patch,
// and returns the hash of the commit object that was committed.
// It will show the changes commited in the form of a patch to stdout and wait for user confirmation.
func (r RepoPolicy) Commit(msg string, confirm bool) (plumbing.Hash, error) {
	origHead, err := r.gitRepo.Head()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting hash for original head: %w", err)
	}
	newCommit, err := r.gitRepo.Commit(msg)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	patch, err := origHead.Patch(newCommit)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("getting diff: %w", err)
	}
	err = patch.Encode(os.Stdout)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encoding diff: %w", err)
	}
	if confirm {
		fmt.Printf("\n----End of diff for %s. Control-C to abort, ⏎/Enter to continue.", r.Name)
		fmt.Scanln()
	}
	return newCommit.Hash, nil
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