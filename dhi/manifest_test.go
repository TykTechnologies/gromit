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

// A release stamps a rebuild-trigger annotation into the remote manifest because
// Docker exposes no way to request a rebuild. If that stamp counted as a
// difference, the apply script would edit the customization to remove it on every
// subsequent run, and each such edit would queue another rebuild -- an endless
// churn of images. It must compare equal.
func TestRebuildTriggerAnnotationIsNotADifference(t *testing.T) {
	desired := strings.NewReader(`
name: example
annotations:
  org.opencontainers.image.title: Example
`)
	stamped := strings.NewReader(`
id: cz_example
name: example
annotations:
  io.tyk.rebuild-trigger: "1234567890"
  org.opencontainers.image.title: Example
`)

	equal, err := dhi.EqualCustomizations(desired, stamped)
	require.NoError(t, err)
	assert.True(t, equal, "a rebuild-trigger stamp must not make the manifest differ")
}

func TestRealAnnotationDifferenceIsStillDetected(t *testing.T) {
	desired := strings.NewReader("name: example\nannotations:\n  a: one\n")
	remote := strings.NewReader("name: example\nannotations:\n  a: two\n")

	equal, err := dhi.EqualCustomizations(desired, remote)
	require.NoError(t, err)
	assert.False(t, equal, "a genuine annotation change must still be a difference")
}
