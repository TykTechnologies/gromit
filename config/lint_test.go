package config

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

var nonScalarKeys = map[string]bool{
	"repos":        true,
	"branches":     true,
	"builds":       true,
	"features":     true,
	"deletedfiles": true,
	"tests":        true,
}

// TestConfigRedundancy keeps config.yaml honest about its own cascade.
//
// Values flow group -> repo -> branch, and features are unioned across all
// three levels. So repeating an inherited value (or feature) at a deeper
// level does nothing - it just looks meaningful and invites drift when we
// update one copy and miss the others. This test fails if config.yaml
// restates anything the cascade already provides.
func TestConfigRedundancy(t *testing.T) {
	raw, err := os.ReadFile("config.yaml")
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	var doc struct {
		Policy struct {
			Groups map[string]map[string]any `yaml:"groups"`
		} `yaml:"policy"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse config.yaml: %v", err)
	}

	var findings []string
	var warnings []string

	for gName, group := range doc.Policy.Groups {
		groupFeatures := toStringSet(group["features"])

		repos, _ := group["repos"].(map[string]any)
		for rName, r := range repos {
			repo, _ := r.(map[string]any)
			if repo == nil {
				continue
			}
			repoPath := fmt.Sprintf("%s.%s", gName, rName)

			for k, v := range repo {
				if nonScalarKeys[k] {
					continue
				}
				if gv, ok := group[k]; ok && gv == v {
					findings = append(findings,
						fmt.Sprintf("%s: %q repeats the group default (%v)", repoPath, k, v))
				}
			}

			repoFeatures := toStringSet(repo["features"])
			for f := range repoFeatures {
				if groupFeatures[f] {
					findings = append(findings,
						fmt.Sprintf("%s: feature %q is already set at group level", repoPath, f))
				}
			}

			branches, _ := repo["branches"].(map[string]any)
			for bName, b := range branches {
				branch, _ := b.(map[string]any)
				if branch == nil {
					continue
				}
				branchPath := fmt.Sprintf("%s.branches.%s", repoPath, bName)

				for k, v := range branch {
					if nonScalarKeys[k] {
						continue
					}
					inherited, ok := repo[k]
					if !ok {
						inherited, ok = group[k]
					}
					if ok && inherited == v {
						findings = append(findings,
							fmt.Sprintf("%s: %q repeats the inherited value (%v)", branchPath, k, v))
					}
					// Heads up: setting `<key>: false` on a branch when the repo
					// has `<key>: true` does NOT work. Our merge (copier with
					// IgnoreEmpty) treats false as empty and drops it, so the
					// branch silently stays true. This is why tyk-analytics
					// release branches say `cgo: false` but have always built
					// with cgo on.
					//
					// Keeping this as a warning until we settle the
					// tyk-analytics cgo intent; after that, we'll move it into
					// `findings` so it fails the build.
					if v == false && inherited == true {
						warnings = append(warnings,
							fmt.Sprintf("%s: %q set to false but inherits true; this override is silently ignored (the merge drops zero values), the effective value is true", branchPath, k))
					}
				}

				for f := range toStringSet(branch["features"]) {
					if repoFeatures[f] || groupFeatures[f] {
						findings = append(findings,
							fmt.Sprintf("%s: feature %q is already set at repo or group level", branchPath, f))
					}
				}
			}
		}
	}

	for _, w := range warnings {
		t.Logf("WARNING: %s", w)
	}
	if len(findings) > 0 {
		t.Errorf("config.yaml restates %d inherited value(s); remove them, the cascade already provides them:", len(findings))
		for _, f := range findings {
			t.Errorf("  - %s", f)
		}
	}
}

func toStringSet(v any) map[string]bool {
	set := make(map[string]bool)
	list, _ := v.([]any)
	for _, item := range list {
		if s, ok := item.(string); ok {
			set[s] = true
		}
	}
	return set
}
