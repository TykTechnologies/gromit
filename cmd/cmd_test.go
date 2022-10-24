package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmdtest"
)

func TestCmd(t *testing.T) {
	if err := exec.Command("go", "build", "..").Run(); err != nil {
		t.Fatalf("error building gromit binary: %v", err)
	}
	//defer os.Remove("./gromit")
	tcDir := "../testdata/cmdtest"
	dirs, err := os.ReadDir(tcDir)
	if err != nil {
		t.Fatal("can't walk the testcas directory")
	}
	for _, dir := range dirs {
		t.Logf("Running tests for: %s", dir.Name())
		ts, err := cmdtest.Read(filepath.Join(tcDir, dir.Name()))
		if err != nil {
			t.Fatalf("error reading testsuite: %v", err)
		}
		ts.Commands["gromit"] = cmdtest.Program("gromit")
		ts.Commands["wait2"] = cmdtest.InProcessProgram("wait", func() int {
			time.Sleep(2 * time.Second)
			return 0
		})
		// t.Log(ts.Commands)
		//ts.Run(t, true)
		// run with true if outputs requires update
		ts.RunParallel(t, true)
	}
}
