package policy

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

type branchPolicies struct {
	Protected    []string            `mapstructure:",omitempty"`
	Deprecations map[string][]string `mapstructure:",omitempty"`
	Backports    map[string]string   `mapstructure:",omitempty"`
	Files        []string            `mapstructure:",omitempty"`
}

type RepoPolicies struct {
	Protected []string
	Repos     map[string]branchPolicies
	Files     []string
}

//go:embed templates
var maTemplates embed.FS

// Gen generates .g/w/sync-automation.yml
func (r *RepoPolicies) Gen(repo, branch, templateDir string) error {
	log.Debug().Str("repo", repo).Str("branch", branch).Str("templateDir", templateDir).Msg("generating meta automation from templateDir")

	opDir := filepath.Join(repo, ".github", "workflows")
	opFile := filepath.Join(opDir, "sync-automation.yml")
	err := os.MkdirAll(opDir, 0755)
	if err != nil {
		return fmt.Errorf("%s: %w", opDir, err)
	}
	op, err := os.Create(opFile)
	if err != nil {
		return fmt.Errorf("%s: %w", opFile, err)
	}
	defer op.Close()

	files := r.Repos[repo].Files
	files = append(files, r.Files...)
	templateVars := struct {
		Timestamp  string
		MAFiles    []string
		SrcBranch  string
		DestBranch string
	}{
		time.Now().UTC().String(),
		files,
		branch,
		r.Repos[repo].Backports[branch],
	}
	t := template.Must(template.New("sync-automation.tmpl").ParseFS(maTemplates, "templates/sync-automation.tmpl"))
	log.Debug().Str("opFile", opFile).Msg("writing meta automation")
	err = t.Execute(op, templateVars)
	return err
}

// String representation
func (r RepoPolicies) String() string {
	w := new(bytes.Buffer)
	fmt.Fprintln(w, `Commits landing on the Source branch are automatically sync'd to the list of Destinations. PRs will be created for the protected branch. Other branches will be updated directly.`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Protected branches: %v\n", r.Protected)
	fmt.Fprintln(w, "Common Files:")
	for _, file := range r.Files {
		fmt.Fprintf(w, " - %s\n", file)
	}
	for repo, pols := range r.Repos {
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
