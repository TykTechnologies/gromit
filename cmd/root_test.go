package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	code := m.Run()

	os.Exit(code)
}

// Used to hold the result of a command execution
type cmdExecution struct {
	RetCode int
	Stdout  []byte
	Stderr  []byte
}

// executeMock cmd will call a subcommand with args capturing stdout and stderr
func executeMockCmd(args ...string) (*cmdExecution, error) {
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
