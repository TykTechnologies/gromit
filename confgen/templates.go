// confgen generates a config directory tree for all components
package confgen

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/rs/zerolog/log"
)

//go:embed templates
var confTemplates embed.FS

// dest is always treated as a directory name
// makeConfigTree() will walk the embed.FS, passing files through a template renderer
func makeConfigTree(fs embed.FS, src string, dest string, tVars templateVars) error {
	entries, err := fs.ReadDir(src)
	if err != nil {
		return err
	}
	os.MkdirAll(dest, 0755)
	log.Trace().Msgf("created dir: %s", dest)

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		destPath := filepath.Join(dest, e.Name())

		log.Trace().Msgf("Copying %s to %s", srcPath, destPath)

		if e.IsDir() {
			makeConfigTree(fs, srcPath, destPath, tVars)
		} else {
			// e is a file
			data, err := ioutil.ReadFile(srcPath)
			if err != nil {
				log.Error().Err(err).Str("srcPath", srcPath).Msgf("could not read from embedded file")
				return err
			}
			tempStr := string(data)
			t := template.Must(template.New(e.Name()).Parse(tempStr))
			f, err := os.Create(destPath)
			if err != nil {
				log.Error().Err(err).Msgf("could not create: %s", destPath)
				return err
			}
			defer f.Close()
			err = t.Execute(f, tVars)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getLicense(path string) (string, error) {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(key), nil

}

// templateVars will be interpolated into templates
type templateVars struct {
	EnvName string
}

// Must will create a config tree if it does not exist
// Only the root path is checked as a full set of templates will be generated into confDir
func Must(confPath string, envName string) error {
	confDir := filepath.Join(confPath, envName)
	// Does a config dir matching the env name exist?
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		tVars := templateVars{
			envName,
		}
		return makeConfigTree(confTemplates, "", confDir, tVars)
	}
	return nil
}
