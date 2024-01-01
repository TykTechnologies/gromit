package policy

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type validator int64

const (
	UNKNOWN_VALIDATOR validator = iota
	GORELEASER
	GHA
)

//go:embed schemas/github-workflow.json
var gha []byte

//go:embed schemas/goreleaser.json
var gorel []byte

type validatorMap map[validator]*jsonschema.Schema

// LoadValidator loads the machinery required for schema validation
// and formatting.
func loadValidators() (validatorMap, error) {
	schemaMap := make(validatorMap)

	c := jsonschema.NewCompiler()
	if err := c.AddResource("gha.json", bytes.NewReader(gha)); err != nil {
		return schemaMap, fmt.Errorf("could not load gha schema: %v", err)
	}
	schemaMap[GHA] = c.MustCompile("gha.json")
	if err := c.AddResource("gorel.json", bytes.NewReader(gorel)); err != nil {
		return schemaMap, fmt.Errorf("could not load gorel schema: %v", err)
	}
	schemaMap[GORELEASER] = c.MustCompile("gorel.json")

	return schemaMap, nil
}

// validator returns a const representing validator that can be
// applied to this file. path is an array of the path components.
func getValidator(path []string) validator {
	n := len(path) - 1
	if n > 3 && path[n-1] == "workflow" && path[n-2] == ".github" {
		return GHA
	}
	gr := regexp.MustCompile("goreleaser(-el7)?\\.yml")
	if n > 1 && gr.MatchString(path[n]) {
		return GORELEASER
	}
	return UNKNOWN_VALIDATOR
}
