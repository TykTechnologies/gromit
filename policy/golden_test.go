package policy

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "regenerate testdata/golden from the current templates and config")

// TestGoldenRender renders every repo/branch combination in the production
// config and compares the output byte-for-byte against the golden files
// committed under testdata/golden.
//
// This is our safety net for template and config changes: if a PR changes
// what gromit generates, this test fails until the goldens are regenerated,
// and the regenerated goldens show up in the PR diff so reviewers see the
// exact effect on every repo and branch. A pure refactor (config collapse,
// template cleanup) should produce no golden changes at all.
//
// To update after an intended change, run `make update-golden`, then commit
// the changes under policy/testdata/golden and check the git diff is what
// you meant to do.
func TestGoldenRender(t *testing.T) {
	config.LoadConfig("")
	var pol Policies
	require.NoError(t, LoadRepoPolicies(&pol), "loading embedded config")

	goldenRoot := filepath.Join("testdata", "golden")
	if *updateGolden {
		require.NoError(t, os.RemoveAll(goldenRoot))
	}

	for _, grp := range pol.Groups {
		for repoName := range grp.Repos {
			rp, err := pol.GetRepoPolicy(repoName)
			require.NoErrorf(t, err, "repopolicy %s", repoName)
			for _, branch := range rp.GetAllBranches() {
				t.Run(repoName+"/"+branch, func(t *testing.T) {
					rp, err := pol.GetRepoPolicy(repoName)
					require.NoError(t, err)
					require.NoError(t, rp.SetBranch(branch))
					b, err := NewBundle(rp.Branchvals.Features)
					require.NoErrorf(t, err, "bundle %v", rp.Branchvals.Features)

					opDir := t.TempDir()
					_, err = b.Render(rp, opDir, nil)
					require.NoError(t, err, "render")

					goldenDir := filepath.Join(goldenRoot, repoName, branch)
					if *updateGolden {
						require.NoError(t, copyTree(opDir, goldenDir))
						return
					}
					compareTrees(t, goldenDir, opDir)
				})
			}
		}
	}
}

// compareTrees fails the test with one error per file that differs between
// the golden tree and the freshly rendered tree, in either direction.
func compareTrees(t *testing.T, goldenDir, renderedDir string) {
	t.Helper()
	golden, err := treeContents(goldenDir)
	if os.IsNotExist(err) {
		t.Fatalf("no golden files at %s; run `make update-golden` and commit the result", goldenDir)
	}
	require.NoError(t, err)
	rendered, err := treeContents(renderedDir)
	require.NoError(t, err)

	for path, want := range golden {
		got, ok := rendered[path]
		if !ok {
			t.Errorf("%s: in golden but no longer rendered", path)
			continue
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: rendered output differs from golden", path)
		}
	}
	for path := range rendered {
		if _, ok := golden[path]; !ok {
			t.Errorf("%s: rendered but missing from golden", path)
		}
	}
	if t.Failed() {
		t.Log("if this change is intended, run `make update-golden` and commit the golden diff")
	}
}

// treeContents returns relative path -> file contents for every regular file
// under root.
func treeContents(root string) (map[string][]byte, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	contents := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		contents[rel] = data
		return nil
	})
	return contents, err
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return fmt.Errorf("writing golden %s: %w", target, err)
		}
		return nil
	})
}
