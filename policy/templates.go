package policy

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
)

// The clunky /*/* is because embed ignores . prefixed dirs like .github
//go:embed templates all:templates
var templates embed.FS

// listBundle prints a directory listing of the embedded bundles
func ListBundles(root string) {
	fs.WalkDir(templates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			fmt.Printf("%s:\n", d.Name())
		} else {
			fmt.Printf("%s\n", d.Name())
		}
		return nil
	})
}

// getTemplate, given a bundle filesystem and a path will return a
// template object with the sub templates parsed into it
func getTemplate(templatePath string) *template.Template {
	templatePaths := []string{templatePath}
	log.Trace().Str("template", templatePath).Msg("top level")
	subTemplates, err := templates.ReadDir(templatePath + ".d")
	if err == nil {
		for _, st := range subTemplates {
			templatePaths = append(templatePaths, st.Name())
		}
		log.Trace().Str("template", templatePath).Strs("subtemplates", templatePaths).Msg("subtemplates")
	}
	return template.Must(
		template.New(filepath.Base(templatePath)).
			Funcs(sprig.TxtFuncMap()).
			Option("missingkey=error").
			ParseFS(templates, templatePaths...))
}

// RenderBundle for each template file that it encounters
// bt.renderTemplates walks a bundle tree and calls renderTemplate for each file
func RenderBundle(bundleDir string, bt BundleVars) error {
	log.Logger = log.With().Str("bundle", bundleDir).Logger()
	log.Debug().Msg("rendering")

	basePath := filepath.Join("templates", bundleDir)
	err := fs.WalkDir(templates, basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("path: %s error: %w", path, err)
		}
		if d.IsDir() {
			if strings.HasSuffix(d.Name(), ".d") {
				return fs.SkipDir
			}
		} else {
			opFile, err := filepath.Rel(basePath, path)
			if err != nil {
				return fmt.Errorf("basePath: %s, path: %s, error: %w", basePath, path, err)
			}
			return bt.renderTemplate(getTemplate(path), opFile)
		}
		return nil
	})

	return err
}

// RepoPolicies.renderTemplate only supports creating output on
// regular filesystems and is not concerned about git repositories
func (rs *RepoPolicies) renderTemplate(t *template.Template, opFile string) error {
	log.Trace().Str("outputPath", opFile).Msg("rendering template")
	op, err := os.Create(opFile)
	if err != nil {
		return err
	}
	defer op.Close()

	return t.Execute(op, rs)
}

// RepoPolicies.renderTemplate will render one template into its corresponding path in the git tree
//  that should be written to in the git repo.
func (r *RepoPolicy) renderTemplate(t *template.Template, opFile string) error {
	// Set current timestamp if not set already
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}
	log.Logger = log.With().Str("repo", r.Name).Logger()
	log.Debug().Msg("rendering")
	op, err := r.gitRepo.CreateFile(opFile)
	if err != nil {
		return err
	}
	defer op.Close()

	return t.Execute(op, r)
}

// renderPR will return the body of a PR
func (r *RepoPolicy) renderPR(bundle string) ([]byte, error) {
	prFile := bundle + ".tmpl"
	path := filepath.Join("templates", "prs", prFile)
	log.Trace().Str("PRFilePath", path).Msg("rendering PRs")
	prContent, err := templates.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Str("bundle", bundle).Msg("failed to open pr file for bundle")
		return []byte{}, err
	}

	t := template.Must(template.
		New(prFile).
		Option("missingkey=error").
		Parse(string(prContent)))

	rendered := new(bytes.Buffer)
	err = t.Execute(rendered, r)
	if err != nil {
		return []byte{}, err
	}
	log.Debug().Str("tmplpath:", path).Msg("successfully wrote template")
	body, err := ioutil.ReadAll(rendered)
	return body, err
}
