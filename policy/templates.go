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
//
//go:embed templates all:templates
var templates embed.FS

// listBundle prints a directory listing of the embedded bundles
func ListBundles(root string) {
	if root != "." {
		root = filepath.Join("templates", root)
	}
	fs.WalkDir(templates, root, func(path string, d fs.DirEntry, err error) error {
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

// getTemplate, given a top level template path will return a template
// object with the sub templates parsed into it
func getTemplate(templatePath string) *template.Template {
	templatePaths := []string{templatePath}
	log.Trace().Str("template", templatePath).Msg("top level")
	dsubDir := filepath.Join(templatePath + ".d")
	subTemplates, err := templates.ReadDir(dsubDir)
	if err == nil {
		for _, st := range subTemplates {
			templatePaths = append(templatePaths, filepath.Join(dsubDir, st.Name()))
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
func RenderBundle(bundleDir, opDir string, bt BundleVars) error {
	log.Logger = log.With().Str("bundle", bundleDir).Logger()

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
			return bt.renderTemplate(getTemplate(path), filepath.Join(opDir, opFile))
		}
		return nil
	})

	return err
}

// RepoPolicies.renderTemplate only supports creating output on
// regular filesystems and is not concerned about git repositories
func (rs *RepoPolicies) renderTemplate(t *template.Template, opFile string) error {
	log.Trace().Str("outputPath", opFile).Msg("rendering RepoPolicies template")
	op, err := os.Create(opFile)
	if err != nil {
		return err
	}
	defer op.Close()

	return t.Execute(op, rs)
}

// RepoPolicies.renderTemplate will render one template into its corresponding path
func (r *RepoPolicy) renderTemplate(t *template.Template, opFile string) error {
	log.Trace().Str("outputPath", opFile).Msg("rendering RepoPolicy template")
	// Set current timestamp if not set already
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}
	dir, _ := filepath.Split(opFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	op, err := os.Create(opFile)
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
