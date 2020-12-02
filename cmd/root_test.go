package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/TykTechnologies/gromit/server"
	"github.com/stretchr/testify/assert"
)

// package variable for the testing app
var a server.App

// Used to hold the result of a command execution
type cmdExecution struct {
	RetCode int
	Stdout  []byte
	Stderr  []byte
}

// Each instance is executed by rootCmd, so Args should contain the subcommand
type cmdTestCase struct {
	Name         string
	Args         []string
	RetCode      int
	ResponseStr  string
	ResponseJSON string
}

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	os.Setenv("GROMIT_TABLENAME", "GromitTest")
	os.Setenv("GROMIT_REPOS", "tyk,tyk-analytics,tyk-pump")
	os.Setenv("GROMIT_REGISTRYID", "046805072452")
	os.Setenv("XDG_CONFIG_HOME", "../testdata")
	var a server.App
	a.Init("../testdata/ca.pem")
	ts := a.Test("../testdata/scerts/cert.pem", "../testdata/scerts/key.pem")
	defer ts.Close()

	code := m.Run()

	os.Exit(code)
}

func executeCmd(args []string) (*cmdExecution, error) {
	o := new(bytes.Buffer)
	e := new(bytes.Buffer)
	rootCmd.SetOut(o)
	rootCmd.SetErr(e)
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
			response, err := executeCmd(tc.Args)
			if err != nil {
				t.Error(err)
			}
			fmt.Printf("captured output: %q\n", response.Stdout)
			checkReturnCode(t, tc.RetCode, response.RetCode)
			if tc.ResponseJSON != "" {
				assert.JSONEq(t, tc.ResponseJSON, string(response.Stdout))
			}
		})
	}
}
