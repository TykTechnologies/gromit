package policy

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/TykTechnologies/gromit/config"
	"errors"
	"path/filepath"
	"io/fs"
)

// FIXME: Test (bundle, features, repo) in parallel
func TestBundleRender(t *testing.T) {
	dirs, err := Bundles.ReadDir("templates")
	if err != nil {
		t.Fatalf("Error reading embed fs: %v", err)
	}
	var pol Policies
	config.LoadConfig("")
	err = LoadRepoPolicies(&pol)
	if err != nil {
		t.Fatalf("Unable to load repo policies: %v", err)
	}
	for _, d := range dirs {
		bundleName := d.Name()
		if d.IsDir() {
			featDirs, err := Features.ReadDir(filepath.Join("template-features", bundleName))
			if !(err == nil || errors.Is(err, fs.ErrNotExist)) {
				t.Fatalf("Unable to load features for bundle %s: %v", bundleName, err)
			}
			var features []string
			for _, feat := range featDirs {
				features = append(features, feat.Name())
			}
			b, err := NewBundle(bundleName, features)
			if err != nil {
				t.Logf("Unable to create bundle obj: %v", err)
				continue
			}
			for r := range pol.Repos {
				t.Logf("Testing bundle %s on repo %s with features %v", bundleName, r, features)
				rp, err := GetRepoPolicy(r, "master")
				if err != nil {
					t.Logf("Error getting repo policy for repo: %s: %v", r, err)
					t.Fail()
					continue
				}
				rp.SetTimestamp(time.Now().UTC())
				tmpDir, err := ioutil.TempDir("", r+"-"+bundleName)
				if err != nil {
					t.Fatalf("Error creating temp dir: %v", err)
				}
				defer os.RemoveAll(tmpDir)

				_, err = b.Render(rp, tmpDir, nil)
				if err != nil {
					t.Logf("Error rendering bundle: %s for repo: %s: %v", bundleName, r, err)
					t.Fail()
					continue
				}

			}

		}

	}

}
