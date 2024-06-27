package policy

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/TykTechnologies/gromit/config"
)

// TestBundleRender renders all the bundles in templates/ for all the
// repos in the config file.
// FIXME: Test (bundle, features, repo) in parallel
func TestBundleRender(t *testing.T) {
	var pol Policies
	config.LoadConfig("")
	err := LoadRepoPolicies(&pol)
	if err != nil {
		t.Fatalf("Unable to load repo policies: %v", err)
	}

	// Iterate through each group->repo->branch  and get the exhaustive list of  features
	// used by each group
	groupFeatures := make(map[string][]string)
	for g, group := range pol.Groups {
		var features []string
		features = append(features, group.Features...)
		for _, repo := range group.Repos {
			features = append(features, repo.Features...)
			for _, branch := range repo.Branches {
				features = append(features, branch.Features...)

			}
		}
		slices.Sort(features)
		groupFeatures[g] = slices.Compact(features)
		t.Logf("all features used by group %s : %v", g, groupFeatures[g])
	}

	for grpName, grp := range pol.Groups {
		b, err := NewBundle(groupFeatures[grpName])
		if err != nil {
			t.Logf("Unable to create bundle obj: %v", err)
		}
		for r := range grp.Repos {
			t.Logf("testing repo %s from group %s with features %v", r, grpName, groupFeatures[grpName])
			rp, err := pol.GetRepoPolicy(r)
			if err != nil {
				t.Logf("Error getting repo policy for repo: %s: %v", r, err)
				t.Fail()
				continue
			}
			rp.SetTimestamp(time.Now().UTC())
			tmpDir, err := os.MkdirTemp("", r+"-"+b.Name)
			if err != nil {
				t.Fatalf("Error creating temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			err = rp.SetBranch("master")
			if err != nil {
				t.Logf("Could not set branch to master for repo: %s", r)
				t.Fail()
				continue
			}
			_, err = b.Render(rp, tmpDir, nil)
			if err != nil {
				t.Logf("Error rendering bundle: %s for repo: %s: %v", b.Name, r, err)
				t.Fail()
				continue
			}
			renderCount, err := countFiles(tmpDir)
			if err != nil {
				t.Fatalf("Could not count the rendered files in %s: %v", tmpDir, err)
			}
			expectedCount := b.Count()
			if renderCount != expectedCount {
				t.Fatalf("Rendered %d files, expected %d", renderCount, expectedCount)
			}
		}
	}
}

func countFiles(tmpDir string) (int, error) {
	count := 0
	err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
