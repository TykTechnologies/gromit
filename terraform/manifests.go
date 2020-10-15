package terraform

import (
	"os"
	"path/filepath"

	"io/ioutil"

	rice "github.com/GeertJohan/go.rice"
	"github.com/rs/zerolog/log"
)

// dest is always treated as a directory name
// copyBoxToDir() will skip a .terraform dir, if found
func copyBoxToDir(b *rice.Box, boxPath string, dest string) error {
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

	for _, e := range entries {
		srcPath := filepath.Join(boxPath, e.Name())
		destPath := filepath.Join(dest, e.Name())

		log.Trace().Msgf("Copying %s to %s", srcPath, destPath)

		if e.IsDir() {
			// Recursively call copyDir()
			if e.Name() == ".terraform" || e.Name() == "terraform.tfstate.d" {
				log.Debug().Msg("skipping terraform dir")
				continue
			}
			copyBoxToDir(b, srcPath, destPath)
		} else {
			// e is a file
			err = ioutil.WriteFile(destPath, b.MustBytes(srcPath), 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// deployManifests to a temporary dir prefixed with destPrefix
func deployManifest(b *rice.Box, destPrefix string) (string, error) {
	tmpDir, err := ioutil.TempDir("", destPrefix)
	if err != nil {
		return "", err
	}

	err = copyBoxToDir(b, "", tmpDir)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not restore embedded manifests to %s", tmpDir)
	}
	return tmpDir, nil
}
