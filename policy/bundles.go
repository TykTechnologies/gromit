package policy

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

//go:embed templates/*/*
var templates embed.FS

// renderTemplates walks a bundle tree and calls renderTemplate for each path
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

	op, err := r.gitRepo.CreateFile(opFile)
	defer op.Close()
	t := template.Must(template.
		New(filepath.Base(path)).
		Option("missingkey=error").
		ParseFS(templates, path))
	if err != nil {
		return err
	}
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
