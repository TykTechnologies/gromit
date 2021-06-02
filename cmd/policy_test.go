package cmd

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicies(t *testing.T) {
	response, err := executeMockCmd("--conf=../testdata/policy-config.yaml", "policy", "--json")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ioutil.ReadFile("../testdata/policy.json")
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(string(response.Stdout))
	require.JSONEq(t, string(expected), string(response.Stdout))
	checkReturnCode(t, 0, response.RetCode)
}
