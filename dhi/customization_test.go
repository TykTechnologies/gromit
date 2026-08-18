package dhi_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPluginCompilerCustomization(t *testing.T) {
	data, err := os.ReadFile("customizations/plugin-compiler-ng.yaml")
	require.NoError(t, err)

	var manifest struct {
		Name    string `yaml:"name"`
		Targets []struct {
			Destination     string `yaml:"destination"`
			TagDefinitionID string `yaml:"tag_definition_id"`
		} `yaml:"targets"`
		Platforms   []string `yaml:"platforms"`
		Compression string   `yaml:"compression"`
		Contents    struct {
			Packages []string `yaml:"packages"`
		} `yaml:"contents"`
		Labels      map[string]string `yaml:"labels"`
		Annotations map[string]string `yaml:"annotations"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(&manifest))

	assert.Equal(t, "plugin compiler ng toolchain", manifest.Name)
	require.Len(t, manifest.Targets, 1)
	assert.Equal(t, "tykio/dhi-busybox-plugin-compiler", manifest.Targets[0].Destination)
	assert.Equal(t, "busybox/debian-13/1-fips", manifest.Targets[0].TagDefinitionID)
	assert.ElementsMatch(t, []string{"linux/amd64", "linux/arm64"}, manifest.Platforms)
	assert.Equal(t, "GZIP", manifest.Compression)
	assert.Equal(t, "https://github.com/TykTechnologies/gromit",
		manifest.Labels["org.opencontainers.image.source"])
	assert.Equal(t, manifest.Labels["org.opencontainers.image.description"],
		manifest.Annotations["org.opencontainers.image.description"])

	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(data, &raw))
	for _, inherited := range []string{"accounts", "environment", "entrypoint", "cmd", "workdir"} {
		assert.NotContains(t, raw, inherited, "inherit %s from the source DHI", inherited)
	}

	assert.ElementsMatch(t, []string{
		"bash", "jq", "ca-certificates", "coreutils", "sed", "grep",
		"findutils", "tar", "binutils", "binutils-gold", "file", "make",
		"gcc", "gcc-x86-64-linux-gnu", "gcc-aarch64-linux-gnu",
		"gcc-s390x-linux-gnu", "g++", "g++-x86-64-linux-gnu",
		"g++-aarch64-linux-gnu", "g++-s390x-linux-gnu", "git",
		"linux-libc-dev", "expat",
	}, manifest.Contents.Packages)

	info, err := os.Stat("apply-plugin-compiler-customization.sh")
	require.NoError(t, err)
	assert.NotZero(t, info.Mode().Perm()&0o111)

	applyScript, err := os.ReadFile("apply-plugin-compiler-customization.sh")
	require.NoError(t, err)
	assert.Contains(t, string(applyScript), `--arg repository "${destination}"`)
	assert.Contains(t, string(applyScript), `(.repository // "") == $repository`)
}
