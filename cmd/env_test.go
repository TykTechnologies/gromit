package cmd

import (
	"testing"
)

func TestPositives(t *testing.T) {
	// Order matters, delete after creating
	cases := []cmdTestCase{
		{
			Name:    "NewTestEnv",
			Args:    []string{"env", "-etest", "new", "-f../testdata/env/new.json"},
			RetCode: 0,
		},
		{
			Name:    "GetTestEnv",
			Args:    []string{"env", "-etest"},
			RetCode: 0,
			// This needs to match the test case that created this env
			ResponseJSON: `{"name":"test","state":"new","tyk":"gw-sha","tyk-analytics":"db-sha","tyk-pump":"pump-sha"}`,
		},
		{
			Name:    "DeleteTestEnv",
			Args:    []string{"env", "-etest", "delete"},
			RetCode: 0,
		},
	}
	runSubTests(t, cases)
}
