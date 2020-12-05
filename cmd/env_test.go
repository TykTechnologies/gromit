package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/TykTechnologies/gromit/devenv"
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

const tableName = "GromitCmdTest"

func runSubTests(t *testing.T, cases []cmdTestCase, tsurl string) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			response, err := executeMockCmd(append(tc.Args, fmt.Sprintf("-s%s", tsurl))...)
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
			Args:    []string{"env", "-eenv-test", "new", "-f../testdata/env/new.json"},
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

	os.Setenv("GROMIT_TABLENAME", tableName)
	var a server.App
	err := devenv.DeleteTable(a.DB, tableName)
	if err != nil {
		t.Fatal(err)
	}
	a.Init("../testdata/ca.pem")
	ts := a.Test("../testdata/scerts/cert.pem", "../testdata/scerts/key.pem")
	defer ts.Close()

	runSubTests(t, cases, ts.URL)
}
