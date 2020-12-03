package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/server"
	"github.com/stretchr/testify/assert"
)

var ts *httptest.Server

const tableName = "GromitCmdTest"

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	os.Setenv("GROMIT_TABLENAME", tableName)
	os.Setenv("GROMIT_REPOS", "tyk,tyk-analytics,tyk-pump")
	os.Setenv("GROMIT_REGISTRYID", "046805072452")
	os.Setenv("XDG_CONFIG_HOME", "../testdata")
	var a server.App
	a.Init("../testdata/ca.pem")
	ts = a.Test("../testdata/scerts/cert.pem", "../testdata/scerts/key.pem")
	defer ts.Close()

	code := m.Run()
	err := devenv.DeleteTable(a.DB, tableName)
	if err != nil {
		fmt.Println(err)
	}

	os.Exit(code)
}

// Each instance is executed by rootCmd, so Args should contain the subcommand
type cmdTestCase struct {
	Name         string
	Args         []string
	RetCode      int
	ResponseStr  string
	ResponseJSON string
}

// Used to hold the result of a command execution
type cmdExecution struct {
	RetCode int
	Stdout  []byte
	Stderr  []byte
}

// executeMock cmd will make an API request to a locally running server
func executeMockCmd(args []string) (*cmdExecution, error) {
	o := new(bytes.Buffer)
	e := new(bytes.Buffer)
	rootCmd.SetOut(o)
	rootCmd.SetErr(e)
	// Add local gromit server to args
	args = append(args, fmt.Sprintf("-s%s", ts.URL))
	rootCmd.SetArgs(args)
	rootCmd.Execute()

	op, err := ioutil.ReadAll(o)
	if err != nil {
		return &cmdExecution{}, err
	}
	eop, err := ioutil.ReadAll(e)
	if err != nil {
		return &cmdExecution{}, err
	}
	return &cmdExecution{
		RetCode: 0,
		Stdout:  op,
		Stderr:  eop,
	}, nil
}

func checkReturnCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected return code %d. Got %d\n", expected, actual)
	}
}

func runSubTests(t *testing.T, cases []cmdTestCase) {
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			response, err := executeMockCmd(tc.Args)
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
