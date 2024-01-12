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

	p := NewParams("REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG", "IS_LTS")

	assert.Equal(t, "repo", p["repo"])
	assert.Equal(t, "main", p["base_ref"])
	assert.Equal(t, "v1.0", p["firstTag"])
	assert.Equal(t, "is_tag", p["trigger"])
	assert.Equal(t, "master", p["gdTag"])

	gdTagTests := []struct {
		baseRef string
		repo    string
		want    string
	}{
		{"refs/heads/release-4.0.12", "tyk", "release-4-lts"},
		{"release-5.1.12", "TykTechnologies/tyk-analytics", "master"},
		{"refs/heads/master", "tyk-analytics", "master"},
		{"refs/heads/release-4.0.12", "tyk-pump", "master"},
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
	os.Clearenv()
	// Test case with all parameters set in the environment
	os.Setenv("REPO", "github.com/username/tyk")
	os.Setenv("BASE_REF", "refs/heads/main")
	os.Setenv("TAGS", "v1.0 v1.1 v1.2")
	os.Setenv("IS_PR", "yes")

	var op bytes.Buffer
	p := NewParams("REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG", "IS_LTS")
	if err := p.SetVersions(&op); err != nil {
		t.Error(err)
	}
	op.WriteString("\n")

	// conf is the set of configuration variations
	// db is the databases to use
	// pump/sink are included only when needed
	defaults := TestVariations{
		"conf": []string{"sha256", "murmur64"},
		"db":   []string{"mongo44", "postgres15"},
		"pump": []string{"tykio/tyk-pump-docker-pub:v1.8.3", "$ECR/tyk-pump:master"},
		"sink": []string{"tykio/tyk-mdcb-docker:v2.4.2", "$ECR/tyk-sink:master"},
	}
	if err := p.SetVariations(&op, defaults); err != nil {
		t.Error(err)
	}
	const want = `versions<<EOF
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
conf<<EOF
["sha256"]
EOF
db<<EOF
["mongo44","postgres15"]
EOF
pump<<EOF
["$ECR/tyk-pump:master"]
EOF
sink<<EOF
["$ECR/tyk-sink:master"]
EOF
`
	assert.Equal(t, "is_pr", p["trigger"])
	assert.Equal(t, want, op.String())
}

func TestTriggerPriority(t *testing.T) {
	os.Setenv("IS_PR", "no")
	os.Setenv("IS_TAG", "yes")
	os.Setenv("IS_LTS", "yes")

	// IS_TAG appears after IS_LTS so the trigger should be is_tag
	p := NewParams("REPO", "BASE_REF", "TAGS", "IS_PR", "IS_LTS", "IS_TAG")

	assert.Equal(t, "is_tag", p["trigger"])
}

func TestDefaults(t *testing.T) {
	// Test case with no parameters set in the environment
	os.Clearenv()

	p := NewParams("REPO", "BASE_REF", "TAGS", "IS_PR", "IS_TAG", "IS_LTS")

	assert.Empty(t, p["repo"])
	assert.Empty(t, p["base_ref"])
	assert.Empty(t, p["tags"])
	assert.Empty(t, p["is_pr"])
	assert.Empty(t, p["is_tag"])
	assert.Empty(t, p["is_lts"])
	assert.Empty(t, p["firstTag"])
	assert.Empty(t, p["trigger"])
	assert.Equal(t, "master", p["gdTag"])
}
