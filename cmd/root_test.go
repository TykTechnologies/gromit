package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"github.com/rs/zerolog/log"
	"github.com/joho/godotenv"
)

// setup environment for the test run and cleanup after
func TestMain(m *testing.M) {
	err := godotenv.Load("../testdata/testvars")
	if err != nil {
		log.Error().Err(err).Msg("Could not load env from testdata/testvars")
	}
	path, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("Could not find cwd")
	}
	log.Info().Str("path", path).Msg("cwd")
	
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
