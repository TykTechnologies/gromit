package dhi

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// EqualCustomizations compares complete customization YAML documents while
// ignoring Docker's server-assigned top-level id and formatting differences.
func EqualCustomizations(left, right io.Reader) (bool, error) {
	leftValue, err := decodeCustomization(left)
	if err != nil {
		return false, fmt.Errorf("decode desired customization: %w", err)
	}
	rightValue, err := decodeCustomization(right)
	if err != nil {
		return false, fmt.Errorf("decode remote customization: %w", err)
	}

	leftJSON, err := json.Marshal(leftValue)
	if err != nil {
		return false, fmt.Errorf("canonicalize desired customization: %w", err)
	}
	rightJSON, err := json.Marshal(rightValue)
	if err != nil {
		return false, fmt.Errorf("canonicalize remote customization: %w", err)
	}
	return string(leftJSON) == string(rightJSON), nil
}

func decodeCustomization(reader io.Reader) (map[string]any, error) {
	decoder := yaml.NewDecoder(reader)
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, fmt.Errorf("customization is empty")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("customization contains multiple YAML documents")
		}
		return nil, err
	}

	delete(value, "id")
	return value, nil
}
