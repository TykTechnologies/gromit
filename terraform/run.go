package terraform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/TykTechnologies/gromit/confgen"
	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/server"
	"github.com/TykTechnologies/gromit/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/kelseyhightower/envconfig"
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

// Apply will validate, plan and apply
func apply(env string, dir string) {
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

	terraformExitOnFailure("apply", "-auto-approve", fmt.Sprintf("-var-file=%s/%s.tfvars.json", dir, env))
}

func setupTerraformCreds(token string) error {
	credFile := fmt.Sprintf("%s/.terraformrc", os.Getenv("HOME"))
	creds := fmt.Sprintf(`credentials "app.terraform.io" {
  token = "%s"
}`, token)
	return ioutil.WriteFile(credFile, []byte(creds), 0600)
}

// Run is the entrypoint from the CLI
func Run(cfg aws.Config, confPath string) error {
	var e server.EnvConfig

	t := time.Now()
	defer func() {
		util.StatTime("run.timetaken", time.Since(t))
	}()
	util.StatCount("run.count", 1)

	// Read env vars prefixed by GROMIT_
	err := envconfig.Process("gromit", &e)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load env")
	}
	log.Info().Interface("env", e).Msg("loaded env")

	err = setupTerraformCreds(os.Getenv("TF_API_TOKEN"))
	if err != nil {
		log.Fatal().Err(err).Msg("unable to setup terraform creds")
	}

	db := dynamodb.New(cfg)
	envs, err := devenv.GetNewEnvs(db, e.TableName, e.Repos)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not get new envs from table %s", e.TableName)
	}
	util.StatGauge("run.nenvs", len(envs))

	procSentinelFile := filepath.Join(confPath, "noprocess")
	if _, err := os.Stat(procSentinelFile); !os.IsNotExist(err) {
		log.Info().Interface("envs", envs).Msgf("Not processing envs as %s exists", procSentinelFile)
		return nil
	}
	log.Trace().Str("sentinelfile", procSentinelFile).Msg("not found")

	var lastError error = nil
	for _, env := range envs {
		log.Info().Interface("env", env).Msg("processing")
		envName := env[devenv.NAME].(string)
		envTime := time.Now()
		defer func() {
			util.StatTime(fmt.Sprintf("run.%s.timetaken", envName), time.Since(envTime))
		}()

		err := confgen.Must(confPath, envName)
		if err != nil {
			util.StatCount("run.failures", 1)
			log.Error().Err(err).Msgf("could not create config tree for env %s", envName)
			continue
		}
		// go.rice only works with string literals
		devManifest := rice.MustFindBox("devenv")
		tfDir, err := deployManifest(devManifest, envName)
		if err != nil {
			util.StatCount("run.failures", 1)
			log.Error().Err(err).Msgf("could not deploy manifest for env %s", envName)
			lastError = err
			continue
		}
		err = makeInputVarfile(tfDir, env)
		if err != nil {
			util.StatCount("run.failures", 1)
			log.Error().Err(err).Msgf("could not write input file for env %s", envName)
			lastError = err
			continue
		}
		apply(envName, tfDir)
		// os.RemoveAll(tfDir)
		// Wait for the apply to catch up before looking for IP addresses
		time.Sleep(1 * time.Minute)
		err = devenv.UpdateClusterIPs(cfg, envName, e.ZoneID, e.Domain)
		if err != nil {
			log.Error().Err(err).Msgf("could not update IPs for env %s", envName)
			util.StatCount("expose.failures", 1)
			lastError = err
			continue
		}
		// Mark env processed so that the runner will not pick it up
		env[devenv.STATE] = devenv.PROCESSED
		err = devenv.UpsertEnv(db, e.TableName, envName, env)
		if err != nil {
			log.Error().Err(err).Str("env", envName).Msg("could not mark env as PROCESSED")
		}
	}
	return lastError
}
