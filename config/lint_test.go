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

// pkgsAlias maps a policy repo to its packagecloud repo where neither
// the repo name nor its packagename matches: tyk-sink publishes its
// packages as tyk-mdcb.
var pkgsAlias = map[string]string{
	"tyk-sink": "tyk-mdcb",
}

// TestUpgradeVersionsProtected keeps the pruning exceptions in sync with
// the upgrade tests.
//
// Every repo's upgradefromver is a version that upgrade tests install
// from packagecloud, so pruning must never delete it. The protection
// lives in the pkgs exceptions list, which is maintained by hand and can
// silently drift when upgradefromver moves: tyk-dashboard bumped to
// 3.0.9 while the exception still said 3.0.8. That is survivable only
// while the version cutoff happens to sit below the test version; the
// moment the cutoff moves, the package is deleted and every upgrade test
// on every branch fails.
//
// This test fails any change where an upgradefromver (repo or branch
// level) is missing from the corresponding pkgs exceptions list.
func TestUpgradeVersionsProtected(t *testing.T) {
	raw, err := os.ReadFile("config.yaml")
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	var doc struct {
		Policy struct {
			Groups map[string]map[string]any `yaml:"groups"`
		} `yaml:"policy"`
		Pkgs map[string]struct {
			Exceptions []string `yaml:"exceptions"`
		} `yaml:"pkgs"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse config.yaml: %v", err)
	}

	var findings []string
	for gName, group := range doc.Policy.Groups {
		repos, _ := group["repos"].(map[string]any)
		for rName, r := range repos {
			repo, _ := r.(map[string]any)
			if repo == nil {
				continue
			}

			// versions to protect -> where they were configured
			versions := make(map[string]string)
			if v, ok := repo["upgradefromver"]; ok {
				versions[fmt.Sprintf("%v", v)] = fmt.Sprintf("%s.%s", gName, rName)
			}
			branches, _ := repo["branches"].(map[string]any)
			for bName, b := range branches {
				branch, _ := b.(map[string]any)
				if v, ok := branch["upgradefromver"]; branch != nil && ok {
					versions[fmt.Sprintf("%v", v)] = fmt.Sprintf("%s.%s.branches.%s", gName, rName, bName)
				}
			}
			if len(versions) == 0 {
				continue
			}

			// The packagecloud repo is the packagename, unless aliased.
			pkgsName := rName
			if pn, ok := repo["packagename"].(string); ok && pn != "" {
				pkgsName = pn
			}
			if alias, ok := pkgsAlias[rName]; ok {
				pkgsName = alias
			}
			pkgsRepo, pruned := doc.Pkgs[pkgsName]
			if !pruned {
				// not a pruned repo (e.g. tyk-ai-studio), nothing to protect
				continue
			}

			exceptions := make(map[string]bool)
			for _, e := range pkgsRepo.Exceptions {
				exceptions[e] = true
			}
			for version, where := range versions {
				if !exceptions["v"+version] {
					findings = append(findings, fmt.Sprintf(
						"%s: upgradefromver %s is not in pkgs.%s.exceptions; pruning could delete it and break upgrade tests. Add `- v%s # used in upgrade tests`",
						where, version, pkgsName, version))
				}
			}
		}
	}

	for _, f := range findings {
		t.Errorf("  - %s", f)
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
