package git

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/waigani/diffparser"
)

func gitDiff(dir string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "diff", "-G", "(^[^#])")
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
		dFiles = append(dFiles, df.OrigName)
	}
	return dFiles, nil
}

func NonTrivial(dir string) ([]string, error) {
	ds, err := gitDiff(dir)
	if err != nil {
		return nil, err
	}
	return parseDiff(ds)
}