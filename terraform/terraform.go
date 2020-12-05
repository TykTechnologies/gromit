package terraform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/rs/zerolog/log"
)

// Will return the output and error
func terraform(args ...string) ([]byte, error) {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
		"TF_CLI_ARGS=-no-color",
	)
	tf := exec.Command("terraform", args...)
	tf.Env = tfEnv

	out, err := tf.CombinedOutput()
	return out, err
}

// Will log and exit if there is an error
func terraformExitOnFailure(args ...string) {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
		"TF_CLI_ARGS=-no-color",
	)
	cmd := args[0]

	tf := exec.Command("terraform", args...)
	tf.Env = tfEnv

	out, err := tf.CombinedOutput()
	if err != nil {
		log.Fatal().
			Str("output", string(out)).
			Err(err).
			Msgf("%s failed", cmd)
	}
	log.Trace().Str(cmd, string(out)).Msg(cmd)
}

// terraformInit will fail if there is an error
func terraformInit(tfEnv []string) {
	tfEnv = append(tfEnv,
		"TF_IN_AUTOMATION=1",
	)
	// XXX: read-only so no locks are needed?
	tf := exec.Command("terraform", "init", "-input=false", "-no-color", "-lock=false")
	tf.Env = tfEnv
	tf.Stdin = strings.NewReader("1")

	out, err := tf.CombinedOutput()
	if err != nil {
		log.Fatal().
			Str("output", string(out)).
			Err(err).
			Msg("init failed")
	}
	log.Trace().Str("output", string(out)).Msg("init")
}

// makeInputFromTFState transforms envMap into terraform inputs
// See master.tfvars for a sample inputfile in hcl format
func makeInputVarfile(tfDir string, envMap devenv.DevEnv) error {
	varFile := fmt.Sprintf("%s.tfvars.json", envMap[devenv.NAME].(string))
	varsJSON, err := json.Marshal(envMap)
	if err != nil {
		return err
	}
	os.Chdir(tfDir)
	err = ioutil.WriteFile(varFile, varsJSON, 0644)
	if err != nil {
		return err
	}
	return nil
}

// doTFCmd knows how to do only apply and destroy
func doTFCmd(cmd string, env string, dir string) {
	os.Chdir(dir)

	tfEnv := append(os.Environ(),
		fmt.Sprintf("TF_WORKSPACE=%s", env),
	)
	terraformInit(tfEnv)

	op, err := terraform("workspace", "select", env)
	if err != nil {
		log.Warn().
			Str("output", string(op)).
			Err(err).
			Msg("env select failed, assuming it needs creation")
		terraformExitOnFailure("workspace", "new", env)
	}

	terraformExitOnFailure("validate")

	terraformExitOnFailure(cmd, "-auto-approve", fmt.Sprintf("-var-file=%s/%s.tfvars.json", dir, env))
}

// terraformCreds sets up the environment for terraform to run
func terraformCreds(token string) error {
	credFile := fmt.Sprintf("%s/.terraformrc", os.Getenv("HOME"))
	creds := fmt.Sprintf(`credentials "app.terraform.io" {
  token = "%s"
}`, token)
	err := ioutil.WriteFile(credFile, []byte(creds), 0600)
	if err != nil {
		return err
	}
	return nil
}
