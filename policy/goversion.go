package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// The Go version of a repository belongs to the repository, not to gromit. Upgrading Go is a change to
// the repository's own files, and gromit must neither make it nor undo it. So when a branch is rendered
// into a checkout, the golang-cross tag that the checkout already has wins over buildenv: buildenv in the
// config is only the value for a checkout that has none yet.
//
// The tag is read from the files gromit itself renders it into, first match wins:
//
//	.github/workflows/release.yml   the first entry of the golang_cross matrix
//	ci/gromit.env                   GO_IMAGE=tykio/golang-cross:<tag>
var (
	releaseTagRe = regexp.MustCompile(`(?m)^[ \t]*golang_cross:[ \t]*\n[ \t]*-[ \t]*['"]?([0-9]+\.[0-9]+-[a-z0-9]+)`)
	envTagRe     = regexp.MustCompile(`(?m)^GO_IMAGE=tykio/golang-cross:([0-9]+\.[0-9]+-[a-z0-9]+)[ \t]*$`)

	// where the tag lives in a checkout, and how to find it there
	goTagSources = []struct {
		file string
		re   *regexp.Regexp
	}{
		{filepath.Join(".github", "workflows", "release.yml"), releaseTagRe},
		{filepath.Join("ci", "gromit.env"), envTagRe},
	}
)

// AdoptBuildenv sets Branchvals.Buildenv to the golang-cross tag of the checkout in dir, when it has one.
// Without one it keeps buildenv from the config, and it is an error when there is none of either. A dir of
// "-" means rendering to stdout: there is no checkout, so the config value is used.
func (rp *RepoPolicy) AdoptBuildenv(dir string) error {
	if dir != "-" {
		for _, src := range goTagSources {
			data, err := os.ReadFile(filepath.Join(dir, src.file))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("reading the golang-cross tag of %s: %w", rp.Name, err)
			}
			if m := src.re.FindSubmatch(data); m != nil {
				rp.Branchvals.Buildenv = string(m[1])
				return nil
			}
		}
	}
	if rp.Branchvals.Buildenv == "" {
		return fmt.Errorf("%s/%s: no golang-cross tag: the checkout in %s has none in %s, and the config has no buildenv for a new repository or branch",
			rp.Name, rp.Branch, dir, goTagFiles())
	}
	return nil
}

func goTagFiles() string {
	var s string
	for i, src := range goTagSources {
		if i > 0 {
			s += " or "
		}
		s += filepath.ToSlash(src.file)
	}
	return s
}
