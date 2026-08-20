package policy

import (
	"os"
	"os/exec"
	"path/filepath"
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
		"NG_BASE_SOURCE: tykio/dhi-busybox-plugin-compiler:"+
			"1.37.0-debian13-fips_plugin-compiler-ng-toolchain")
	assert.NotContains(t, buildWorkflow,
		"NG_BASE_SOURCE: tykio/dhi-busybox-plugin-compiler:"+
			"1.37.0-debian13-fips_plugin-compiler-ng-toolchain@sha256:")
	// There is no separate pull-request base. PRs build the same DHI
	// customization the release builds, so the compile/load gate validates the
	// artifact that actually ships.
	assert.NotContains(t, buildWorkflow, "NG_PR_BASE_SOURCE")
	assert.NotContains(t, buildWorkflow, "debian:bookworm-slim")
	// That base is a private Docker Hub repo, so pull requests need the Docker
	// Hub login too -- it must not be gated to non-pull_request events.
	loginIdx := strings.Index(buildWorkflow, "- name: Login to Docker Hub")
	require.Greater(t, loginIdx, -1)
	loginStep := buildWorkflow[loginIdx:]
	if end := strings.Index(loginStep, "\n      - name:"); end > -1 {
		loginStep = loginStep[:end]
	}
	assert.NotContains(t, loginStep, `github.event_name != 'pull_request'`)
	// Fork pull requests get no secrets at all, so they cannot pull the base.
	// The job must skip for them rather than fail with an opaque 401, and the
	// condition must stay a bare expression: a ${{ }} fragment spliced into an
	// already-bare `if` string-concatenates and silently always passes.
	dockerBuildIf := buildWorkflow[strings.Index(buildWorkflow, "  docker-build:"):]
	dockerBuildIf = dockerBuildIf[:strings.Index(dockerBuildIf, "runs-on:")]
	assert.Contains(t, dockerBuildIf,
		"github.event.pull_request.head.repo.full_name == github.repository")
	assert.NotContains(t, dockerBuildIf, "${{")
	assert.Contains(t, buildWorkflow,
		`group: ${{ github.workflow }}-${{ startsWith(github.ref, 'refs/tags/') && 'tags' || github.ref }}`)
	assert.NotContains(t, buildWorkflow,
		`group: ${{ github.workflow }}-${{ github.ref }}`)
	assert.Equal(t, 1, strings.Count(buildWorkflow,
		`docker buildx imagetools inspect "$source"`))
	assert.Equal(t, 2, strings.Count(buildWorkflow,
		`--format '{{json .}}'`))
	assert.NotContains(t, buildWorkflow, `imagetools inspect "$source" --raw`)
	assert.NotContains(t, buildWorkflow, `imagetools inspect "$go_source" --raw`)
	assert.Equal(t, 1, strings.Count(buildWorkflow, `source="$NG_BASE_SOURCE"`))
	assert.Equal(t, 1, strings.Count(buildWorkflow,
		`docker buildx imagetools inspect "$go_source"`))
	assert.Equal(t, 1, strings.Count(buildWorkflow,
		`go_source="tykio/golang-cross:${GOLANG_CROSS}"`))
	assert.Equal(t, 2, strings.Count(buildWorkflow,
		`echo "platform_digest=$platform_digest" >> "$GITHUB_OUTPUT"`))
	assert.Equal(t, 3, strings.Count(buildWorkflow,
		`printf 'source_base_linux_amd64_digest=%s\n'`))
	assert.Equal(t, 3, strings.Count(buildWorkflow,
		`printf 'go_toolchain_linux_amd64_digest=%s\n'`))
	assert.NotContains(t, buildWorkflow, `GOLANG_IMAGE=tykio/golang-cross:`)
	assert.Equal(t, 1, strings.Count(buildWorkflow, "name: Build workflow-local NG base"))
	assert.Contains(t, buildWorkflow, `BASE_IMAGE=${{ steps.source-base.outputs.ref }}`)
	assert.Contains(t, buildWorkflow,
		`GO_TOOLCHAIN_IMAGE_REF=${{ steps.source-go.outputs.ref }}`)
	assert.NotContains(t, buildWorkflow, `PREPROVISIONED_TOOLCHAIN`)
	assert.Contains(t, buildWorkflow,
		`base="${{ steps.login-ecr.outputs.registry }}/tyk-plugin-compiler@`+
			`${{ steps.build-ng-base.outputs.digest }}"`)
	assert.Contains(t, buildWorkflow,
		`driver: ${{ github.event_name == 'pull_request' && 'docker' || 'docker-container' }}`)
	assert.Contains(t, buildWorkflow,
		`push: ${{ github.event_name != 'pull_request' }}`)
	assert.Contains(t, buildWorkflow,
		`load: ${{ github.event_name == 'pull_request' }}`)
	assert.Contains(t, buildWorkflow,
		`sbom: ${{ github.event_name != 'pull_request' }}`)
	assert.Contains(t, buildWorkflow,
		`provenance: ${{ github.event_name != 'pull_request' && 'mode=max' || 'false' }}`)
	assert.Contains(t, buildWorkflow,
		"if: ${{ github.event_name != 'pull_request' }}\n"+
			"        uses: aws-actions/configure-aws-credentials@")
	// Deliberately NOT gated to non-pull_request, unlike the ECR steps above:
	// the toolchain base is a private Docker Hub repo that every event must pull.
	assert.NotContains(t, buildWorkflow,
		"if: ${{ github.event_name != 'pull_request' }}\n"+
			"        uses: docker/login-action@")
	assert.Equal(t, 3, strings.Count(buildWorkflow, "latest=false"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "sbom: true"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "provenance: mode=max"))
	assert.Equal(t, 1, strings.Count(buildWorkflow, "docker/setup-buildx-action@"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "name: Verify pushed NG digest and attestations"))
	assert.Equal(t, 3, strings.Count(buildWorkflow,
		`docker buildx imagetools inspect "$exact_ref" --format '{{json .SBOM}}'`))
	assert.Equal(t, 3, strings.Count(buildWorkflow,
		`docker buildx imagetools inspect "$exact_ref" --format '{{json .Provenance}}'`))
	assert.Equal(t, 3, strings.Count(buildWorkflow, `test "$resolved" = "$PUBLISHED_DIGEST"`))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "expected_repositories=2"))
	assert.Equal(t, 3, strings.Count(buildWorkflow, "actions/upload-artifact@"))
	assertPluginCompilerSelfTestBuilds(t, buildWorkflow, 3)
	assertPluginCompilerVariantParity(t, buildWorkflow)

	releaseDockerfile := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/Dockerfile.release")
	assert.Equal(t, 1, strings.Count(releaseDockerfile, "ARG WITH_GATEWAY_SELFTEST=0"))
	assert.Contains(t, releaseDockerfile, "test ! -e /usr/local/bin/tyk")
	assert.Contains(t, releaseDockerfile, `test ! -e "$TYK_GW_PATH/tyk"`)

	releaseDockerignore := readRenderedFile(
		t,
		outputDir,
		"ci/images/plugin-compiler-ng/Dockerfile.release.dockerignore",
	)
	assert.Contains(t, strings.Split(releaseDockerignore, "\n"), "/tyk")
	assert.Contains(t, strings.Split(releaseDockerignore, "\n"),
		"/ci/images/plugin-compiler-ng/scripts/loadtest-gate.sh")

	baseDockerfile := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/Dockerfile.base")
	assert.Contains(t, baseDockerfile, "Using pre-provisioned compiler toolchain")
	assert.Contains(t, baseDockerfile, "USER 0")
	assert.NotContains(t, baseDockerfile, "USER root")
	assert.Contains(t, baseDockerfile, `needs="gcc ld ld.gold readelf jq file make bash sed awk grep tar find"`)
	assert.Contains(t, baseDockerfile, `test -s /etc/ssl/certs/ca-certificates.crt`)
	assert.Contains(t, baseDockerfile, `command -v apt-get >/dev/null`)
	// The toolchain is never installed at build time: it must arrive
	// pre-provisioned in the DHI customization, so that every compiler package
	// carries a +dhi version covered by Docker's maintenance obligation and by
	// published DHI vulnerability decisions. There is no fallback and no switch.
	assert.NotContains(t, baseDockerfile, `PREPROVISIONED_TOOLCHAIN`)
	assert.Contains(t, baseDockerfile,
		`echo "Using pre-provisioned compiler toolchain from ${BASE_IMAGE}"`)
	assert.Contains(t, baseDockerfile,
		`is missing pre-provisioned toolchain components:$missing`)
	assert.Contains(t, baseDockerfile,
		`ERROR: the toolchain must be provisioned by the DHI customization, not installed here.`)
	// The verification step reads the base and fails; it must never install.
	verifyStart := strings.Index(baseDockerfile,
		`echo "Using pre-provisioned compiler toolchain from ${BASE_IMAGE}"`)
	slimStart := strings.Index(baseDockerfile,
		`# --- Safe filesystem slimming + package-integrity gate `)
	require.Greater(t, verifyStart, -1)
	require.Greater(t, slimStart, verifyStart)
	verifyBlock := baseDockerfile[verifyStart:slimStart]
	assert.NotContains(t, verifyBlock, "apt-get")
	assert.NotContains(t, verifyBlock, "apk add")
	assert.Contains(t, verifyBlock, "exit 1")
	assert.Contains(t, baseDockerfile,
		`needs="$needs x86_64-linux-gnu-g++ aarch64-linux-gnu-g++ s390x-linux-gnu-g++"`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib/libpython"*.so*`)
	assert.Contains(t, baseDockerfile,
		`test -z "$(find "$SR/usr/lib" -maxdepth 1 -name 'libpython*.so*' -print -quit)"`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libkrb5*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libgssapi_krb5*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libk5crypto*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libgssrpc*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libkadm5*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libkdb5*`)
	assert.Contains(t, baseDockerfile, `"$SR/usr/lib"/libkrad*`)
	assert.Contains(t, baseDockerfile, `rm -rf "$SR/usr/lib/krb5"`)
	assert.Contains(t, baseDockerfile,
		`test -z "$(find "$SR/usr/lib" -maxdepth 1 \( -name 'libkrb5*'`)
	assert.Contains(t, baseDockerfile, `test ! -e "$SR/usr/lib/krb5"`)
	assert.Contains(t, baseDockerfile, "preserving unrelated generic libraries")
	assert.Equal(t, 2, strings.Count(baseDockerfile, "ARG BASE_IMAGE"))
	assert.Contains(t, baseDockerfile, `dpkg --help 2>&1 | grep -q -- '--audit'`)
	assert.Contains(t, baseDockerfile, `test -z "$(dpkg --audit)"`)
	assert.Contains(t, baseDockerfile, `packages > 0 && packages == statuses && bad == 0`)
	assert.Contains(t, baseDockerfile, "apt-get check")
	assert.NotContains(t, baseDockerfile, "dpkg --purge --force-all")
	assert.NotContains(t, baseDockerfile, "rm -f \"/var/lib/dpkg/status.d/")
	assert.NotContains(t, baseDockerfile, "rewrite-imports")

	buildScript := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/data/build.sh")
	assert.Contains(t, buildScript, `plugin_name must be a file basename, not a path`)
	assert.Contains(t, buildScript, `[ "$(basename "$source_entry")" != ".git" ]`)
	assert.Contains(t, buildScript,
		"Preserving existing go.mod module path and Go source imports")
	assert.NotContains(t, buildScript, "rewrite-imports")
	assert.NotContains(t, buildScript, "go mod edit -module")
	assert.Contains(t, buildScript,
		`drop_godebug && /^[[:space:]]*godebug[[:space:]]+/ { next }`)
	assert.Contains(t, buildScript,
		"removing plugin godebug key unsupported by Gateway Go")
	assert.Contains(t, buildScript,
		"GOWORK=off GO111MODULE=on go list .")
	assert.Contains(t, buildScript,
		`drop_ignore && /^[[:space:]]*ignore[[:space:]]+/ { next }`)
	assert.NotContains(t, buildScript, `sed -i -e "s,\"${OLD_MODULE}`)

	_, rewriteErr := os.Stat(filepath.Join(
		outputDir,
		"ci/images/plugin-compiler-ng/data/rewrite-imports.go",
	))
	require.ErrorIs(t, rewriteErr, os.ErrNotExist)

	validator := readRenderedFile(t, outputDir, "ci/images/plugin-compiler-ng/data/validate-plugin.sh")
	assert.Contains(t, validator, "unable to inspect ELF dynamic dependencies")
	assert.Contains(t, validator, "unable to inspect ELF version requirements")
	assert.Contains(t, validator, "the official Gateway image does not ship libpython")

	gateScript := readRenderedFile(t, outputDir,
		"ci/images/plugin-compiler-ng/scripts/loadtest-gate.sh")
	assert.Contains(t, gateScript, `plugin load -f /gate-plugin.so -s "$SYMBOL"`)
	assert.Contains(t, gateScript, `"path": "/gate/plugin.so"`)
	assert.Contains(t, gateScript, `"target_url": "http://upstream:8081/"`)
	assert.Contains(t, gateScript, `r.Header.Get("Foo")`)
	assert.Contains(t, gateScript, `[ "$response" = "Bar" ]`)
	assert.Contains(t, gateScript,
		"redis:6.0-alpine@sha256:2b35fc7d2908e25aa6aa197f97882c8a67829d3b106ad5ea5c8028f816f26aa8")
	assert.Less(t,
		strings.Index(gateScript, `if [ "${VALIDATE_ONLY:-0}" = "1" ]`),
		strings.Index(gateScript, `docker network create "$NETWORK"`))
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
		edition            string
		binaryStrings      string
		wantFailure        string
		wantOutput         string
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
		{
			name: "native Go FIPS uses its embedded build setting",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "crypto/internal/boring\ncrypto/internal/fips140\nbuild\tGOFIPS140=v1.0.0",
			wantOutput:    "FIPS: GOFIPS140=v1.0.0 (embedded build setting)",
		},
		{
			name: "legacy BoringCrypto uses its explicit C symbol",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "_goboringcrypto\ncrypto/internal/boring\ncrypto/internal/fips140",
			wantOutput:    "FIPS: boringcrypto (embedded build setting/symbol)",
		},
		{
			name: "legacy BoringCrypto build setting is accepted",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "build\tGOEXPERIMENT=boringcrypto",
			wantOutput:    "FIPS: boringcrypto (embedded build setting/symbol)",
		},
		{
			name: "ordinary Go crypto packages are not FIPS evidence",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "crypto/internal/boring\ncrypto/internal/fips140",
			wantFailure:   "plugin shows NO FIPS crypto",
		},
		{
			name: "disabled native Go FIPS is rejected",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "build\tGOFIPS140=off\ncrypto/internal/fips140",
			wantFailure:   "plugin shows NO FIPS crypto",
		},
		{
			name: "conflicting native and legacy evidence fails closed",
			dynamic: strings.Join([]string{
				"Dynamic section at offset 0x1 contains 1 entry:",
				" 0x0000000000000001 (NEEDED) Shared library: [libc.so.6]",
			}, "\n"),
			edition:       "ee-fips",
			binaryStrings: "build\tGOFIPS140=v1.0.0\n_goboringcrypto",
			wantFailure:   "conflicting native GOFIPS140 and legacy boringcrypto",
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
  printf '%s: go1.25.12\n\tbuild\t-buildmode=plugin\n\tbuild\t-tags=ee,fips\n' "$3"
else
  echo "go version go1.25.12 linux/amd64"
fi
`)
			markerCommand := "#!/bin/sh\ncat <<'EOF'\n" + test.binaryStrings + "\nEOF\n"
			writeExecutable(t, filepath.Join(fakeBin, "strings"), markerCommand)
			writeExecutable(t, filepath.Join(fakeBin, "nm"), markerCommand)

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
			edition := test.edition
			if edition == "" {
				edition = "ce"
			}
			cmd.Env = []string{
				"PATH=" + fakeBin + string(os.PathListSeparator) +
					"/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
				"EXPECT_EDITION=" + edition,
			}
			output, runErr := cmd.CombinedOutput()
			if test.wantFailure == "" {
				require.NoError(t, runErr, string(output))
				assert.Contains(t, string(output), "validation OK")
				if test.wantOutput != "" {
					assert.Contains(t, string(output), test.wantOutput)
				}
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
		usesNativeFIPS   bool
	}{
		{branch: "release-5.3", goImage: "1.23-bullseye", variantCount: 1},
		{branch: "release-5.8", goImage: "1.26-bullseye", variantCount: 3, usesBoringCrypto: true},
		{branch: "release-5.8.15", goImage: "1.26-bullseye", variantCount: 3, usesBoringCrypto: true},
		{branch: "release-5.13", goImage: "1.26-bullseye", variantCount: 3, usesNativeFIPS: true},
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
				assert.NotContains(t, workflow, "FIPS_GOFIPS140=")
			} else {
				assert.NotContains(t, workflow, "boringcrypto")
			}
			if test.usesNativeFIPS {
				assert.Contains(t, workflow, "FIPS_GOFIPS140=v1.0.0")
				assert.Contains(t, workflow, "SELFTEST_GOFIPS140=v1.0.0")
			}
		})
	}
}

func assertPluginCompilerSelfTestBuilds(t *testing.T, workflow string, variantCount int) {
	t.Helper()

	type workflowStep struct {
		Name string         `yaml:"name"`
		ID   string         `yaml:"id"`
		If   string         `yaml:"if"`
		Uses string         `yaml:"uses"`
		Env  map[string]any `yaml:"env"`
		With map[string]any `yaml:"with"`
	}
	var rendered struct {
		Jobs map[string]struct {
			Strategy any            `yaml:"strategy"`
			Steps    []workflowStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(workflow), &rendered))

	job, ok := rendered.Jobs["docker-build"]
	require.True(t, ok)
	assert.Nil(t, job.Strategy)
	assert.NotContains(t, workflow, "matrix:")
	jobsWithSteps := 0
	for _, candidate := range rendered.Jobs {
		if len(candidate.Steps) > 0 {
			jobsWithSteps++
		}
	}
	assert.Equal(t, 1, jobsWithSteps)

	var gateBuilds, pushedBuilds, proofSteps, evidenceUploads int
	var publishIDs []string
	for _, step := range job.Steps {
		buildArgs, _ := step.With["build-args"].(string)
		switch {
		case strings.HasPrefix(step.Name, "Build local NG gate image"):
			gateBuilds++
			assert.Equal(t, false, step.With["push"])
			assert.Equal(t, true, step.With["load"])
			assert.Contains(t, buildArgs, `GOLANG_IMAGE=${{ steps.source-go.outputs.ref }}`)
			assert.Contains(t, buildArgs, "WITH_GATEWAY_SELFTEST=1")
			assert.NotContains(t, buildArgs, "WITH_GATEWAY_SELFTEST=0")
		case strings.HasPrefix(step.Name, "Build and push NG image"):
			pushedBuilds++
			publishIDs = append(publishIDs, step.ID)
			assert.NotEmpty(t, step.ID)
			assert.Equal(t, true, step.With["push"])
			assert.Equal(t, "${{ github.event_name != 'pull_request' }}", step.If)
			assert.Contains(t, buildArgs, `GOLANG_IMAGE=${{ steps.source-go.outputs.ref }}`)
			assert.Contains(t, buildArgs, "WITH_GATEWAY_SELFTEST=0")
			assert.NotContains(t, buildArgs, "WITH_GATEWAY_SELFTEST=1")
			assert.Equal(t, true, step.With["sbom"])
			assert.Equal(t, "mode=max", step.With["provenance"])
		case strings.HasPrefix(step.Name, "Verify pushed NG digest and attestations"):
			proofSteps++
			assert.Equal(t, "${{ github.event_name != 'pull_request' }}", step.If)
			assert.Contains(t, step.Env["PUBLISHED_DIGEST"], ".outputs.digest")
		case strings.HasPrefix(step.Name, "Retain pushed NG evidence"):
			evidenceUploads++
			assert.Contains(t, step.Uses, "actions/upload-artifact@")
		}
	}

	assert.Equal(t, variantCount, gateBuilds)
	assert.Equal(t, variantCount, pushedBuilds)
	assert.Equal(t, variantCount, proofSteps)
	assert.Equal(t, variantCount, evidenceUploads)
	assert.Len(t, publishIDs, variantCount)
	expectedPublishIDs := []string{"build-push-ng"}
	if variantCount > 1 {
		expectedPublishIDs = append(expectedPublishIDs, "build-push-ee-ng")
	}
	if variantCount > 2 {
		expectedPublishIDs = append(expectedPublishIDs, "build-push-fips-ng")
	}
	assert.ElementsMatch(t, expectedPublishIDs, publishIDs)

	suffixes := []string{""}
	if variantCount > 1 {
		suffixes = append(suffixes, " EE")
	}
	if variantCount > 2 {
		suffixes = append(suffixes, " FIPS")
	}
	lastIndex := -1
	for _, suffix := range suffixes {
		for _, prefix := range []string{
			"Build local NG gate image",
			"Gate NG plugin load",
			"Build and push NG image",
			"Verify pushed NG digest and attestations",
			"Retain pushed NG evidence",
		} {
			name := prefix + suffix
			index := -1
			for i, step := range job.Steps {
				if step.Name == name {
					index = i
					break
				}
			}
			require.Greater(t, index, lastIndex, "step %q must be sequential", name)
			lastIndex = index
		}
	}
}

func assertPluginCompilerVariantParity(t *testing.T, workflow string) {
	t.Helper()

	for _, gate := range []string{
		"tyk-plugin-compiler-ng-gate:std ce",
		"tyk-plugin-compiler-ng-gate:ee ee",
		"tyk-plugin-compiler-ng-gate:fips ee-fips",
	} {
		assert.Contains(t, workflow, gate)
	}
	for _, image := range []string{
		"tykio/tyk-plugin-compiler,",
		"tykio/tyk-plugin-compiler-ee,",
		"tykio/tyk-plugin-compiler-fips,",
	} {
		assert.Contains(t, workflow, image)
	}
}

func readRenderedFile(t *testing.T, outputDir, name string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(name)))
	require.NoError(t, err)
	return string(contents)
}

// TestPluginCompilerNGImagePlatforms pins what `imageplatforms` controls. The
// key is the feature flag for native multi-arch compiler images: with it at its
// default the render must be byte-for-byte what an amd64-only build always was,
// and every piece of emulation machinery must appear only when a non-native
// platform is actually asked for.
func TestPluginCompilerNGImagePlatforms(t *testing.T) {
	render := func(t *testing.T, platforms string) string {
		t.Helper()

		var pol Policies
		config.LoadConfig("")
		require.NoError(t, LoadRepoPolicies(&pol))

		repo, err := pol.GetRepoPolicy("tyk")
		require.NoError(t, err)
		require.NoError(t, repo.SetBranch("master"))

		ng := repo.Branchvals.PluginCompiler.NextGen
		ng.ImagePlatforms = platforms
		repo.Branchvals.PluginCompiler.NextGen = ng

		bundle, err := NewBundle([]string{"plugin-compiler-ng"})
		require.NoError(t, err)

		outputDir := t.TempDir()
		_, err = bundle.Render(repo, outputDir, nil)
		require.NoError(t, err)

		return readRenderedFile(t, outputDir, ".github/workflows/plugin-compiler-ng-build.yml")
	}

	// Everything below is emulation machinery. None of it may cost an
	// amd64-only build anything -- notably the go.dev lookup, which would
	// otherwise make every release depend on a third-party host it has no
	// reason to contact.
	armOnly := []string{
		"setup-qemu-action",
		"go.dev/dl",
		"GO_TARBALL_SHA256_ARM64",
		"Build linux/arm64 NG gate image",   // the per-platform Tier B build
		"Validate linux/arm64 NG toolchain", // its VALIDATE_ONLY companion
		"COMPILER_PLATFORM",                 // which pins that companion to the emulated image
	}

	t.Run("amd64 only is the default and renders no emulation", func(t *testing.T) {
		workflow := render(t, "")

		for _, marker := range armOnly {
			assert.NotContains(t, workflow, marker,
				"an amd64-only image must not carry %q", marker)
		}
		assert.NotContains(t, workflow, "linux/arm64,",
			"no build step may span platforms")

		// `load: true` accepts exactly one platform, so the Tier A gate build
		// stays single-platform no matter what else is requested.
		assert.Equal(t, 7, strings.Count(workflow, "platforms: linux/amd64\n"),
			"expected the base build plus a gate and a push build per variant")
	})

	t.Run("adding arm64 enables emulation and spans the push", func(t *testing.T) {
		workflow := render(t, "linux/amd64,linux/arm64")

		for _, marker := range armOnly {
			assert.Contains(t, workflow, marker,
				"a multi-platform image must carry %q", marker)
		}

		// The published index spans both platforms; the gate build cannot.
		assert.Equal(t, 3, strings.Count(workflow, "platforms: linux/amd64,linux/arm64\n"),
			"expected the push step per variant to span both platforms")
		assert.Equal(t, 3, strings.Count(workflow, "platforms: linux/amd64\n"),
			"the Tier A gate build stays on the gate platform")
		// Tier B is paid once, not per variant: the toolchain it exercises is
		// the same in all of them and the gate runs emulated.
		assert.Equal(t, 1, strings.Count(workflow, "platforms: linux/arm64\n"),
			"expected exactly one Tier B gate build, for ExtraGateVariant")
		assert.Contains(t, workflow, "- name: Build linux/arm64 NG gate image EE\n")
		assert.NotContains(t, workflow, "- name: Build linux/arm64 NG gate image FIPS\n")
		assert.NotContains(t, workflow, "- name: Build linux/arm64 NG gate image\n")

		// Pull requests use the `docker` driver, which cannot build more than
		// one platform, so the base build has to narrow itself there.
		assert.Contains(t, workflow,
			"platforms: ${{ github.event_name == 'pull_request' && 'linux/amd64' || 'linux/amd64,linux/arm64' }}")

		// Emulated work must never run on a pull request: forks have no
		// secrets, and the `docker` driver cannot load a foreign platform.
		for _, step := range []string{
			"Set up QEMU",
			"Build linux/arm64 NG gate image EE",
			"Validate linux/arm64 NG toolchain EE",
		} {
			idx := strings.Index(workflow, "- name: "+step+"\n")
			require.NotEqual(t, -1, idx, "step %q not rendered", step)
			assert.Contains(t, workflow[idx:min(idx+200, len(workflow))],
				"if: ${{ github.event_name != 'pull_request' }}",
				"step %q must be release-only", step)
		}
	})
}
