package policy

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

//go:embed templates/*/*
var templates embed.FS

// GenTemplate will render a template bundle from a directory tree rooted at `templates/<bundle>`.
func (r *RepoPolicy) GenTemplate(bundle string) error {
	log.Logger = log.With().Str("bundle", bundle).Interface("repo", r.Name).Logger()
	log.Info().Msg("rendering")
	// Set current timeatamp if not set already
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}

	// Check if the given bundle is valid.
	bundlePath := filepath.Join("templates", bundle)
	_, err := fs.Stat(templates, bundlePath)
	if err != nil {
		return ErrUnKnownBundle
	}
	return r.renderTemplates(bundlePath)
}

// renderTemplates walks a bundle tree and calls renderTemplate for each file
func (r *RepoPolicy) renderTemplates(bundleDir string) error {
	return fs.WalkDir(templates, bundleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("Walk error: (%s): %v ", path, err)
		}
		if d.IsDir() {
			return nil
		}
		return r.renderTemplate(bundleDir, path)
	})
}

// renderTemplate will render one template into its corresponding path in the git tree
// The first two elements of the supplied path will be stripped to remove the templates/<bundle> to derive the
// path that should be written to in the git repo.
func (r *RepoPolicy) renderTemplate(bundleDir, path string) error {
	opFile, err := filepath.Rel(bundleDir, path)
	if err != nil {
		return err
	}

	log.Trace().Str("templatePath", path).Str("outputPath", opFile).Msg("rendering")
	op, err := r.gitRepo.CreateFile(opFile)
	defer op.Close()
	t := template.Must(template.
		New(filepath.Base(path)).
		Option("missingkey=error").
		ParseFS(templates, path))
	log.Trace().Interface("vars", r).Msg("template vars")
	err = t.Execute(op, r)
	if err != nil {
		return err
	}
	log.Debug().Str("path", opFile).Msg("wrote")
	_, err = r.gitRepo.AddFile(opFile)
	if err != nil {
		return err
	}
	return nil
}

func (r *RepoPolicy) renderPR(bundle string) ([]byte, error) {
	prFile := bundle + ".tmpl"
	path := filepath.Join("templates", "prs", prFile)
	log.Trace().Str("PRFilePath", path).Msg("rendering PRs")

	t := template.Must(template.
		New(prFile).
		Option("missingkey=error").
		ParseFS(templates, path))

	rendered := new(bytes.Buffer)
	err := t.Execute(rendered, r)
	if err != nil {
		return []byte{}, err
	}
	log.Debug().Str("tmplpath:", path).Msg("successfully wrote template")
	body, err := ioutil.ReadAll(rendered)
	return body, err
}
