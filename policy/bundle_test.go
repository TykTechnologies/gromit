package policy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TykTechnologies/gromit/config"
)

// TestBundleRender renders all the bundles in templates/ for all the
// repos in the config file.
// FIXME: Test (bundle, features, repo) in parallel
func TestBundleRender(t *testing.T) {
	featDirs, err := templates.ReadDir("templates")
	if err != nil {
		t.Fatalf("Error reading embedded fs: %v", err)
	}
	var features []string
	for _, fd := range featDirs {
		if fd.IsDir() && fd.Name() != "subtemplates" {
			features = append(features, fd.Name())
		}
	}
	var pol Policies
	config.LoadConfig("")
	err = LoadRepoPolicies(&pol)
	if err != nil {
		t.Fatalf("Unable to load repo policies: %v", err)
	}
	b, err := NewBundle(features)
	if err != nil {
		t.Logf("Unable to create bundle obj: %v", err)
	}
	for r := range pol.Repos {
		t.Logf("testing repo %s with features %v", r, features)
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
