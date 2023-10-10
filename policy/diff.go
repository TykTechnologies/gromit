package policy

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/waigani/diffparser"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var LBlue = "\033[94m"
var White = "\033[97m"

func gitDiff(dir string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("git", "diff", "-w", "--ignore-cr-at-eol", "-I^# Generated on:.*$", "--ignore-blank-lines", "HEAD")
	cmd.Dir = dir
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("diff in %s: %w", dir, err)
	}
	log.Trace().Bytes("output", out.Bytes()).Str("dir", dir).Msg("git diff")

	prettyPrint(out.String())
	return out.String(), nil
}

func prettyPrint(out string) {

	red := regexp.MustCompile(`^-[^-]{2}.*`)
	green := regexp.MustCompile(`^\+[^\+]{2}.*`)
	lblue := regexp.MustCompile(`^@@ .*`)
	yellow := regexp.MustCompile(`(^-{3} .*)|(^\+{3} .*)|(^diff --git .*)|(^index [\d|\w]{7,8}..[\d|\w]{7,8}.*)`)

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {

		if green.MatchString(scanner.Text()) {
			fmt.Println(Green + scanner.Text())
		} else if red.MatchString(scanner.Text()) {
			fmt.Println(Red + scanner.Text())
		} else if lblue.MatchString(scanner.Text()) {
			fmt.Println(LBlue + scanner.Text())
		} else if yellow.MatchString(scanner.Text()) {
			fmt.Println(Yellow + scanner.Text())
		} else {
			fmt.Println(Reset + scanner.Text())
		}

	}

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

func NonTrivialDiff(dir string) ([]string, error) {
	ds, err := gitDiff(dir)
	if err != nil {
		return nil, err
	}
	return parseDiff(ds)
}
