package policy

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
)

//go:embed templates/*/*
var templates embed.FS

// GenTemplate will render a template bundle from a directory tree rooted at `templates/<bundle>`.
func (r *RepoPolicy) GenTemplate(bundle string) error {
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
	return r.renderTemplates(bundlePath)
}

// GenTerraformPolicyTemplate generates the terraform policy file
// from the given template file.
func (r *RepoPolicy) GenGpacPolicyTemplate(src string, dst string, fileName string) error {

	opFile := dst + fileName
	op, err := os.Create(opFile)
	if err != nil {
		return err
	}
	defer op.Close()

	t := template.Must(template.
		New(filepath.Base(src + fileName)).
		Funcs(sprig.FuncMap()).
		Option("missingkey=error").
		ParseFiles(src + fileName),
	)
	log.Debug().Interface("repo policy", r).Str("tmpl", src+fileName).Str("output", opFile).Msg("rendering terraform tmpl")
	// Set current timestamp if not set already
	if r.Timestamp == "" {
		r.SetTimestamp(time.Time{})
	}
	err = t.Execute(op, r)
	if err != nil {
		return err
	}
	log.Debug().Msg("templates rendered successfully")
	return nil
}

func CopyGpacStaticFiles(src string, dst string) error {

	return fs.WalkDir(templates, src, func(path string, d fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			log.Err(errWalk).Msgf("Walk error: (%s)", path)
			return errWalk
		}

		// Ignore templatized .tfvars files
		if filepath.Ext(path) == ".tfvars" {
			return nil
		}

		opFile, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		inputFile := filepath.Join(dst, opFile)

		if d.IsDir() {
			os.Mkdir(inputFile, os.ModePerm)
			return nil
		}

		fin, err := templates.Open(path)

		if err != nil {
			log.Error().Err(err).Msgf("Error while opening %s", path)
		}
		defer fin.Close()

		fout, err := os.Create(inputFile)
		if err != nil {
			log.Error().Err(err).Msgf("Error while Create %s", inputFile)
		}
		defer fout.Close()

		// Copy file to final destination
		_, err = io.Copy(fout, fin)

		if err != nil {
			log.Error().Err(err).Msg("Error while copying file")
		}
		return nil
	})

}

// renderTemplates walks a bundle tree and calls renderTemplate for each file
func (r *RepoPolicy) renderTemplates(bundleDir string) error {
	return fs.WalkDir(templates, bundleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("Walk error: (%s): %v ", path, err)
		}
		if d.IsDir() {
			if strings.HasSuffix(path, ".d") {
				err := r.renderTemplate(bundleDir, path, true)
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
		return r.renderTemplate(bundleDir, path, false)
	})
}

// renderTemplate will render one template into its corresponding path in the git tree
// The first two elements of the supplied path will be stripped to remove the templates/<bundle> to derive the
// path that should be written to in the git repo.
func (r *RepoPolicy) renderTemplate(bundleDir, path string, isDir bool) error {

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

	op, err := r.gitRepo.CreateFile(opFile)
	defer op.Close()
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
