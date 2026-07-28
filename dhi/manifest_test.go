package dhi_test

import (
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/dhi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEqualCustomizations(t *testing.T) {
	desired := `
name: compiler
targets:
  - destination: tykio/compiler
    tag_definition_id: busybox/debian-13/1-fips
labels:
  title: compiler
  source: gromit
`
	remote := `
# server-generated file
id: cz_example
name: compiler
targets:
  - destination: tykio/compiler
    tag_definition_id: busybox/debian-13/1-fips
labels:
  source: gromit
  title: compiler
`

	equal, err := dhi.EqualCustomizations(strings.NewReader(desired), strings.NewReader(remote))
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestEqualCustomizationsDetectsChange(t *testing.T) {
	left := "name: compiler\ncontents:\n  packages: [git, gcc]\n"
	right := "id: cz_example\nname: compiler\ncontents:\n  packages: [git, clang]\n"

	equal, err := dhi.EqualCustomizations(strings.NewReader(left), strings.NewReader(right))
	require.NoError(t, err)
	assert.False(t, equal)
}
