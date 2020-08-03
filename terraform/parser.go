package terraform

import (
	"encoding/json"
	"fmt"
	"os"

	rice "github.com/GeertJohan/go.rice"
	"github.com/rs/zerolog/log"
)

// This is how each variable is represented in terraform
type tfVar struct {
	Sensitive bool        `json:"sensitive"`
	VarType   interface{} `json:"type"`
	Value     interface{} `json:"value"`
}

func (tv *tfVar) getMapValue() map[string]interface{} {
	return tv.Value.(map[string]interface{})
}

func (tv *tfVar) getStringValue() string {
	return tv.Value.(string)
}

// TFOutput models the JSON terraform output
type TFOutput map[string]tfVar

// TFInputs can be rendered into a tfvars.json file
type TFInputs map[string]string

// getOutput will init terraform and fetch the output from the state
// that is defined in the manifest
func getOutput(dir string, wrkSpace string) (TFOutput, error) {
	os.Chdir(dir)
	tfOutput := make(TFOutput)
	if len(wrkSpace) < 1 {
		wrkSpace = "default"
	}
	tfEnv := append(os.Environ(),
		fmt.Sprintf("TF_WORKSPACE=%s", wrkSpace))
	terraformInit(tfEnv)
	_, err := terraform("workspace", "select", wrkSpace)
	if err != nil {
		return tfOutput, err
	}

	// This refresh is needed to init the infra state (which uses
	// a prefix to choose envs) correctly
	_, err = terraform("refresh")
	if err != nil {
		return tfOutput, err
	}
	op, err := terraform("output", "-json")
	if err != nil {
		return tfOutput, err
	}
	log.Trace().Str("output", string(op)).Msgf("tf output from %s for workspace %s", dir, wrkSpace)
	if err != nil {
		return tfOutput, err
	}

	err = json.Unmarshal(op, &tfOutput)
	if err != nil {
		return tfOutput, err
	}
	return tfOutput, nil
}

// GetInfraValues will return a TFOuput filled with the combined base
// and infra outputs
func GetInfraValues() (TFOutput, error) {
	base := rice.MustFindBox("base")
	baseDir, err := deployManifest(base, "base")
	if err != nil {
		return TFOutput{}, err
	}
	baseVars, err := getOutput(baseDir, "default")
	if err != nil {
		return baseVars, err
	}
	log.Trace().Interface("base", baseVars).Msg("parsed output from base")
	os.RemoveAll(baseDir)

	infra := rice.MustFindBox("infra")
	infraDir, err := deployManifest(infra, "infra")
	if err != nil {
		return baseVars, err
	}
	// TODO make euc1 an environment variable or part of the config
	infraVars, err := getOutput(infraDir, "euc1")
	if err != nil {
		return baseVars, err
	}
	log.Trace().Interface("infra", infraVars).Msgf("parsed output from infra")
	os.RemoveAll(infraDir)

	// Merge infraVars into baseVars
	for k, v := range infraVars {
		baseVars[k] = v
	}
	return baseVars, err
}
