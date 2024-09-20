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
	Level map[string]ghMatrix `copier:"-"` // map testsuite→ghMatrix
}

// Tree depth of ghMatrix
const (
	Testsuite = 0
	Branch    = 1
	Trigger   = 2
	Repo      = 3
)

// variations flattens the saved matrix form so that it can be looked up in any order
// The template are in repo→branch→trigger→testsuite
// API calls use testsuite→branch→trigger→repo
// The ghMatrix used here is not recursive.
type variations struct {
	Leaves map[string]ghMatrix
	Paths  []variationPath
}

type variationPath struct {
	Testsuite, Trigger, Branch, Repo string
}

func NewVariations() *variations {
	var v variations
	v.Leaves = make(map[string]ghMatrix)

	return &v
}

// RepoTestsuiteVariations maps file→variations
type AllTestsuiteVariations map[string]variations

func (av AllTestsuiteVariations) Files() []string {
	keys := make([]string, 0, len(av))
	for k := range av {
		keys = append(keys, k)
	}
	return keys
}

func (v variations) Repos() []string {
	var rvals []string
	for _, path := range v.Paths {
		rvals = append(rvals, path.Repo)
	}
	return newSetFromSlices(rvals).Members()
}

func (v variations) Branches(repo string) []string {
	var rvals []string
	for _, path := range v.Paths {
		if path.Repo == repo {
			rvals = append(rvals, path.Branch)
		}
	}
	return newSetFromSlices(rvals).Members()
}

func (v variations) Triggers(repo, branch string) []string {
	var rvals []string
	for _, path := range v.Paths {
		if path.Repo == repo && path.Branch == branch {
			rvals = append(rvals, path.Trigger)
		}
	}
	return newSetFromSlices(rvals).Members()
}

func (v variations) Testsuites(repo, branch, trigger string) []string {
	var rvals []string
	for _, path := range v.Paths {
		if path.Repo == repo && path.Branch == branch && path.Trigger == trigger {
			rvals = append(rvals, path.Testsuite)
		}
	}
	return newSetFromSlices(rvals).Members()
}

func (v variations) Lookup(repo, branch, trigger, testsuite string) *ghMatrix {
	m, found := v.Leaves[createVariationKey(repo, branch, trigger, testsuite)]
	if !found {
		log.Debug().Msgf("(%s, %s, %s, %s) not known, using (%s, master, %s, %s)", repo, branch, trigger, testsuite, repo, trigger, testsuite)
		m = v.Leaves[createVariationKey(repo, "master", trigger, testsuite)]
	}
	return &m
}

func createVariationKey(keys ...string) string {
	var vkey string
	for _, key := range keys {
		vkey += key
	}
	return vkey
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
		av[fname] = *tv
		numVariations++
	}
	if numVariations < 1 {
		return av, fmt.Errorf("No loadable files in %s", tvDir)
	}
	return av, nil
}

// loadVariation unrolls the compact saved representation from a file
// it also sets up handlers for the loaded variations
func loadVariation(tvFile string) (*variations, error) {
	v := NewVariations()
	var vp variationPath

	data, err := os.ReadFile(tvFile)
	if err != nil {
		return v, err
	}
	var saved ghMatrix
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return v, fmt.Errorf("could not unmarshal data from %s: %s: %w", tvFile, string(data), err)
	}
	// top level variations
	var global ghMatrix
	err = copier.CopyWithOption(&global, &saved, copier.Option{IgnoreEmpty: true})
	if err != nil {
		log.Warn().Err(err).Msgf("could not copy global variations")
	}
	parseVariations(saved, 0, v, vp)
	log.Debug().Interface("v", v).Msgf("loaded from %s", tvFile)
	return v, nil
}

func parseVariations(sv ghMatrix, depth int, v *variations, path variationPath) {
	for level, levelMatrix := range sv.Level {
		levelMatrix.EnvFiles = append(levelMatrix.EnvFiles, sv.EnvFiles...)
		levelMatrix.Pump = removeDuplicates(append(levelMatrix.Pump, sv.Pump...))
		levelMatrix.Sink = removeDuplicates(append(levelMatrix.Sink, sv.Sink...))
		levelMatrix.Distros.Deb = removeDuplicates(append(levelMatrix.Distros.Deb, sv.Distros.Deb...))
		levelMatrix.Distros.Rpm = removeDuplicates(append(levelMatrix.Distros.Rpm, sv.Distros.Rpm...))
		switch depth {
		case Testsuite:
			path.Testsuite = level
		case Branch:
			path.Branch = level
		case Trigger:
			path.Trigger = level
		case Repo:
			path.Repo = level
			key := createVariationKey(path.Repo, path.Branch, path.Trigger, path.Testsuite)
			var tsPath = path // make a copy
			v.Leaves[key] = levelMatrix
			v.Paths = append(v.Paths, tsPath)
		}
		if depth > Repo {
			log.Warn().Fields(sv).Msgf("cannot parse test variation levels > %d", Repo)
			return
		} else {
			parseVariations(levelMatrix, depth+1, v, path)
		}
	}
	return
}
