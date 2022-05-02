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

// TestPortsPolicy can test a repo with back/forward ports
func TestPortsPolicy(t *testing.T) {
	timeStamp := "2021-06-02 06:47:55.826883255 +0000 UTC"
	cases := []struct {
		cfgFile      string
		srcBranches  []string
		destBranches []string
		prVars       prVars
		maVars       maVars
		name         string
	}{
		{
			cfgFile:      "../testdata/policies/gateway.yaml",
			name:         "tyk",
			srcBranches:  []string{"master"},
			destBranches: []string{"release-4"},
			prVars: prVars{
				RepoName:     "tyk",
				Files:        gatewayFiles,
				SrcBranch:    "master",
				DestBranches: []string{"release-4"},
				Remove:       false,
			},
			maVars: maVars{
				Timestamp: timeStamp,
				MAFiles:   gatewayFiles,
				SrcBranch: "master",
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
			srcBranches, err := rp.SrcBranches(tc.name)
			if err != nil {
				t.Errorf("Failed to get source branches for %s: %v", tc.name, err)
			}
			assert.ElementsMatch(t, tc.srcBranches, srcBranches)
			prVars, err := rp.getPRVars(tc.name, tc.srcBranches[0], false)
			if err != nil {
				t.Errorf("Failed to get prVars for %s (%s): %v", tc.name, tc.srcBranches[0], err)
			}
			assert.Equal(t, tc.prVars, prVars)
			maVars, err := rp.getMAVars(tc.name, tc.srcBranches[0])
			if err != nil && err != ErrUnknownBranch {
				t.Errorf("Failed to get maVars for %s(%s): %v", tc.name, tc.srcBranches[0], err)
			}
			// Hack to make the timestamps match
			maVars.Timestamp = timeStamp
			assert.Equal(t, tc.maVars, maVars)
		})
	}
}
