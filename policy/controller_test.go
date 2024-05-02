package policy

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParams(t *testing.T) {
	// Test case with all parameters set in the environment
	os.Setenv("REPO", "github.com/username/repo")
	os.Setenv("BASE_REF", "refs/heads/main")
	os.Setenv("TAGS", "v1.0 v1.1 v1.2")
	os.Setenv("IS_PR", "no")
	os.Setenv("IS_TAG", "yes")
	os.Setenv("IS_LTS", "no")
	os.Setenv("JOB", "ui")

	p := NewParams("JOB", "REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG", "IS_LTS")

	assert.Equal(t, "repo", p["repo"])
	assert.Equal(t, "main", p["base_ref"])
	assert.Equal(t, "v1.0", p["firstTag"])
	assert.Equal(t, "is_tag", p["trigger"])
	assert.Equal(t, "master", p["gdTag"])

	gdTagTests := []struct {
		job     string
		baseRef string
		repo    string
		want    string
	}{
		{"api", "refs/heads/release-4.0.12", "tyk", "release-4-lts"},
		{"ui", "release-5.1.12", "TykTechnologies/tyk-analytics", "master"},
		{"api", "refs/heads/master", "tyk-analytics", "master"},
		{"ui", "refs/heads/release-4.0.12", "tyk-pump", "master"},
		{"api", "refs/heads/release-4.0.13", "tyk-automated-tests", "release-4-lts"},
		{"ui", "refs/heads/release-5.1.13", "tyk-automated-tests", "master"},
	}
	for _, tc := range gdTagTests {
		t.Run(fmt.Sprintf("%s/%s", tc.repo, tc.want), func(t *testing.T) {
			os.Setenv("BASE_REF", tc.baseRef)
			os.Setenv("REPO", tc.repo)
			p := NewParams("BASE_REF", "REPO")
			assert.Equal(t, tc.want, p["gdTag"])
		})
	}
}

func TestOutput(t *testing.T) {
	testCases := []struct {
		job     string
		want    string
		trigger string
		isPR    string
		isTag   string
		isLTS   string
		baseRef string
	}{
		{
			job: "ui",
			want: `versions<<EOF
tyk_image=$ECR/tyk:master
tyk_analytics_image=$ECR/tyk-analytics:master
tyk_pump_image=$ECR/tyk-pump:master
tyk_sink_image=$ECR/tyk-sink:master
# override default above with just built tag
tyk_image=v1.0
# alfa and beta have to come after the override
tyk_alfa_image=$tyk_image
tyk_beta_image=$tyk_image
EOF
gd_tag=master
pump<<EOF
["$ECR/tyk-pump:master"]
EOF
sink<<EOF
["$ECR/tyk-sink:master"]
EOF
ui_cache_db<<EOF
["redis7"]
EOF
ui_conf<<EOF
["sha256","murmur128"]
EOF
ui_db<<EOF
["mongo7","postgres15"]
EOF
exclude<<EOF
[{"db":"mongo7","ui_conf":"murmur128"},{"db":"postgres15","ui_conf":"sha256"}]
EOF
`,
			trigger: "is_pr",
			isPR:    "yes",
			isTag:   "no",
			isLTS:   "no",
			baseRef: "refs/heads/main",
		},
		{
			job: "ui",
			want: `versions<<EOF
tyk_image=$ECR/tyk:master
tyk_analytics_image=$ECR/tyk-analytics:master
tyk_pump_image=$ECR/tyk-pump:master
tyk_sink_image=$ECR/tyk-sink:master
# override default above with just built tag
tyk_image=v1.0
# alfa and beta have to come after the override
tyk_alfa_image=$tyk_image
tyk_beta_image=$tyk_image
EOF
gd_tag=master
pump<<EOF
["tykio/tyk-pump-docker-pub:v1.8","$ECR/tyk-pump:master"]
EOF
sink<<EOF
["tykio/tyk-mdcb-docker:v2.4","$ECR/tyk-sink:master"]
EOF
ui_cache_db<<EOF
["redis7"]
EOF
ui_conf<<EOF
["sha256","murmur128"]
EOF
ui_db<<EOF
["mongo7","postgres15"]
EOF
exclude<<EOF
[{"pump":"tykio/tyk-pump-docker-pub:v1.8","sink":"$ECR/tyk-sink:master"},{"pump":"$ECR/tyk-pump:master","sink":"tykio/tyk-mdcb-docker:v2.4"},{"db":"mongo7","ui_conf":"murmur128"},{"db":"postgres15","ui_conf":"sha256"}]
EOF
`,
			trigger: "",
			isPR:    "no",
			isTag:   "no",
			isLTS:   "no",
			baseRef: "refs/heads/main",
		}, {
			job: "api",
			want: `versions<<EOF
tyk_image=$ECR/tyk:release-5-lts
tyk_analytics_image=$ECR/tyk-analytics:release-5-lts
tyk_pump_image=$ECR/tyk-pump:master
tyk_sink_image=$ECR/tyk-sink:master
# override default above with just built tag
tyk_image=v1.0
# alfa and beta have to come after the override
tyk_alfa_image=$tyk_image
tyk_beta_image=$tyk_image
EOF
gd_tag=release-5-lts
api_cache_db<<EOF
["redis7"]
EOF
api_conf<<EOF
["sha256","murmur128"]
EOF
api_db<<EOF
["mongo7","postgres15"]
EOF
pump<<EOF
["tykio/tyk-pump-docker-pub:v1.8","$ECR/tyk-pump:master"]
EOF
sink<<EOF
["tykio/tyk-mdcb-docker:v2.4","$ECR/tyk-sink:master"]
EOF
exclude<<EOF
[{"pump":"tykio/tyk-pump-docker-pub:v1.8","sink":"$ECR/tyk-sink:master"},{"pump":"$ECR/tyk-pump:master","sink":"tykio/tyk-mdcb-docker:v2.4"},{"api_conf":"murmur128","db":"mongo7"},{"api_conf":"sha256","db":"postgres15"}]
EOF
`,
			trigger: "is_lts",
			isPR:    "no",
			isTag:   "no",
			isLTS:   "yes",
			baseRef: "release-5-lts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.job, func(t *testing.T) {
			os.Clearenv()
			// Set up environment variables based on test case
			os.Setenv("REPO", "github.com/username/tyk")
			os.Setenv("TAGS", "v1.0 v1.1 v1.2")
			os.Setenv("JOB", tc.job)
			os.Setenv("IS_PR", tc.isPR)
			os.Setenv("IS_TAG", tc.isTag)
			os.Setenv("IS_LTS", tc.isLTS)
			os.Setenv("BASE_REF", tc.baseRef)

			var op bytes.Buffer
			p := NewParams("JOB", "REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG")
			if err := p.SetVersions(&op); err != nil {
				t.Error(err)
			}
			op.WriteString("\n")

			defaults := GHoutput{
				TestVariations: map[string][]string{
					p["job"] + "_conf":     {"sha256", "murmur128"},
					p["job"] + "_db":       {"mongo7", "postgres15"},
					p["job"] + "_cache_db": {"redis7"},
					"pump":                 {"tykio/tyk-pump-docker-pub:v1.8", "$ECR/tyk-pump:master"},
					"sink":                 {"tykio/tyk-mdcb-docker:v2.4", "$ECR/tyk-sink:master"},
				},
				Exclusions: []map[string]string{
					{"pump": "tykio/tyk-pump-docker-pub:v1.8", "sink": "$ECR/tyk-sink:master"},
					{"pump": "$ECR/tyk-pump:master", "sink": "tykio/tyk-mdcb-docker:v2.4"},
					{"db": "mongo7", p["job"] + "_conf": "murmur128"},
					{"db": "postgres15", p["job"] + "_conf": "sha256"},
				},
			}

			if err := p.SetOutputs(&op, defaults); err != nil {
				t.Error(err)
			}

			assert.Equal(t, tc.trigger, p["trigger"])
			assert.Equal(t, tc.want, op.String())
		})
	}
}

func TestTriggerPriority(t *testing.T) {
	// Test case with no parameters set in the environment
	os.Clearenv()

	os.Setenv("JOB", "api")
	os.Setenv("IS_PR", "no")
	os.Setenv("IS_TAG", "yes")
	os.Setenv("IS_LTS", "yes")

	// IS_TAG appears after IS_LTS so the trigger should be is_tag
	p := NewParams("JOB", "REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG")

	assert.Equal(t, "is_tag", p["trigger"])
}

func TestDefaults(t *testing.T) {
	// Test case with no parameters set in the environment
	os.Clearenv()

	p := NewParams("JOB", "REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG")

	assert.Empty(t, p["job"])
	assert.Empty(t, p["repo"])
	assert.Empty(t, p["base_ref"])
	assert.Empty(t, p["tags"])
	assert.Empty(t, p["is_pr"])
	assert.Empty(t, p["is_tag"])
	assert.Empty(t, p["firstTag"])
	assert.Empty(t, p["trigger"])
	assert.Equal(t, "master", p["gdTag"])
}
