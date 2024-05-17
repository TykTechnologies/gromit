package policy

import (
	"fmt"
	"os"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// ghMatrix models the github action matrix structure
// recursion allows it to compactly represent the save state
type ghMatrix struct {
	EnvFiles []struct {
		Cache  string
		DB     string
		Config string
	}
	Pump  []string
	Sink  []string
	Level map[string]ghMatrix `copier:"-"` // assumption: copies are _never_ recursive
}

// Tree depth of ghMatrix
const (
	Branch    = 0
	Trigger   = 1
	TestSuite = 2
)

// repoVariations models the matrix for a repo along with some fields
// to ease template rendering. The ghMatrix used here is not recursive.
type repoVariations struct {
	Leaves map[string]ghMatrix
	Paths  []variationPath
}

// TestsuiteVariations maps savedVariations to a form suitable for runtime use
type TestsuiteVariations map[string]repoVariations

type variationPath struct {
	Branch, Trigger, Testsuite string
}

func (rv repoVariations) Branches() []string {
	var rvals []string
	for _, path := range rv.Paths {
		rvals = append(rvals, path.Branch)
	}
	return newSetFromSlices(rvals).Members()
}

func (rv repoVariations) Triggers(branch string) []string {
	var rvals []string
	for _, path := range rv.Paths {
		if path.Branch == branch {
			rvals = append(rvals, path.Trigger)
		}
	}
	return newSetFromSlices(rvals).Members()
}

func (rv repoVariations) Testsuites(branch, trigger string) []string {
	var rvals []string
	for _, path := range rv.Paths {
		if path.Branch == branch && path.Trigger == trigger {
			rvals = append(rvals, path.Testsuite)
		}
	}
	return newSetFromSlices(rvals).Members()
}

func parseVariations(sv ghMatrix, depth int, rv *repoVariations, path variationPath) {
	for level, levelMatrix := range sv.Level {
		levelMatrix.EnvFiles = append(levelMatrix.EnvFiles, sv.EnvFiles...)
		levelMatrix.Pump = append(levelMatrix.Pump, sv.Pump...)
		levelMatrix.Sink = append(levelMatrix.Sink, sv.Sink...)
		switch depth {
		case Branch:
			path.Branch = level
		case Trigger:
			path.Trigger = level
		case TestSuite:
			path.Testsuite = level
			key := fmt.Sprintf("%s-%s-%s", path.Branch, path.Trigger, path.Testsuite)
			var tsPath = path // make a copy
			rv.Leaves[key] = levelMatrix
			rv.Paths = append(rv.Paths, tsPath)
		}
		if depth > TestSuite {
			log.Fatal().Fields(sv).Msgf("cannot parse test variation levels > %d", TestSuite)
		} else {
			parseVariations(levelMatrix, depth+1, rv, path)
		}
	}
	return
}

// loadVariations returns the persisted test variations from disk
func loadVariations(fname string) (TestsuiteVariations, error) {
	data, err := os.ReadFile(fname)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not read %s", fname)
	}
	var saved ghMatrix
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return nil, err
	}
	// top level variations
	var global ghMatrix
	err = copier.CopyWithOption(&global, &saved, copier.Option{IgnoreEmpty: true})
	if err != nil {
		log.Warn().Err(err).Msgf("could not copy global variations from %s", fname)
	}
	tv := make(TestsuiteVariations)
	for repo, matrix := range saved.Level {
		var rv repoVariations
		var vp variationPath
		rv.Leaves = make(map[string]ghMatrix)
		// apply defaults to every repo
		matrix.EnvFiles = append(matrix.EnvFiles, global.EnvFiles...)
		matrix.Pump = append(matrix.Pump, global.Pump...)
		matrix.Sink = append(matrix.Sink, global.Sink...)

		parseVariations(matrix, 0, &rv, vp)
		tv[repo] = rv
	}
	return tv, err
}
