package policy

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/TykTechnologies/gromit/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPluginCompilerNGReleaseSupplyChainMetadata(t *testing.T) {
	var pol Policies
	config.LoadConfig("")
	require.NoError(t, LoadRepoPolicies(&pol))

	repo, err := pol.GetRepoPolicy("tyk")
	require.NoError(t, err)
	require.NoError(t, repo.SetBranch("master"))

	bundle, err := NewBundle([]string{"plugin-compiler-ng"})
	require.NoError(t, err)

	outputDir := t.TempDir()
	_, err = bundle.Render(repo, outputDir, nil)
	require.NoError(t, err)

	buildWorkflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-build.yml")
	assert.Contains(t, buildWorkflow, `NG_SLIM: "0"`)
	assert.Contains(t, buildWorkflow,
		"tykio/tyk-plugin-compiler,enable=${{ startsWith(github.ref, 'refs/tags/') }}")
	assert.Contains(t, buildWorkflow, "id: build-ng-base")
	assert.Contains(t, buildWorkflow,
		`base="tykio/tyk-plugin-compiler@${{ steps.build-ng-base.outputs.digest }}"`)
	assert.Contains(t, buildWorkflow,
		`base="${{ steps.login-ecr.outputs.registry }}/tyk-plugin-compiler@${{ steps.build-ng-base.outputs.digest }}"`)
	assert.Contains(t, buildWorkflow, `BASE_IMAGE=${{ steps.source-base.outputs.ref }}`)
	assert.Equal(t, 1, strings.Count(buildWorkflow, "docker/setup-buildx-action@"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "latest=false"))
	assert.Equal(t, 4, strings.Count(buildWorkflow, "sbom: true"))
	assert.Equal(t, 4, strings.Count(buildWorkflow, "provenance: mode=max"))
	assert.Equal(t, 2, strings.Count(buildWorkflow, "contents: read"))
	assertPluginCompilerSelfTestBuilds(t, buildWorkflow, 3)

	releaseDockerfile := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/Dockerfile.release")
	assert.Equal(t, 1, strings.Count(releaseDockerfile, "ARG WITH_GATEWAY_SELFTEST=0"))
	assert.Contains(t, releaseDockerfile, "test ! -e /usr/local/bin/tyk")

	baseWorkflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-base.yml")
	assert.Contains(t, baseWorkflow, "NG_BASE_SOURCE: dhi.io/busybox:1-debian-fips-dev")
	assert.Contains(t, baseWorkflow, `BASE_IMAGE=${{ steps.source-base.outputs.ref }}`)
	assert.Equal(t, 1, strings.Count(baseWorkflow, "docker/setup-buildx-action@"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "sbom: true"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "provenance: mode=max"))
	assert.Equal(t, 1, strings.Count(baseWorkflow, "contents: read"))

	baseDockerfile := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/Dockerfile.base")
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib/libpython"*.so*`)
	assert.Contains(t, baseDockerfile,
		`test -z "$(find "$SR/usr/lib" -maxdepth 1 -name 'libpython*.so*' -print -quit)"`)
	assert.Equal(t, 2, strings.Count(baseDockerfile, "ARG BASE_IMAGE"))
	assert.Contains(t, baseDockerfile, `test -z "$(dpkg --audit)"`)
	assert.Contains(t, baseDockerfile, "apt-get check")
	assert.NotContains(t, baseDockerfile, "dpkg --purge --force-all")
	assert.NotContains(t, baseDockerfile, "rm -f \"/var/lib/dpkg/status.d/")
	assert.Contains(t, baseDockerfile,
		"COPY data/rewrite-imports.go /usr/local/lib/tyk-plugin-compiler/rewrite-imports.go")

	buildScript := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/data/build.sh")
	assert.Contains(t, buildScript, `plugin_name must be a file basename, not a path`)
	assert.Contains(t, buildScript, `[ "$(basename "$source_entry")" != ".git" ]`)
	assert.Contains(t, buildScript,
		"GO111MODULE=off go run /usr/local/lib/tyk-plugin-compiler/rewrite-imports.go")
	assert.Contains(t, buildScript,
		`drop_godebug && /^[[:space:]]*godebug[[:space:]]+/ { next }`)
	assert.Contains(t, buildScript,
		"removing plugin godebug key unsupported by Gateway Go")
	assert.Contains(t, buildScript,
		"GOWORK=off GO111MODULE=on go list .")
	assert.Contains(t, buildScript,
		`drop_ignore && /^[[:space:]]*ignore[[:space:]]+/ { next }`)
	assert.NotContains(t, buildScript, `sed -i -e "s,\"${OLD_MODULE}`)

	rewriteImports := readRenderedFile(t, outputDir,
		"ci/images/plugin-compiler-ng/data/rewrite-imports.go")
	assert.Contains(t, rewriteImports, "parser.ImportsOnly")

	validator := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/data/validate-plugin.sh")
	assert.Contains(t, validator, "unable to inspect ELF dynamic dependencies")
	assert.Contains(t, validator, "unable to inspect ELF version requirements")
	assert.Contains(t, validator, "the official Gateway image does not ship libpython")
}

func TestPluginCompilerNGValidatorRejectsUncheckedOrPythonLinkedPlugin(t *testing.T) {
	validator, err := filepath.Abs(filepath.Join(
		"templates", "plugin-compiler-ng", "ci", "images", "plugin-compiler-ng",
		"data", "validate-plugin.sh",
	))
	require.NoError(t, err)

	tests := []struct {
		name               string
		dynamic            string
		dynamicReadelfFail bool
		versionReadelfFail bool
		wantFailure        string
	}{
		{
			name: "supported glibc dependency",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
		},
		{
			name:               "dynamic inspection fails closed",
			dynamicReadelfFail: true,
			wantFailure:        "unable to inspect ELF dynamic dependencies",
		},
		{
			name:        "missing dynamic section fails closed",
			dynamic:     "There is no dynamic section in this file.",
			wantFailure: "readelf did not report a valid ELF dynamic section",
		},
		{
			name: "libpython dependency is unsupported",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 2 entries:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
				" 0x0000000000000001 (NEEDED) Shared library: [libpython2.7.so.1.0]",
			}, "\n"),
			wantFailure: "the official Gateway image does not ship libpython",
		},
		{
			name: "path-qualified libpython dependency is unsupported",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 2 entries:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
				" 0x0000000000000001 (NEEDED) Shared library: [/opt/lib/libpython2.7.so.1.0]",
			}, "\n"),
			wantFailure: "the official Gateway image does not ship libpython",
		},
		{
			name: "version inspection fails closed",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			versionReadelfFail: true,
			wantFailure:        "unable to inspect ELF version requirements",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeBin := t.TempDir()
			writeExecutable(t, filepath.Join(fakeBin, "file"), `#!/bin/sh
echo "ELF 64-bit LSB shared object"
`)
			writeExecutable(t, filepath.Join(fakeBin, "go"), `#!/bin/sh
if [ "$1" = "version" ] && [ "${2:-}" = "-m" ]; then
  printf '%s: go1.25.12\n' "$3"
else
  echo "go version go1.25.12 linux/amd64"
fi
`)

			readelf := `#!/bin/sh
case "$1" in
  -h) echo "  Machine: Advanced Micro Devices X86-64" ;;
  -d)
`
			if test.dynamicReadelfFail {
				readelf += "    exit 2\n"
			} else {
				readelf += "    cat <<'EOF'\n" + test.dynamic + "\nEOF\n"
			}
			readelf += `    ;;
  --version-info)
`
			if test.versionReadelfFail {
				readelf += "    exit 2\n"
			} else {
				readelf += "    echo 'No version information found in this file.'\n"
			}
			readelf += `    ;;
esac
`
			writeExecutable(t, filepath.Join(fakeBin, "readelf"), readelf)

			artifact := filepath.Join(t.TempDir(), "plugin.so")
			require.NoError(t, os.WriteFile(artifact, []byte("fixture"), 0o600))

			cmd := exec.Command("/bin/bash", validator, artifact)
			cmd.Env = []string{
				"PATH=" + fakeBin + string(os.PathListSeparator) +
					"/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
				"EXPECT_EDITION=ce",
			}
			output, runErr := cmd.CombinedOutput()
			if test.wantFailure == "" {
				require.NoError(t, runErr, string(output))
				assert.Contains(t, string(output), "validation OK")
				return
			}
			require.Error(t, runErr, string(output))
			assert.Contains(t, string(output), test.wantFailure)
		})
	}
}

func TestPluginCompilerNGBuildScriptRejectsUnsafeInputs(t *testing.T) {
	buildScript, err := filepath.Abs(filepath.Join(
		"templates", "plugin-compiler-ng", "ci", "images", "plugin-compiler-ng",
		"data", "build.sh",
	))
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		wantFailure string
	}{
		{
			name:        "plugin name path traversal",
			args:        []string{"../../tmp/plugin.so", "", "linux", "amd64"},
			wantFailure: "plugin_name must be a file basename",
		},
		{
			name:        "plugin id path traversal",
			args:        []string{"plugin.so", "../../tmp", "linux", "amd64"},
			wantFailure: "plugin_id must not contain a path separator",
		},
		{
			name:        "unsafe goos",
			args:        []string{"plugin.so", "", "../linux", "amd64"},
			wantFailure: "GOOS must contain only",
		},
		{
			name:        "unsafe goarch",
			args:        []string{"plugin.so", "", "linux", "amd64/../../tmp"},
			wantFailure: "GOARCH must contain only",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{buildScript}, test.args...)
			cmd := exec.Command("/bin/bash", args...)
			output, runErr := cmd.CombinedOutput()
			require.Error(t, runErr, string(output))
			assert.Contains(t, string(output), test.wantFailure)
		})
	}
}

func TestPluginCompilerNGRewriteImportsUsesGoSyntax(t *testing.T) {
	helper, err := filepath.Abs(filepath.Join(
		"templates", "plugin-compiler-ng", "ci", "images", "plugin-compiler-ng",
		"data", "rewrite-imports.go",
	))
	require.NoError(t, err)

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "main.go")
	source := `package main

import (
	"example.com/a&b/pkg"
	alias "example.com/a&b"
	"example.com/a&b-extra/unchanged"
)

var notAnImport = "example.com/a&b/pkg"
var _ = alias.Value
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o740))
	for _, explicitImportDir := range []string{"testdata", ".hidden", "_tooling"} {
		dir := filepath.Join(sourceDir, explicitImportDir)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "valid.go"),
			[]byte("package explicit\nimport \"example.com/a&b/pkg\"\n"),
			0o640,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "broken.go"),
			[]byte("package broken\nimport (\n"),
			0o640,
		))
	}
	for _, ignoredDir := range []string{"vendor", ".git"} {
		dir := filepath.Join(sourceDir, ignoredDir)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "broken.go"),
			[]byte("package broken\nimport (\n"),
			0o640,
		))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(sourceDir, "_ignored.go"),
		[]byte("package broken\nimport (\n"),
		0o640,
	))

	cmd := exec.Command(
		filepath.Join(runtime.GOROOT(), "bin", "go"), "run", helper,
		"example.com/a&b", "tyk.internal/tyk_plugin-safe", sourceDir,
	)
	cmd.Env = []string{
		"CGO_ENABLED=0",
		"GOCACHE=" + t.TempDir(),
		"GO111MODULE=off",
		"GOROOT=" + runtime.GOROOT(),
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
	}
	output, runErr := cmd.CombinedOutput()
	require.NoError(t, runErr, string(output))

	rewrittenBytes, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	rewritten := string(rewrittenBytes)
	assert.Contains(t, rewritten, `"tyk.internal/tyk_plugin-safe/pkg"`)
	assert.Contains(t, rewritten, `alias "tyk.internal/tyk_plugin-safe"`)
	assert.Contains(t, rewritten, `"example.com/a&b-extra/unchanged"`)
	assert.Contains(t, rewritten, `var notAnImport = "example.com/a&b/pkg"`)
	for _, explicitImportDir := range []string{"testdata", ".hidden", "_tooling"} {
		validBytes, err := os.ReadFile(filepath.Join(sourceDir, explicitImportDir, "valid.go"))
		require.NoError(t, err)
		assert.Contains(t, string(validBytes), `"tyk.internal/tyk_plugin-safe/pkg"`)

		brokenBytes, err := os.ReadFile(filepath.Join(sourceDir, explicitImportDir, "broken.go"))
		require.NoError(t, err)
		assert.Equal(t, "package broken\nimport (\n", string(brokenBytes))
	}
	for _, ignoredPath := range []string{
		filepath.Join(sourceDir, "vendor", "broken.go"),
		filepath.Join(sourceDir, ".git", "broken.go"),
		filepath.Join(sourceDir, "_ignored.go"),
	} {
		ignoredBytes, err := os.ReadFile(ignoredPath)
		require.NoError(t, err)
		assert.Equal(t, "package broken\nimport (\n", string(ignoredBytes))
	}

	info, err := os.Stat(sourcePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o740), info.Mode().Perm())
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o700))
}

func TestGatewayReleaseSupplyChainMetadata(t *testing.T) {
	var pol Policies
	config.LoadConfig("")
	require.NoError(t, LoadRepoPolicies(&pol))

	repo, err := pol.GetRepoPolicy("tyk")
	require.NoError(t, err)
	require.NoError(t, repo.SetBranch("master"))

	bundle, err := NewBundle([]string{"releng"})
	require.NoError(t, err)

	outputDir := t.TempDir()
	_, err = bundle.Render(repo, outputDir, nil)
	require.NoError(t, err)

	releaseWorkflow := readRenderedFile(t, outputDir, ".github/workflows/release.yml")
	assert.Equal(t, 6, strings.Count(releaseWorkflow, "provenance: mode=max"))
	assert.Equal(t, 6, strings.Count(releaseWorkflow, "sbom: true"))
	assert.NotContains(t, releaseWorkflow, "Attach base image VEX")
	assert.NotContains(t, releaseWorkflow, "cosign attest --yes --type openvex")
	assert.NotContains(t, releaseWorkflow, "sigstore/cosign-installer")
}

func TestPluginCompilerNGReleaseBranchConfiguration(t *testing.T) {
	tests := []struct {
		branch           string
		goImage          string
		variantCount     int
		usesBoringCrypto bool
	}{
		{branch: "release-5.3", goImage: "1.23-bullseye", variantCount: 1},
		{branch: "release-5.8", goImage: "1.25-bullseye", variantCount: 3, usesBoringCrypto: true},
		{branch: "release-5.8.15", goImage: "1.25-bullseye", variantCount: 3, usesBoringCrypto: true},
		{branch: "release-5.13", goImage: "1.25-bullseye", variantCount: 3},
	}

	for _, test := range tests {
		t.Run(test.branch, func(t *testing.T) {
			var pol Policies
			config.LoadConfig("")
			require.NoError(t, LoadRepoPolicies(&pol))

			repo, err := pol.GetRepoPolicy("tyk")
			require.NoError(t, err)
			require.NoError(t, repo.SetBranch(test.branch))

			bundle, err := NewBundle([]string{"plugin-compiler-ng"})
			require.NoError(t, err)

			outputDir := t.TempDir()
			_, err = bundle.Render(repo, outputDir, nil)
			require.NoError(t, err)

			workflow := readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-build.yml")
			assert.Contains(t, workflow, "GOLANG_CROSS: "+test.goImage)
			assert.Contains(t, workflow, `NG_SLIM: "0"`)
			assert.Equal(t, test.variantCount, strings.Count(workflow, "latest=false"))
			assertPluginCompilerSelfTestBuilds(t, workflow, test.variantCount)
			if test.usesBoringCrypto {
				assert.Contains(t, workflow, "FIPS_GOEXPERIMENT=boringcrypto")
				assert.Contains(t, workflow, "SELFTEST_GOEXPERIMENT=boringcrypto")
			} else {
				assert.NotContains(t, workflow, "boringcrypto")
			}
		})
	}
}

func assertPluginCompilerSelfTestBuilds(t *testing.T, workflow string, variantCount int) {
	t.Helper()

	var rendered struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string         `yaml:"name"`
				With map[string]any `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(workflow), &rendered))

	job, ok := rendered.Jobs["docker-build"]
	require.True(t, ok)

	var gateBuilds, pushedBuilds int
	for _, step := range job.Steps {
		buildArgs, _ := step.With["build-args"].(string)
		switch {
		case strings.HasPrefix(step.Name, "Build local NG gate image"):
			gateBuilds++
			assert.Equal(t, false, step.With["push"])
			assert.Equal(t, true, step.With["load"])
			assert.Contains(t, buildArgs, "WITH_GATEWAY_SELFTEST=1")
			assert.NotContains(t, buildArgs, "WITH_GATEWAY_SELFTEST=0")
		case strings.HasPrefix(step.Name, "Build and push NG to dockerhub/ECR"):
			pushedBuilds++
			assert.Equal(t, true, step.With["push"])
			assert.Contains(t, buildArgs, "WITH_GATEWAY_SELFTEST=0")
			assert.NotContains(t, buildArgs, "WITH_GATEWAY_SELFTEST=1")
		}
	}

	assert.Equal(t, variantCount, gateBuilds)
	assert.Equal(t, variantCount, pushedBuilds)
}

func readRenderedFile(t *testing.T, outputDir, name string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(contents)
}
