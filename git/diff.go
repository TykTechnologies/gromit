package git

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/waigani/diffparser"
)

func gitDiff(dir string, isPR bool) (string, error) {
	var out bytes.Buffer
	var cmd *exec.Cmd
	if isPR {
		cmd = exec.Command("git", "diff", "-G", "(^[^#])", "--staged")
	} else {
		cmd = exec.Command("git", "diff", "-G", "(^[^#])")
	}
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("diff in %s: %w", dir, err)
	}
	log.Trace().Bytes("output", out.Bytes()).Str("dir", dir).Msg("git diff")
	return out.String(), nil
}

func parseDiff(ds string) ([]string, error) {
	d, err := diffparser.Parse(ds)
	if err != nil {
		return nil, err
	}
	var dFiles []string
	for _, df := range d.Files {
		fname := df.OrigName
		if fname == "" {
			// When a new file is added
			fname = df.NewName
		}
		dFiles = append(dFiles, fname)
	}
	return dFiles, nil
}

func NonTrivial(dir string, isPR bool) ([]string, error) {
	ds, err := gitDiff(dir, isPR)
	if err != nil {
		return nil, err
	}
	return parseDiff(ds)
}
