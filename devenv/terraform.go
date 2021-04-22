package devenv

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"time"

	"github.com/TykTechnologies/gromit/confgen"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
)

//go:embed terraform
var devManifest embed.FS

// tfInit will
// 1. setup credentials to access state in TF Cloud
// 2. Create a config tree for the environment rooted at confPath
// 3. Deploy the embedded terraform manifests to a temporary directory
// 4. Return a tfRunner that can used to operate upon the terraform manifests deployed in (3)
func (d *DevEnv) tfInit(confPath string) tfRunner {
	token := os.Getenv("TF_API_TOKEN")
	if len(token) < 1 {
		util.StatCount("run.failures", 1)
		log.Fatal().Str("env", d.Name).Msg("TF_API_TOKEN not found in env")
	}
	log.Info().Interface("env_data", d).Msg("processing")

	procSentinelFile := filepath.Join(confPath, "noprocess")
	if _, err := os.Stat(procSentinelFile); !os.IsNotExist(err) {
		log.Fatal().Msgf("%s exists", procSentinelFile)
	}
	log.Trace().Str("sentinelfile", procSentinelFile).Msg("not found")

	err := confgen.Must(confPath, d.Name)
	if err != nil {
		util.StatCount("run.failures", 1)
		log.Error().Err(err).Str("env", d.Name).Msg("could not create config tree")
	}
	tfDir, err := deployManifest(devManifest, d.Name)
	if err != nil {
		util.StatCount("run.failures", 1)
		log.Error().Err(err).Str("env", d.Name).Msg("could not deploy manifest")
	}
	err = d.makeInputVarfile(tfDir)
	if err != nil {
		util.StatCount("run.failures", 1)
		log.Error().Err(err).Str("env", d.Name).Msgf("could not write input file")
	}
	return tfRunner{
		env:   d.Name,
		dir:   tfDir,
		token: token,
	}
}

// Sow will run an deploy an env with terraform
func (d *DevEnv) Sow(confPath string) error {
	log.Info().Str("envName", d.Name).Msg("starting")

	t := time.Now()
	defer func() {
		util.StatTime("run.timetaken", time.Since(t))
	}()

	util.StatCount("run.count", 1)
	tf := d.tfInit(confPath)
	tf.Apply()
	// os.RemoveAll(tfDir)
	// Wait for the apply to catch up before looking for IP addresses
	time.Sleep(1 * time.Minute)
	// Mark env processed so that the runner will not pick it up
	d.MarkProcessed()
	return d.Save()
}

// Reap will destroy an env that was created with Sow() and delete it from the DB
func (d *DevEnv) Reap(confPath string) error {
	log.Info().Str("envName", d.Name).Msg("starting")

	t := time.Now()
	defer func() {
		util.StatTime("reap.timetaken", time.Since(t))
	}()

	util.StatCount("reap.count", 1)
	tf := d.tfInit(confPath)
	tf.Destroy()
	return d.Delete()
}

// makeInputVarfile transforms verions into terraform inputs
// See master.tfvars for a sample inputfile in hcl format
func (d *DevEnv) makeInputVarfile(tfDir string) error {
	varFile := fmt.Sprintf("%s.tfvars.json", d.Name)
	inputMap := make(map[string]string)
	for k, v := range d.versions {
		inputMap[k] = v
	}
	inputMap["base"] = os.Getenv("GROMIT_BASE")
	inputMap["infra"] = os.Getenv("GROMIT_INFRA")
	inputMap["name"] = d.Name

	varsJSON, err := json.Marshal(inputMap)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(tfDir, varFile), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(varsJSON)
	if err != nil {
		return err
	}
	return nil
}

// dest is always treated as a directory name
func copyEmbedDir(fs embed.FS, src string, dest string) error {
	entries, err := fs.ReadDir(src)
	if err != nil {
		return err
	}
	os.MkdirAll(dest, 0755)

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		destPath := filepath.Join(dest, e.Name())

		log.Trace().Msgf("Copying %s to %s", srcPath, destPath)

		if e.IsDir() {
			// Recursively call copyDir()
			if e.Name() == ".terraform" || e.Name() == "terraform.tfstate.d" {
				log.Debug().Msg("skipping terraform dir")
				continue
			}
			copyEmbedDir(fs, srcPath, destPath)
		} else {
			// e is a file
			f, err := fs.Open(srcPath)
			if err != nil {
				return err
			}
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(destPath, data, 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// deployManifests to a temporary dir prefixed with destPrefix
func deployManifest(fs embed.FS, destPrefix string) (string, error) {
	tmpDir, err := ioutil.TempDir("", destPrefix)
	if err != nil {
		return "", err
	}

	err = copyEmbedDir(fs, "terraform", tmpDir)
	if err != nil {
		log.Fatal().Err(err).Str("dest", tmpDir).Msg("could not restore embedded manifests")
	}
	return tmpDir, nil
}
