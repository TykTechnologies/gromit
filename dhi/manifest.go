package dhi

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// RebuildTriggerAnnotation is written by CI solely to force a manifest difference
// and is ignored when comparing customizations.
const RebuildTriggerAnnotation = "io.tyk.rebuild-trigger"

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
	// The release workflow stamps a run id into this annotation purely to make the
	// manifest differ, because Docker offers no way to request a customization
	// rebuild (docker-hardened-images/advisories#2017). It describes nothing about
	// the desired image, so it must not count as a difference here -- otherwise the
	// apply script would see every post-release manifest as changed, edit it to
	// remove the stamp, and that edit would itself queue another rebuild.
	if annotations, ok := value["annotations"].(map[string]any); ok {
		delete(annotations, RebuildTriggerAnnotation)
		if len(annotations) == 0 {
			delete(value, "annotations")
		}
	}
	return value, nil
}
