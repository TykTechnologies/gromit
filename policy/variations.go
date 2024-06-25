package policy

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// ghMatrix models the github action matrix structure
// recursion allows it to compactly represent the save state
type ghMatrix struct {
	EnvFiles []struct {
		Cache  string `json:"cache"`
		DB     string `json:"db"`
		Config string `json:"config"`
	}
	Pump  []string            `json:"pump"`
	Sink  []string            `json:"sink"`
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

func (rv repoVariations) Lookup(branch, trigger, testsuite string) *ghMatrix {
	m, found := rv.Leaves[createVariationKey(branch, trigger, testsuite)]
	if !found {
		return nil
	}
	return &m
}

func createVariationKey(branch, trigger, testsuite string) string {
	return fmt.Sprintf("%s-%s-%s", branch, trigger, testsuite)
}

func removeDuplicates(s []string) []string {
	bucket := make(map[string]bool)
	var result []string
	for _, str := range s {
		if _, ok := bucket[str]; !ok {
			bucket[str] = true
			result = append(result, str)
		}
	}
	return result
}

func parseVariations(sv ghMatrix, depth int, rv *repoVariations, path variationPath) {
	for level, levelMatrix := range sv.Level {
		levelMatrix.EnvFiles = append(levelMatrix.EnvFiles, sv.EnvFiles...)
		levelMatrix.Pump = removeDuplicates(append(levelMatrix.Pump, sv.Pump...))
		levelMatrix.Sink = removeDuplicates(append(levelMatrix.Sink, sv.Sink...))
		switch depth {
		case Branch:
			path.Branch = level
		case Trigger:
			path.Trigger = level
		case TestSuite:
			path.Testsuite = level
			key := createVariationKey(path.Branch, path.Trigger, path.Testsuite)
			var tsPath = path // make a copy
			rv.Leaves[key] = levelMatrix
			rv.Paths = append(rv.Paths, tsPath)
		}
		if depth > TestSuite {
			log.Warn().Fields(sv).Msgf("cannot parse test variation levels > %d", TestSuite)
			return
		} else {
			parseVariations(levelMatrix, depth+1, rv, path)
		}
	}
	return
}
