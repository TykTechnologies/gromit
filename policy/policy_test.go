package policy

import (
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
)

var gatewayFiles = []string{
	"ci/*",
	"tyk.conf.example",
}

// TestPolicyConfig can test a repo with back/forward ports
func TestPolicyConfig(t *testing.T) {
	timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"

	repoPol := repoPolicy{
		Files: []string{"tyk.conf.example"},
		Ports: map[string][]string{
			"master":    []string{"release-4"},
			"release-3": []string{"release-3-lts"},
		},
		Protected: []string{"release-3-lts"},
	}

	pol := Policy{
		Protected: []string{"master"},
		Repos: map[string]repoPolicy{
			"tyk": repoPol,
		},
		Files: []string{"ci/*"},
		Ports: nil,
	}

	cases := []struct {
		cfgFile string
		policy  Policy
		maVars  maVars
		prVars  prVars
		name    string
	}{
		{
			cfgFile: "../testdata/policies/gateway.yaml",
			policy:  pol,
			name:    "tyk",
			prVars: prVars{
				Files:        gatewayFiles,
				RepoName:     "tyk",
				SrcBranch:    "master",
				DestBranches: []string{"release-4"},
			},
			maVars: maVars{
				Timestamp: timeStamp,
				MAFiles:   gatewayFiles,
				SrcBranch: "master",
			},
		},
	}
	var rp Policy
	for _, tc := range cases {
		t.Run(tc.name, func(T *testing.T) {
			config.LoadConfig(tc.cfgFile)
			err := LoadRepoPolicies(&rp)
			if err != nil {
				t.Fatalf("Could not load policy for %s: %v", tc.name, err)
			}

			// Test if config is parsed correctly - check equality between the test case, and
			// the parsed config file.
			assert.EqualValues(t, tc.policy, rp)

			// Test if the template vars get parsed correctly.
			p, err := rp.getPRVars(tc.name, tc.prVars.SrcBranch)
			if err != nil {
				t.Errorf("Failed to get prVars for %s (%s): %v", tc.name, tc.prVars.SrcBranch, err)
			}
			assert.EqualValues(t, tc.prVars, p)

			//maVars, err := rp.getMAVars(tc.name, tc.srcBranches[0])
			//if err != nil && err != ErrUnknownBranch {
			//	t.Errorf("Failed to get maVars for %s(%s): %v", tc.name, tc.srcBranches[0], err)
			//}
			//for _, p := range tc.protected {
			//	prStatus, err := rp.IsProtected(tc.name, p)
			//	if err != nil {
			//		t.Errorf("Failed to get IsProtected status for repo %s, branch %s: %v", tc.name, p, err)
			//	}
			//	assert.Equal(t, prStatus, true)
			//}
			//// Hack to make the timestamps match
			//maVars.Timestamp = timeStamp
			//assert.Equal(t, tc.maVars, maVars)
		})
	}
}
