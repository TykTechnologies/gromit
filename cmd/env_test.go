//go:build disabled

package cmd

import (
	"testing"

	"github.com/TykTechnologies/gromit/server"
	"github.com/stretchr/testify/assert"
)

// Each instance is executed by rootCmd, so Args should contain the subcommand
type cmdTestCase struct {
	Name         string
	Args         []string
	RetCode      int
	ResponseStr  string
	ResponseJSON string
}

func runEnvTests(t *testing.T, cases []cmdTestCase) {
	s, _ := server.StartTestServer("../testdata/env-config.yaml")
	defer s.Close()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			response, err := executeMockCmd(tc.Args...)
			if err != nil {
				t.Error(err)
			}
			//fmt.Printf("captured output: %q\n", response.Stdout)
			checkReturnCode(t, tc.RetCode, response.RetCode)
			if tc.ResponseJSON != "" {
				assert.JSONEq(t, tc.ResponseJSON, string(response.Stdout))
			}
		})
	}
}

func TestEnvCmd(t *testing.T) {
	// Order matters, delete after creating
	cases := []cmdTestCase{
		{
			Name:    "NewTestEnv",
			Args:    []string{"env", "-eenv-test", "new", "--file ../testdata/env/new.json"},
			RetCode: 0,
		},
		{
			Name:    "GetTestEnv",
			Args:    []string{"env", "-eenv-test"},
			RetCode: 0,
			// This needs to match the test case that created this env
			ResponseJSON: `{"name":"env-test","state":"new","tyk":"gw-sha","tyk-analytics":"db-sha","tyk-pump":"pump-sha"}`,
		},
		{
			Name:    "DeleteTestEnv",
			Args:    []string{"env", "-eenv-test", "delete"},
			RetCode: 0,
		},
		{
			Name:    "CheckDeleteTestEnv",
			Args:    []string{"env", "-eenv-test"},
			RetCode: 0,
			// This needs to match the test case that created this env
			ResponseJSON: `{"name":"env-test","state":"deleted","tyk":"gw-sha","tyk-analytics":"db-sha","tyk-pump":"pump-sha"}`,
		},
	}
	runEnvTests(t, cases)
}
