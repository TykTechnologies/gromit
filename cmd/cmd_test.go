package cmd

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmdtest"
)

var update = flag.Bool("update", false, "update the testcases for cmdtest")

func TestCmd(t *testing.T) {
	flag.Parse()
	if err := exec.Command("go", "build", "..").Run(); err != nil {
		t.Fatalf("error building gromit binary: %v", err)
	}
	t.Cleanup(func() {
		t.Log("Cleaning up..")
		os.Remove("./gromit")
	})
	tcDir := "cmdtest"
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
		// set to true for prod - might spew secrets otherwise.
		ts.DisableLogging = true
		ts.Commands["gromit"] = cmdtest.Program("gromit")
		ts.Commands["wait5"] = cmdtest.InProcessProgram("wait", func() int {
			time.Sleep(5 * time.Second)
			return 0
		})
		ts.RunParallel(t, *update)
	}
}
