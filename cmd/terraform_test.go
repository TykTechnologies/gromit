package cmd

import (
	"testing"
)

func TestTerraform(t *testing.T) {
	response, err := executeMockCmd("env", "-ecluster-test", "new", "-f../testdata/env/new.json")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
	response, err = executeMockCmd("cluster", "sow", "../testdata/config")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
	response, err = executeMockCmd("env", "-ecluster-test", "rm")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
	response, err = executeMockCmd("cluster", "reap", "../testdata/config")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
}
