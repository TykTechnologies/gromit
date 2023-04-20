package policy

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/TykTechnologies/gromit/config"
)

func TestBundleRender(t *testing.T) {
	dirs, err := Bundles.ReadDir("templates")
	if err != nil {
		t.Fatalf("Error reading embed fs: %v", err)
	}
	var pol Policies
	config.LoadConfig("")
	err = LoadRepoPolicies(&pol)
	t.Logf("Policy object: %v", pol)
	if err != nil {
		t.Fatalf("Unable to load repo policies: %v", err)
	}
	for _, d := range dirs {
		bundleName := d.Name()
		if d.IsDir() {
			b, err := NewBundle(bundleName)
			if err != nil {
				t.Logf("Unable to create bundle obj: %v", err)
				continue
			}
			for r, _ := range pol.Repos {
				t.Logf("Testing bundle: %s on repo: %s", bundleName, r)
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
