package policy

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/rs/zerolog/log"
)

// GenTemplate will render a template bundle from a directory tree rooted at `templates/<bundle>`.
func (r *RepoPolicy) GenTemplateTf(bundle string) error {
	log.Logger = log.With().Str("bundle", bundle).Interface("repo", r.Name).Logger()
	log.Info().Msg("rendering")
	// Set current timestamp if not set already
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}

	// Check if the given bundle is valid.
	bundlePath := filepath.Join("templates", bundle)
	_, err := fs.Stat(templates, bundlePath)
	if err != nil {
		return ErrUnKnownBundle
	}
	return r.renderTemplatesTf(bundlePath)
}

func (r *RepoPolicy) renderTemplatesTf(bundleDir string) error {
	return fs.WalkDir(templates, bundleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("Walk error: (%s): %v ", path, err)
		}
		if d.IsDir() {
			if strings.HasSuffix(path, ".d") {
				err := r.renderTemplateTf(bundleDir, path, true)
				if err != nil {
					return err
				}
				// Skip directory entirely if ".d" directory
				return fs.SkipDir
			}
			return nil
		}
		// check if <file>.d exists, if exists, then already parsed/ will be parsed
		// as part of the dir parse call.
		_, statErr := fs.Stat(templates, path+".d")
		if statErr == nil {
			log.Info().Str("dir_path", path).Msg(".d directory exists, so not rendering independently")
			return nil
		}
		return r.renderTemplateTf(bundleDir, path, false)
	})
}

// CreateFile will create a file in a directory, truncating it if it already exists with the embedded git worktree.
// Any intermediate directories are also created.
func (r *RepoPolicy) CreateFile(path string) (billy.File, error) {
	var fs billy.Filesystem
	fs = memfs.New()
	log.Debug().Msg("Creating File now")
	op, err := fs.Create(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return op, nil
}

// renderTemplate will render one template into its corresponding path in the git tree
// The first two elements of the supplied path will be stripped to remove the templates/<bundle> to derive the
// path that should be written to in the git repo.
func (r *RepoPolicy) renderTemplateTf(bundleDir, path string, isDir bool) error {

	var parsePaths []string
	if isDir {
		dir := path
		path = strings.TrimSuffix(path, ".d")
		parsePaths = append(parsePaths, dir+"/**", path)
		log.Info().Strs("parsePaths", parsePaths).Msg(".d exists, so parsing the file as well as dir contents")
	} else {
		parsePaths = append(parsePaths, path)
	}
	opFile, err := filepath.Rel(bundleDir, path)
	if err != nil {
		return err
	}

	log.Trace().Str("templatePath", path).Str("outputPath", opFile).Msg("rendering")

	op, err := r.CreateFile(opFile)
	defer op.Close()
	// op, err := r.gitRepo.CreateFile(opFile)
	// defer op.Close()
	t := template.Must(template.
		New(filepath.Base(path)).
		Funcs(sprig.FuncMap()).
		Option("missingkey=error").
		ParseFS(templates, parsePaths...))
	log.Trace().Interface("vars", r).Msg("template vars")
	err = t.Execute(op, r)
	if err != nil {
		return err
	}
	log.Debug().Str("path", opFile).Msg("wrote")

	// _, err = r.gitRepo.AddFile(opFile)
	// if err != nil {
	// 	return err
	// }

	return nil
}
