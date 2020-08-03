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
