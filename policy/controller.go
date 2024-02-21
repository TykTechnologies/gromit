package policy

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/constraints"
)

// policy.SetVariations needs to print the variations in a
// well-defined order for TestOutput
func sortedKeys[K constraints.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// TestVariations models the variations of the test matrix in
// release.yml:api-tests. Each key is a row in the matrix.
type TestVariations map[string][]string

// runParameters is a private type that models the runtime parameters
// required to test a repo
type runParameters map[string]string

// SetVariations prints the test variations formatted as a multi-line
// github output parameter. The contents are json formatted.
func (p runParameters) SetVariations(op io.Writer, tv TestVariations) error {

	if p["job"] == "api-test" {
		switch p["trigger"] {
		case "is_pr":
			tv["api_conf"] = []string{"sha256"}
			tv["api_db"] = []string{"mongo44", "postgres15"}
			tv["pump"] = []string{"$ECR/tyk-pump:master"}
			tv["sink"] = []string{"$ECR/tyk-sink:master"}
		case "is_tag":
			// Defaults are fine
			tv["api_conf"] = []string{"sha256"}
			tv["api_db"] = []string{"mongo44", "postgres15"}
		case "is_lts":
			tv["api_conf"] = []string{"sha256"}
			tv["api_db"] = []string{"mongo44", "postgres15"}
			tv["pump"] = []string{"tykio/tyk-pump-docker-pub:v1.8"}
			tv["sink"] = []string{"tykio/tyk-mdcb-docker:v2.4"}
		}
	}

	if p["job"] == "ui-test" {
		switch p["trigger"] {
		case "is_pr":
			tv["ui_conf"] = []string{"sha256"}
			tv["ui_db"] = []string{"mongo44", "postgres15"}
			tv["pump"] = []string{"$ECR/tyk-pump:master"}
			tv["sink"] = []string{"$ECR/tyk-sink:master"}
		case "is_tag":
			// Defaults are fine
			tv["ui_conf"] = []string{"sha256"}
			tv["ui_db"] = []string{"mongo44", "postgres15"}
		case "is_lts":
			tv["ui_conf"] = []string{"sha256"}
			tv["ui_db"] = []string{"mongo44", "postgres15"}
			tv["pump"] = []string{"tykio/tyk-pump-docker-pub:v1.8"}
			tv["sink"] = []string{"tykio/tyk-mdcb-docker:v2.4"}
		}
	}

	for _, v := range sortedKeys(tv) {
		json, err := json.Marshal(tv[v])
		if err != nil {
			return err
		}
		ghop := fmt.Sprintf("%s<<EOF\n%s\nEOF\n", v, json)
		if _, err := op.Write([]byte(ghop)); err != nil {
			return err
		}
	}
	return nil
}

// SetVersions returns the preamble to versions.env and the tag that
// should be used for tyk-automated-tests, which should follow the
// gateway or dashboard tag. The output, formatted as a multi-line
// github actions output is written to op. ECR is set before env up in
// release.yml:api-tests
func (p runParameters) SetVersions(op io.Writer) error {
	return template.Must(template.New("policy").Funcs(sprig.TxtFuncMap()).Parse(`versions<<EOF
tyk_image=$ECR/tyk:{{ .gdTag }}
tyk_analytics_image=$ECR/tyk-analytics:{{ .gdTag }}
tyk_pump_image=$ECR/tyk-pump:master
tyk_sink_image=$ECR/tyk-sink:master
# override default above with just built tag
{{ .repo | replace "-" "_" }}_image={{ .firstTag }}
# alfa and beta have to come after the override
tyk_alfa_image=$tyk_image
tyk_beta_image=$tyk_image
EOF
gd_tag={{ .gdTag }}`)).Execute(op, p)
}

// NewParams looks in the environment for the named parameters and
// returns a map suitable for usage in versions.env and to decide the
// test scope
func NewParams(paramNames ...string) runParameters {
	var trigger, firstTag string
	params := make(runParameters)
	for _, pn := range paramNames {
		p := os.Getenv(pn)
		if p == "" {
			log.Warn().Msgf("%s is nil", pn)
		}
		switch pn {
		case "REPO", "BASE_REF":
			p = p[strings.LastIndex(p, "/")+1:]
		case "TAGS":
			if tags := strings.Fields(p); len(tags) > 0 {
				firstTag = tags[0]
			}
		case "IS_PR", "IS_TAG":
			if p == "yes" {
				trigger = strings.ToLower(pn)
			}
		}
		log.Trace().Msgf("env %s: %s", pn, p)
		params[strings.ToLower(pn)] = p
	}
	params["firstTag"] = firstTag
	params["trigger"] = trigger

	params["gdTag"] = "master"
	ltsBranch := regexp.MustCompile(`^release-(\d+)(?:\.0(?:\.\d+)?)?(?:-(lts|\d+(?:\.0)?))?$`).FindStringSubmatch(params["base_ref"])
	repo := params["repo"]
	if (repo == "tyk" || repo == "tyk-analytics" || repo == "tyk-automated-tests") && len(ltsBranch) > 0 {
		log.Debug().Msgf("detected %s LTS branch", repo)
		params["gdTag"] = fmt.Sprintf("release-%s-lts", ltsBranch[1])
		params["trigger"] = "is_lts"
	}

	log.Debug().Interface("params", params).Msg("calculated from env")

	return params
}
