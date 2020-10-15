package confgen

import (
	"os"
	"path/filepath"
	"text/template"

	rice "github.com/GeertJohan/go.rice"
	"github.com/rs/zerolog/log"
)

// dest is always treated as a directory name
// makeConfigTree() will walk the box, passing files through a template renderer
func makeConfigTree(b *rice.Box, boxPath string, dest string, envName string) error {
	boxFile, err := b.Open(boxPath)
	if err != nil {
		return err
	}
	defer boxFile.Close()
	entries, err := boxFile.Readdir(0)
	if err != nil {
		return err
	}
	os.MkdirAll(dest, 0755)
	log.Trace().Msgf("created dir: %s", dest)

	for _, e := range entries {
		srcPath := filepath.Join(boxPath, e.Name())
		destPath := filepath.Join(dest, e.Name())

		log.Trace().Msgf("Copying %s to %s", srcPath, destPath)

		if e.IsDir() {
			makeConfigTree(b, srcPath, destPath, envName)
		} else {
			// e is a file
			tempStr, err := b.String(srcPath)
			if err != nil {
				log.Error().Err(err).Msgf("could not read as string: %s", srcPath)
				return err
			}
			t := template.Must(template.New(e.Name()).Parse(tempStr))
			f, err := os.Create(destPath)
			if err != nil {
				log.Error().Err(err).Msgf("could not create: %s", destPath)
				return err
			}
			defer f.Close()
			err = t.Execute(f, struct{ EnvName string }{envName})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Must will create a config tree if it does not exist
// Only the root path is checked as a full set of templates will be generated into confDir
func Must(confPath string, envName string) error {
	confDir := filepath.Join(confPath, envName)
	// Does a config dir matching the env name exist?
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		configs := rice.MustFindBox("templates")
		return makeConfigTree(configs, "", confDir, envName)
	}
	return nil
}
