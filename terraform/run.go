package terraform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	rice "github.com/GeertJohan/go.rice"
	"github.com/rs/zerolog/log"
)

// Will return the output and error
func terraform(args ...string) (string, error) {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
	)
	noColourCmdLine := append(args, "-no-color")
	tf := exec.Command("terraform", noColourCmdLine...)
	tf.Env = tfEnv

	out, err := tf.CombinedOutput()
	return string(out), err
}

// Will immediately exit if there is an error
func terraformWrapped(args ...string) {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
	)
	cmd := args[0]

	noColourCmdLine := append(args, "-no-color")
	tf := exec.Command("terraform", noColourCmdLine...)
	tf.Env = tfEnv

	out, err := tf.CombinedOutput()
	if err != nil {
		log.Fatal().
			Str("output", string(out)).
			Err(err).
			Msgf("%s failed", cmd)
	}
	log.Debug().Str(cmd, string(out)).Msg(cmd)
}

func terraformInit(env string) {
	tfEnv := append(os.Environ(),
		"TF_IN_AUTOMATION=1",
		fmt.Sprintf("TF_WORKSPACE=%s", env),
	)
	tf := exec.Command("terraform", "init", "-input=false")
	tf.Env = tfEnv
	tf.Stdin = strings.NewReader("1")

	out, err := tf.CombinedOutput()
	if err != nil {
		log.Fatal().
			Str("output", string(out)).
			Err(err).
			Msg("init failed")
	}
	log.Debug().Str("output", string(out)).Msg("init")
}

// Apply will validate, plan and apply
func Apply(env string, dir string) {
	os.Chdir(dir)

	terraformInit(env)

	op, err := terraform("workspace", "select", env)
	if err != nil {
		log.Warn().
			Str("output", op).
			Err(err).
			Msg("env select failed, assuming it needs creation")
		terraformWrapped("workspace", "new", env)
	}

	terraformWrapped("validate")

	terraformWrapped("plan", "-out=tfplan", fmt.Sprintf("-var-file=%s.tfvars", env))

	terraformWrapped("apply", "tfplan")
}

// Run is the entrypoint from the CLI
func Run(env string) error {
	// go.rice only works with string literals
	manifests := rice.MustFindBox("manifests")
	tfDir, err := deployManifests(manifests, env)
	if err != nil {
		return err
	}

	Apply(env, tfDir)
	return os.RemoveAll(tfDir)
}
