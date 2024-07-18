package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jinzhu/copier"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// ghMatrix models the github action matrix structure
// recursion allows it to compactly represent the save state
type ghMatrix struct {
	EnvFiles []struct {
		Cache      string `json:"cache"`
		DB         string `json:"db"`
		Config     string `json:"config"`
		APIMarkers string `json:"apimarkers"`
		UIMarkers  string `json:"uimarkers"`
	} `json:"envfiles"`
	Pump    []string `json:"pump"`
	Sink    []string `json:"sink"`
	Distros struct {
		Deb []string `json:"deb"`
		Rpm []string `json:"rpm"`
	} `json:"distros"`
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

// RepoTestsuiteVariations maps savedVariations to a form suitable for runtime use
type RepoTestsuiteVariations map[string]repoVariations
type AllTestsuiteVariations map[string]RepoTestsuiteVariations

func (av AllTestsuiteVariations) Files() []string {
	keys := make([]string, 0, len(av))
	for k := range av {
		keys = append(keys, k)
	}
	return keys
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
		m = rv.Leaves[createVariationKey("master", trigger, testsuite)]
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
		levelMatrix.Distros.Deb = removeDuplicates(append(levelMatrix.Distros.Deb, sv.Distros.Deb...))
		levelMatrix.Distros.Rpm = removeDuplicates(append(levelMatrix.Distros.Rpm, sv.Distros.Rpm...))
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

// loadAllVariations loads all yaml files in tvDir returning it in a
// map indexed by filename
func loadAllVariations(tvDir string) (AllTestsuiteVariations, error) {
	files, err := os.ReadDir(tvDir)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read directory")
	}

	numVariations := 0
	av := make(AllTestsuiteVariations)
	for _, file := range files {
		fname := file.Name()
		yaml, _ := regexp.MatchString("\\.ya?ml$", fname)
		if !yaml {
			continue
		}
		pathName := filepath.Join(tvDir, fname)
		tv, err := loadVariation(pathName)
		if err != nil {
			log.Warn().Err(err).Msgf("could not load test variation from %s", pathName)
		}
		av[fname] = tv
		numVariations++
	}
	if numVariations < 1 {
		return av, fmt.Errorf("No loadable files in %s", tvDir)
	}
	return av, nil
}

// loadVariation unrolls the compact saved representation from a file
// it also sets up handlers for the loaded variations
func loadVariation(tvFile string) (RepoTestsuiteVariations, error) {
	data, err := os.ReadFile(tvFile)
	if err != nil {
		return nil, err
	}
	var saved ghMatrix
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal data from %s: %s: %w", tvFile, string(data), err)
	}
	// top level variations
	var global ghMatrix
	err = copier.CopyWithOption(&global, &saved, copier.Option{IgnoreEmpty: true})
	if err != nil {
		log.Warn().Err(err).Msgf("could not copy global variations")
	}
	tv := make(RepoTestsuiteVariations)
	for repo, matrix := range saved.Level {
		var rv repoVariations
		var vp variationPath
		rv.Leaves = make(map[string]ghMatrix)
		// apply defaults to every repo
		matrix.EnvFiles = append(matrix.EnvFiles, global.EnvFiles...)
		matrix.Pump = removeDuplicates(append(matrix.Pump, global.Pump...))
		matrix.Sink = removeDuplicates(append(matrix.Sink, global.Sink...))
		matrix.Distros.Deb = removeDuplicates(append(matrix.Distros.Deb, global.Distros.Deb...))
		matrix.Distros.Rpm = removeDuplicates(append(matrix.Distros.Rpm, global.Distros.Rpm...))

		parseVariations(matrix, 0, &rv, vp)
		tv[repo] = rv
	}
	log.Debug().Interface("tv", tv).Msgf("loaded from %s", tvFile)
	return tv, nil
}
