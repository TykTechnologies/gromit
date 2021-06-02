package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
)

var dashboardFiles = []string{
	"bin/unlock-agent.sh",
	".goreleaser.yml",
	"Dockerfileg.std",
	"Dockerfile.slim",
	"aws/byol.pkr.hcl",
	".github/workflows/release.yml",
	".github/workflows/del-env.yml",
	"integration/terraform/outputs.tf",
	"install/before_install.sh",
	"install/post_install.sh",
	"install/post_remove.sh",
	".github/workflows/update-gomod.yml",
	".github/workflows/build-assets.yml",
}
var gatewayFiles = []string{
	"bin/unlock-agent.sh",
	".goreleaser.yml",
	"Dockerfileg.std",
	"Dockerfile.slim",
	"aws/byol.pkr.hcl",
	".github/workflows/release.yml",
	".github/workflows/del-env.yml",
	"integration/terraform/outputs.tf",
	"install/before_install.sh",
	"install/post_install.sh",
	"install/post_remove.sh",
	"images/*",
}
var baseFiles = []string{
	"bin/unlock-agent.sh",
	".goreleaser.yml",
	"Dockerfileg.std",
	"Dockerfile.slim",
	"aws/byol.pkr.hcl",
	".github/workflows/release.yml",
	".github/workflows/del-env.yml",
	"integration/terraform/outputs.tf",
	"install/before_install.sh",
	"install/post_install.sh",
	"install/post_remove.sh",
}

// TestPortsPolicy can test a repo with back/forward ports
func TestPortsPolicy(t *testing.T) {
	timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"
	cases := []struct {
		cfgFile     string
		protBranch  string
		testBranch  string
		srcBranches []string
		prVars      prVars
		maVars      maVars
		name        string
	}{
		{
			cfgFile:     "../testdata/policies/gateway.yaml",
			protBranch:  "release-3-lts",
			testBranch:  "release-3.0.5",
			name:        "tyk",
			srcBranches: []string{"release-3.2.0", "release-3.0.5", "release-3.1.2"},
			prVars: prVars{
				RepoName: "tyk",
				Files:    gatewayFiles,
				Backports: map[string]string{
					"release-3.0.5": "releng/release-3-lts",
					"release-3.1.2": "releng/release-3.1",
					"release-3.2.0": "releng/release-3.2",
				},
				Branch: "release-3.0.5",
				Remove: false,
			},
			maVars: maVars{
				Timestamp:  timeStamp,
				MAFiles:    gatewayFiles,
				SrcBranch:  "release-3.0.5",
				DestBranch: "releng/release-3-lts",
			},
		},
		{
			cfgFile:     "../testdata/policies/dashboard.yaml",
			protBranch:  "master",
			testBranch:  "release-3.0.5",
			name:        "tyk-analytics",
			srcBranches: []string{"release-3.1.2", "release-3.0.5"},
			prVars: prVars{
				RepoName: "tyk-analytics",
				Files:    dashboardFiles,
				Backports: map[string]string{
					"release-3.0.5": "releng/release-3-lts",
					"release-3.1.2": "releng/release-3.1",
				},
				Branch: "release-3.0.5",
				Remove: false,
			},
			maVars: maVars{
				Timestamp:  timeStamp,
				MAFiles:    dashboardFiles,
				SrcBranch:  "release-3.0.5",
				DestBranch: "releng/release-3-lts",
			},
		},
		{
			cfgFile:    "../testdata/policies/pump.yaml",
			protBranch: "master",
			name:       "tyk-pump",
			testBranch: "release-1.3",
			prVars: prVars{
				RepoName: "tyk-pump",
				Files:    baseFiles,
				Branch:   "release-1.3",
				Remove:   false,
			},
			maVars: maVars{
				Timestamp: timeStamp,
				MAFiles:   baseFiles,
			},
		},
		{
			cfgFile:     "../testdata/policies/mdcb.yaml",
			protBranch:  "master",
			testBranch:  "release-1.8",
			name:        "tyk-sink",
			srcBranches: []string{"release-1.9", "release-1.8", "release-1.7"},
			prVars: prVars{
				RepoName: "tyk-sink",
				Files:    baseFiles,
				Backports: map[string]string{
					"release-1.9": "releng/master",
					"release-1.8": "releng/master",
					"release-1.7": "releng/master",
				},
				Branch: "release-1.8",
				Remove: false,
			},
			maVars: maVars{
				Timestamp:  timeStamp,
				MAFiles:    baseFiles,
				SrcBranch:  "release-1.8",
				DestBranch: "releng/master",
			},
		},
	}
	var rp RepoPolicies
	for _, tc := range cases {
		t.Run(tc.name, func(T *testing.T) {
			config.LoadConfig(tc.cfgFile)
			err := LoadRepoPolicies(&rp)
			if err != nil {
				t.Fatalf("Could not load policy for %s: %v", tc.name, err)
			}
			isProtected, err := rp.IsProtected(tc.name, tc.protBranch)
			if err != nil {
				t.Errorf("Failed to get protected status for %s: %w", tc.protBranch, err)
			}
			assert.Equal(t, true, isProtected)
			srcBranches, err := rp.SrcBranches(tc.name)
			if err != nil {
				t.Errorf("Failed to get source branches for %s: %w", tc.protBranch, err)
			}
			assert.ElementsMatch(t, tc.srcBranches, srcBranches)
			prVars, err := rp.getPRVars(tc.name, tc.testBranch, false)
			if err != nil {
				t.Errorf("Failed to get prVars for %s (%s): %w", tc.name, tc.protBranch, err)
			}
			assert.Equal(t, tc.prVars, prVars)
			maVars, err := rp.getMAVars(tc.name, tc.testBranch)
			if err != nil && err != ErrUnknownBranch {
				t.Errorf("Failed to get maVars for %s(%s): %w", tc.name, tc.protBranch, err)
			}
			// Hack to make the timestamps match
			maVars.Timestamp = timeStamp
			assert.Equal(t, tc.maVars, maVars)
		})
	}
}
