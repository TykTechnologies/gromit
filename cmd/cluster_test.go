package cmd

import (
	"testing"
)

func TestClusterRun(t *testing.T) {
	executeMockCmd("cluster", "run", "/does-not-exist")
}
