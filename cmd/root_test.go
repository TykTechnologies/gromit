package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/server"
)

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	os.Setenv("TF_VAR_base", "base-devenv-euc1-test")
	os.Setenv("TF_VAR_infra", "infra-devenv-euc1-test")

	a := server.App{
		TableName:  os.Getenv("GROMIT_TABLENAME"),
		RegistryID: os.Getenv("GROMIT_REGISTRYID"),
		Repos:      strings.Split(os.Getenv("GROMIT_REPOS"), ","),
	}
	err := a.Init([]byte(os.Getenv("GROMIT_CA")), []byte(os.Getenv("GROMIT_SERVE_CERT")), []byte(os.Getenv("GROMIT_SERVE_KEY")))
	if err != nil {
		fmt.Println("could not init test app", err)
		os.Exit(1)
	}

	ts := a.Test()
	defer ts.Close()
	os.Setenv("GROMIT_SERVE_URL", ts.URL)
	code := m.Run()

	os.Exit(code)
}

// Used to hold the result of a command execution
type cmdExecution struct {
	RetCode int
	Stdout  []byte
	Stderr  []byte
}

// executeMock cmd will make an API request to a locally running server
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
