package devenv

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// Directory to perform the terraform run in
type tfRunner struct {
	env   string
	dir   string
	token string
}

// terraform can be used to run any terraform command that does not
// require interactive input
func (t *tfRunner) runCmd(args ...string) error {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
		"TF_CLI_ARGS=-no-color",
	)
	chdir := fmt.Sprintf("-chdir=%s", t.dir)
	args = append([]string{chdir}, args...)
	tf := exec.Command("terraform", args...)
	tf.Env = tfEnv

	out, err := tf.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed with error: %w, output was: %s", err, out)
	}
	log.Trace().Str("output", string(out)).Msg("init")
	return nil
}

// terraformInit needs to use the same directory as runCmd
func (t *tfRunner) init() error {
	credFile := fmt.Sprintf("%s/.terraformrc", os.Getenv("HOME"))
	creds := fmt.Sprintf(`credentials "app.terraform.io" {
  token = "%s"
}`, t.token)
	err := ioutil.WriteFile(credFile, []byte(creds), 0600)
	if err != nil {
		return err
	}
	chdir := fmt.Sprintf("-chdir=%s", t.dir)
	// XXX: read-only so no locks are needed?
	tf := exec.Command("terraform", chdir, "init", "-input=false", "-no-color", "-lock=false")
	tf.Stdin = strings.NewReader("1")

	out, err := tf.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed with error: %w, output was: %s", err, out)
	}
	log.Trace().Str("output", string(out)).Msg("init")
	return nil
}

// doTFCmd knows how to apply and destroy and will barf at the first sign of trouble
// It does not return error as it uses log.Fatal()
func (t *tfRunner) doTFCmd(cmd string) {
	if err := t.init(); err != nil {
		log.Fatal().Err(err).Msg("init failed")
	}
	if err := t.runCmd("workspace", "select", t.env); err != nil {
		log.Warn().Err(err).Msg("env select failed, assuming it needs creation")
		if err = t.runCmd("workspace", "new", t.env); err != nil {
			log.Fatal().Err(err).Msg("could not create new env either")
		}
	}
	if err := t.runCmd("validate"); err != nil {
		log.Fatal().Err(err).Msg("validate failed")
	}
	varFile := fmt.Sprintf("-var-file=%s.tfvars.json", t.env)
	if err := t.runCmd(cmd, "-auto-approve", varFile); err != nil {
		log.Fatal().Err(err).Str("cmd", cmd).Msg("failed")
	}
}

func (t *tfRunner) Apply() {
	t.doTFCmd("apply")
}

func (t *tfRunner) Destroy() {
	t.doTFCmd("destroy")
}
