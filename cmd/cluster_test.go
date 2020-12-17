package cmd

import (
	"os"
	"testing"
)

const clusterTestTableName = "GromitClusterTest"

func TestClusterSow(t *testing.T) {
	os.Setenv("GROMIT_TABLENAME", clusterTestTableName)
	response, err := executeMockCmd("env", "-ecluster-test", "new", "-f../testdata/env/new.json")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
	response, err = executeMockCmd("cluster", "sow", "../testdata/config", "-c test")
	if err != nil {
		t.Fatal(err)
	}
	checkReturnCode(t, 0, response.RetCode)
}
