package cmd

import (
	"testing"
)

func TestClusterSow(t *testing.T) {
	response, err := executeMockCmd("env", "-ecluster-test", "new", "-f../testdata/env/new.json")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
	response, err = executeMockCmd("cluster", "sow", "-ctest", "../testdata/config")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
}
