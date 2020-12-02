package cmd

import (
	"testing"
)

func TestPositives(t *testing.T) {
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
	}
	runSubTests(t, cases)
}
