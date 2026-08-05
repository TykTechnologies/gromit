# Plugin Compiler NG And DHI VEX Goal

Last updated: 2026-07-29

## North Star - Do Not Weaken

An enterprise customer must be able to copy a released Tyk Gateway or Plugin
Compiler NG image into a private registry and run unmodified stock Trivy
against the mirrored immutable digest:

```bash
trivy image --scanners vuln --severity HIGH,CRITICAL \
  --vex repo --vex oci --show-suppressed \
  private.registry.example/namespace/tyk-plugin-compiler-ee@sha256:...
```

This passes only when:

- the command uses a stock Trivy release, not a fork, patch, wrapper, or
  Tyk-specific scanner;
- Docker's live DHI decisions remain the authority. Stock Trivy consumes them
  directly when possible; a Tyk-hosted compatibility repository is acceptable
  only if direct consumption is impossible and it mechanically preserves
  Docker's current statements without creating a Tyk vulnerability decision;
- registry or namespace changes preserve the released image basename and do
  not prevent inherited DHI package-PURL decisions from matching;
- the mirror retains the digest-bound VEX transport that the selected stock
  Trivy release actually discovers: Cosign's legacy `.att` tag for Trivy 0.72,
  plus an OCI 1.1 referrer only for future-compatible clients;
- the image-bound VEX is not allowed to independently suppress while
  EXP-003's stale-source conflict remains unresolved;
- active inherited DHI OS findings exactly reflect Docker's current applicable
  decisions. The expected current result is `0 Critical / 0 High` only if the
  revision-bound Docker evidence actually leaves none active;
- `--show-suppressed` identifies every applied decision and its source;
- legitimate child-layer, application, unmatched, `affected`, and
  `under_investigation` findings remain visible.

Do not claim success from a source DHI scan, a Docker Hub-only scan, a local
VEX file, Docker Scout output, `--ignore-unfixed`, or a zero-only summary.
The final proof is this command against the private-registry mirror.

## Scope Decision - 2026-07-29

The Git and Perl package-removal investigation is closed. Plugin Compiler NG
will retain Docker's pre-provisioned Debian Git package and its declared Perl
dependency closure. Those packages appearing in an SBOM or raw scanner report
is not itself the defect being addressed.

For the immutable DHI revision captured in EXP-010, Docker's signed
image-specific VEX marks the source-package versions behind all 59 inherited
Critical and High occurrences `not_affected`. That evidence closes the package
removal investigation, but it does not settle current authoritative
affectedness: EXP-014 found a later overlapping `under_investigation` statement
in Docker's signed public feed. Docker's precedence, currentness, and revocation
contract therefore remains open.

The remaining VEX delivery question is:

1. How can unmodified stock Trivy discover Docker's current applicable
   image-specific decisions after the child image is copied to a private
   registry?
2. How can Docker's signed SBOM source-to-binary relationships be projected to
   the exact package PURLs Trivy reports as explicitly Tyk-authored
   compatibility bytes, while preserving Docker's decision fields and
   provenance without claiming that Docker signed the translation?
3. What live freshness, completeness, and revocation contract prevents a stale
   repository or image-bound record from suppressing a newer Docker
   `affected` or `under_investigation` decision?

Docker's signed image VEX contains the decisions for the captured revision.
The DHI console is corroborating UI evidence, not a separate decision source.
Neither is permission to manufacture blanket `not_affected` statements. Each
later image or VEX revision requires a new evidence record. The proof must
demonstrate how stock Trivy applies the current Docker records to the mirrored
image and leaves any later affected or unresolved decision visible.

## Experiment Ledger - Append Only

Every experiment records immutable inputs, exact commands, counts, result, and
cleanup. Never replace an older result when Docker, Trivy, the image, or the
VEX feed changes; append a new experiment.

Lessons now enforced:

- official documentation is not proof that our customized child image works;
- a changed live feed requires a new experiment, not reinterpretation of an
  old result;
- raw, suppressed, and active counts must be retained together;
- source image success does not prove private-registry mirror success;
- repository VEX and OCI VEX must be tested separately before the combined
  north-star command;
- every report must record the Trivy release, vulnerability DB metadata, VEX
  repository commit, image index/platform digests, and cleanup performed.

### EXP-001 - Public BusyBox VEX lacked Trivy binary aliases

- Date: 2026-07-27/28
- Trivy: `0.72.0`
- Image:
  `dhi.io/busybox:1-debian-fips-dev@sha256:d383017cc5b4984c4088845242dbb70c075102aae43c16c45038d5750c411f42`
- Platform digest:
  `sha256:4086ad19166e954afdff09ffa27a9fa1daa8c57ca747d4ce76bd67bc93626621`
- Public advisories commit:
  `01b0f4f4f4c247221005d0c8b96e381af34ab291`
- Method: direct `--vex <public-file>` comparison with authenticated Scout VEX.
- Raw: `4 Critical / 18 High`.
- Public file: `0 Critical / 6 High`, `16` suppressed.
- Authenticated Scout export: `0 Critical / 0 High`, `22` suppressed.
- Root cause: the old public file omitted scanner-matchable Debian binary
  aliases for five `busybox` High findings and one `gpgv` High finding.
- Result: **FAIL** for the north star.
- Full reproduction:
  `docs/docker-dhi-public-vex-trivy-bug.md`.

### EXP-002 - Current Docker Trivy VEX repository against alpha8 EE

- Date: 2026-07-29
- Status: **FAIL**
- Trivy: `0.72.0`
- Trivy DB:
  - source: `mirror.gcr.io/aquasec/trivy-db:2`;
  - manifest:
    `sha256:43f675dddccf9888eb9034f785a134c4f1f6446e53cd77136f5fbd6688eede34`;
  - DB layer:
    `sha256:728a29c1190b0fcbeb6cf6eb2c77b544c43fa6470744524d6246efdd3eb6caab`;
  - updated: `2026-07-29T07:47:23.42987139Z`.
- Public advisories commit:
  `b42ed4ce6b2862a3ebf4112d474d748d37a5b033`
- Cached archive ETag:
  `W/"758c4f3c814cf9533db678fff534b94d8a243abdb4f5a279b0b7fa72c4ecc1e7"`
- Evidence limitation: Trivy's downloaded VEX cache did not retain the Git
  commit, so the archive cannot be cryptographically tied to the observed
  repository HEAD by this run alone.
- Image:
  `tykio/tyk-plugin-compiler-ee:v5.15.0-alpha8-ng`
- Exact scanned reference:
  `docker.io/tykio/tyk-plugin-compiler-ee@sha256:443a2f8496232d41023007d7228df5a105556565f4ea36c759f47a69001f0a62`
- Index digest:
  `sha256:2176a0e047802465bd9a0819106f6a4e3bc0aa3225cca3d27c86ca74b1a3fba5`
- Linux/amd64 digest:
  `sha256:443a2f8496232d41023007d7228df5a105556565f4ea36c759f47a69001f0a62`
- Image config:
  `sha256:7007e09773192f6df46c8fd970f2b716931ed9c32015c36b2fe2821eae4e8dac`
- Detected OS: Debian `13.6`.
- Method: isolated stock-Trivy configuration with Docker's public advisories
  repository, raw versus `--vex repo --show-suppressed`, then
  `--vex oci --show-suppressed`.
- Common Trivy arguments:
  `--cache-dir <isolated-cache> --config <isolated-config> --timeout 15m
  image --disable-telemetry --image-src remote --platform linux/amd64
  --scanners vuln --severity HIGH,CRITICAL --format json`.
- Mode-specific arguments were no VEX flags for raw,
  `--vex repo --show-suppressed` for repository VEX, and
  `--vex oci --show-suppressed` for OCI VEX.
- Observation before scan: the current public BusyBox document now contains
  exact Debian binary package PURLs, including a versioned `busybox` PURL that
  the old EXP-001 document lacked.
- Raw result: `17 Critical / 82 High`, none suppressed.
- Repository VEX result: `9 Critical / 57 High` active and
  `8 Critical / 25 High` suppressed.
- OCI VEX result: `17 Critical / 82 High` active, none suppressed; Trivy
  reported `No VEX attestations found`.
- Repository VEX partitioned all `99` findings into `66` active and `33`
  suppressed. Every suppression was a Docker `not_affected` decision with
  `inline_mitigations_already_exist`.
- A `registry-1.docker.io` alias of the same immutable digest produced the
  identical active/suppressed vulnerability and package-PURL sets. This proves
  package-PURL matching survives that alias, but it is not a private-registry
  copy proof.
- One preliminary invocation failed before scanning because Trivy 0.72 requires
  `--disable-telemetry` after the `image` subcommand. The corrected raw,
  repository, OCI, and alias scans all exited zero and produced JSON.
- Result: Docker's current repository improved on EXP-001 but does not account
  for all findings in this child image. Whether the remaining 66 are inherited
  unchanged from DHI or introduced/changed by child layers is pending the
  separate lineage classification. The alpha also lacks a Trivy-discoverable
  VEX attestation. **FAIL** for the north star.
- Evidence gap: the exact isolated repository configuration, Trivy executable
  SHA-256, JSON reports, and report SHA-256 values were not retained. The
  observed Git HEAD is not cryptographically tied to Trivy's downloaded
  archive. Do not use this run as release evidence.
- Cleanup: isolated temporary HOME, Trivy cache, Docker auth, Git config, and
  JSON reports under `/tmp/gromit-trivy-acceptance.H4aip9` were removed. No
  Docker daemon was used.

### EXP-003 - Trivy 0.72 cannot let live repository VEX revoke stale OCI VEX

- Date: 2026-07-29
- Status: **FAIL / NO-SHIP DESIGN**
- Trivy source commit:
  `8a32853686209a428179bb3a1688802b25691564`
- `go-vex`: `v0.2.7`, commit
  `3185a64ed27703fc3fe4af8cd5e1ce0ed2fa2569`.
- Method: source audit of repeated `--vex` flag handling, source
  initialization, repository selection, OCI discovery, OpenVEX matching, and
  suppression.
- Result: `--vex repo --vex oci` applies suppressive sources with logical-OR
  semantics. A repository `affected`, `under_investigation`, or unmatched
  result does not veto a later OCI `not_affected` or `fixed` result.
- Trivy does not compare repository and OCI document timestamps, versions,
  `last_updated`, authors, signatures, or authority. It also consumes only the
  first discovered OCI VEX document.
- Consequence: a static suppressive OCI snapshot can hide a finding after
  Docker changes its live decision to `affected` or `under_investigation`.
  Reversing CLI source order does not fix this.
- Verification: Trivy's `go test ./pkg/vex` passed. No Docker daemon or
  repository edits were used.
- Result: attaching a suppressive compatibility snapshot as a freshness
  fallback is unsafe and cannot pass the north star's negative-control
  requirement. Do not ship this design merely because it makes the count zero.

### EXP-004 - Trivy 0.72 VEX attestation transport compatibility audit

- Date: 2026-07-29
- Status: **SOURCE AUDIT COMPLETE / INTEGRATION NOT RUN**
- Trivy: `0.72.0`.
- Trivy source:
  `8a32853686209a428179bb3a1688802b25691564`.
- OpenVEX discovery source:
  `7c54efc57553`.
- Cosign compatibility source: `v2.6.2`.
- Finding: `--vex oci` discovers Cosign 2.6.2's legacy digest-tagged
  `sha256-<digest>.att` object with predicate type
  `https://openvex.dev/ns`; it does not discover a generic OCI 1.1 OpenVEX
  referrer.
- Trivy consumes only the first discovered OCI VEX document and does not
  verify its Cosign signature. Publication therefore requires one complete
  aggregate, replace semantics, and a separate `cosign verify-attestation`
  gate before acceptance.
- Source-derived proposal: a private-registry-portable image product can omit
  `repository_url` and `arch` while retaining the exact image basename and
  digest. Every vulnerable package must remain an exact Trivy-emitted
  subcomponent PURL. This still requires a registry integration proof.
- Source-derived proposal: a same-basename mirror can match that product.
  Renaming the image basename requires a destination-specific product and is
  not covered by the portable form.
- Source-derived proposal: `oras cp --recursive` preserves OCI referrers but
  not Cosign's unrelated legacy `.att` tags. A Trivy 0.72-compatible mirror
  must explicitly copy the attestation tags for the image index and scanned
  platform digest. No ORAS version or registry integration was tested.
- Result: the transport is implementable, but a suppressive attachment remains
  blocked by EXP-003's stale-decision safety failure and EXP-002's 66
  unaccounted findings. No registry mutation or Docker command was performed.

### EXP-005 - Classify EXP-002's 66 active findings

- Date: 2026-07-29
- Status: **FAIL / ROOT CAUSE IDENTIFIED**
- Interpretation scope: historical public-repository-only result. EXP-010
  supersedes its affectedness interpretation with Docker's signed
  image-specific VEX for the later immutable DHI revision. It does not reopen
  Git or Perl package removal.
- Trivy: `0.72.0`.
- Trivy executable SHA-256:
  `242c86fa4fb1304014631cd6b89638bb342dea11a1bf9fa56687fed5d2c18d90`.
- Image, platform, config, database, and advisory commit match EXP-002.
- Docker advisories tree:
  `67774157f4fe00a45fbc42bec038c502ca20d47b`.
- Trivy's downloaded VEX directory was byte-identical to that checkout.
- `1 Critical / 43 High` are product/subcomponent PURL mismatches:
  - `gpgv`: Docker statements use source package `gnupg2`;
  - `libexpat1`: Docker statements use source package `expat`;
  - `linux-libc-dev`: Docker statements use source package `linux`, selected
    binary products, or image products.
- `8 Critical / 14 High` are deliberately active because the latest applicable
  Docker statement is `under_investigation`:
  - six `util-linux` binary packages retain High `CVE-2026-53615`;
  - four Perl binary packages retain Critical `CVE-2026-13221` and
    `CVE-2026-57433`, plus High `CVE-2026-48962` and `CVE-2026-57432`.
- There were no absent CVE statements across the repository and no installed
  package version mismatches.
- Result: the 44 PURL gaps can be pursued as Docker/Trivy interoperability or
  carefully evidenced binary-alias work. The 22 newer
  `under_investigation` findings must not be VEX-suppressed; they require
  package remediation/removal, a newer clean DHI rebuild, or a later
  authoritative Docker decision.
- Scope note added 2026-07-29: package remediation/removal is no longer a Tyk
  workstream. The pre-provisioned Docker package closure is retained; this
  project now waits for a current Docker decision or fixes only VEX
  interoperability.
- Full package and CVE inventory:
  `docs/plugin-compiler-ng-exp002-root-cause.md`.
- Cleanup: the isolated 2.1 GB Trivy workspace and all `exp002-vex.*`
  artifacts were removed. No Docker daemon was used.

### EXP-006 - Empirical Trivy 0.72 cross-source suppression test

- Date: 2026-07-29
- Status: **PASS / NO-SHIP DESIGN CONFIRMED**
- Trivy tag: `v0.72.0`, a lightweight tag resolving to commit
  `8a32853686209a428179bb3a1688802b25691564`.
- Independent tag verification:

  ```text
  $ git ls-remote https://github.com/aquasecurity/trivy.git refs/tags/v0.72.0 'refs/tags/v0.72.0^{}'
  8a32853686209a428179bb3a1688802b25691564	refs/tags/v0.72.0
  ```

- Successful isolated checkout command:

  ```bash
  git clone --depth 1 --branch v0.72.0 --single-branch \
    https://github.com/aquasecurity/trivy.git trivy
  ```

- Temporary test source:

  ```go
  package vex_test

  import (
  	"testing"

  	"github.com/aquasecurity/trivy/pkg/sbom/core"
  	"github.com/aquasecurity/trivy/pkg/types"
  	"github.com/aquasecurity/trivy/pkg/vex"
  )

  type fakeSource struct {
  	label       string
  	notAffected bool
  }

  func (s fakeSource) NotAffected(_ types.DetectedVulnerability, _, _ *core.Component) (types.ModifiedFinding, bool) {
  	return types.ModifiedFinding{Source: s.label}, s.notAffected
  }

  func TestClientNotAffectedCrossSourceOrder(t *testing.T) {
  	sourceA := fakeSource{label: "repo", notAffected: false}
  	sourceB := fakeSource{label: "oci", notAffected: true}

  	tests := []struct {
  		name    string
  		sources []vex.VEX
  	}{
  		{
  			name:    "A_then_B",
  			sources: []vex.VEX{sourceA, sourceB},
  		},
  		{
  			name:    "B_then_A",
  			sources: []vex.VEX{sourceB, sourceA},
  		},
  	}

  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			client := vex.Client{VEXes: tt.sources}

  			modified, notAffected := client.NotAffected(types.DetectedVulnerability{}, nil, nil)

  			if !notAffected {
  				t.Fatal("Client.NotAffected() returned false, want true")
  			}
  			if got, want := modified.Source, "oci"; got != want {
  				t.Fatalf("Client.NotAffected() source = %q, want %q", got, want)
  			}
  		})
  	}
  }
  ```

- Exact test command and output:

  ```text
  $ go test ./pkg/vex -run TestClientNotAffectedCrossSourceOrder -count=1 -v
  === RUN   TestClientNotAffectedCrossSourceOrder
  === RUN   TestClientNotAffectedCrossSourceOrder/A_then_B
  === RUN   TestClientNotAffectedCrossSourceOrder/B_then_A
  --- PASS: TestClientNotAffectedCrossSourceOrder (0.00s)
      --- PASS: TestClientNotAffectedCrossSourceOrder/A_then_B (0.00s)
      --- PASS: TestClientNotAffectedCrossSourceOrder/B_then_A (0.00s)
  PASS
  ok  	github.com/aquasecurity/trivy/pkg/vex	0.692s
  ```

- Exit status: `0`.
- Pre-cleanup Git status proved the test was the only checkout change:

  ```text
  ## HEAD (no branch)
  ?? pkg/vex/source_order_regression_test.go
  ```

- Result: with repository-like `false` first and OCI-like `true` second,
  production `Client.NotAffected` returns `true` and attributes the result to
  `oci`. Reversing the order still returns `true` and attributes `oci`, because
  it is the first and only suppressing source. A non-suppressing result is not
  a veto; the operative rule is **first suppressing source wins**, not first
  configured source wins.
- Auditor limitation: this is a unit proof of the real
  `Client.NotAffected` contract using fake `vex.VEX` implementations. It does
  not instantiate repository/OCI loaders or perform an end-to-end image scan.
  In reverse order, OCI short-circuits and the trailing repository fake is not
  called. If both sources returned `true`, source order would select the first.
- Failed setup attempts retained as evidence:
  - the first delegated attempt was blocked by an automated classifier before
    producing a test result; its 112 MB temporary root was removed;
  - a full-history clone in the successful runner failed with
    `fetch-pack: unexpected disconnect while reading sideband packet`, exit
    `130`; the partial clone was removed before the shallow tag clone.
- Runner report before deletion: 175 lines / 4537 bytes, SHA-256
  `7128b1d4aa143b097dec82a72beb5c00a217a20a640a0058d56ab75583086d0b`.
- Independent auditor report before deletion: 118 lines / 5334 bytes,
  SHA-256
  `2928b14540444a90d5c8c1e1c9e10bec1709dc07c1c0b96aa7f58be81edddea1`.
- Cleanup proof:

  ```text
  PASS clone absent: /private/tmp/trivy-source-order-runner.s01S0G/trivy
  PASS test absent: /private/tmp/trivy-source-order-runner.s01S0G/trivy/pkg/vex/source_order_regression_test.go
  PASS temporary root absent: /private/tmp/trivy-source-order-runner.s01S0G
  PASS audit root absent: /private/tmp/trivy-source-order-auditor.QOQpGx
  ```

- No Docker command, publication, commit, or Tyk worktree access occurred.

### EXP-007 - Current moving custom DHI compiler base with public repository VEX

- Date: 2026-07-29.
- Interpretation scope: exact stock-Trivy/public-repository result retained as
  evidence of the interoperability gap. EXP-010 adds Docker's signed
  image-specific VEX for the same immutable platform digest and supersedes the
  repository-only affectedness interpretation.
- Purpose: determine whether Docker's refreshed moving customization removes
  EXP-005's active findings and establish the exact current stock-Trivy/public
  repository result without Docker daemon use.
- Moving input:
  `docker.io/tykio/dhi-busybox-plugin-compiler:1.37.0-debian13-fips_plugin-compiler-ng-toolchain`.
- Resolved index:
  `sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885`.
- Exact linux/amd64 scan target:
  `docker.io/tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`.
- Image config:
  `sha256:3588cfaca2d08544e614ff378d3ebdf333fa0ae1406f22c30a4d4b5a94608ba8`.
- Detected OS: Debian `13.6`.
- Trivy: `/opt/homebrew/bin/trivy` `0.72.0`, SHA-256
  `242c86fa4fb1304014631cd6b89638bb342dea11a1bf9fa56687fed5d2c18d90`.
- Trivy DB:
  - updated: `2026-07-29T07:47:23.42987139Z`;
  - downloaded: `2026-07-29T10:55:45.166486Z`;
  - manifest:
    `sha256:43f675dddccf9888eb9034f785a134c4f1f6446e53cd77136f5fbd6688eede34`;
  - layer:
    `sha256:728a29c1190b0fcbeb6cf6eb2c77b544c43fa6470744524d6246efdd3eb6caab`.
- Docker public advisories checkout:
  - commit: `b42ed4ce6b2862a3ebf4112d474d748d37a5b033`;
  - tree: `67774157f4fe00a45fbc42bec038c502ca20d47b`;
  - downloaded by Trivy at `2026-07-29T13:49:24.312234+03:00`;
  - a recursive diff against a fresh checkout, excluding `.git`, proved that
    the bytes consumed from Trivy's cache matched that checkout.
- Isolated repository configuration:

  ```yaml
  repositories:
    - name: docker
      url: https://github.com/docker-hardened-images/advisories
      enabled: true
      username: ""
      password: ""
      token: ""
      insecure: false
  ```

- The moving tag and linux/amd64 manifest/config were resolved with stock
  `skopeo` remote inspection. `$TMP` below replaces only the isolated temporary
  root:

  ```bash
  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/skopeo inspect \
    --creds "$registry_user:$registry_secret" --raw \
    docker://docker.io/tykio/dhi-busybox-plugin-compiler:1.37.0-debian13-fips_plugin-compiler-ng-toolchain \
    > $TMP/tag-index.json

  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/skopeo inspect \
    --creds "$registry_user:$registry_secret" --raw \
    docker://docker.io/tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3 \
    > $TMP/manifest-amd64.json

  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/skopeo inspect \
    --creds "$registry_user:$registry_secret" \
    --override-os linux --override-arch amd64 --config \
    docker://docker.io/tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3 \
    > $TMP/config-amd64.json
  ```

- Exact normalized repository download:

  ```bash
  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/trivy --cache-dir $TMP/trivy-cache --timeout 20m \
    vex repo init

  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/trivy --debug --cache-dir $TMP/trivy-cache \
    --timeout 20m vex repo download docker

  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    git clone --depth=1 --branch main \
    https://github.com/docker-hardened-images/advisories.git \
    $TMP/advisories-git

  diff -qr --exclude=.git \
    $TMP/trivy-cache/vex/repositories/docker/0.1 \
    $TMP/advisories-git > $TMP/vex-cache-vs-git.diff
  ```

- Exact normalized raw scan:

  ```bash
  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/trivy --debug --cache-dir $TMP/trivy-cache \
    --timeout 20m image --image-src remote --platform linux/amd64 \
    --username "$registry_user" --password "$registry_secret" \
    --scanners vuln --severity HIGH,CRITICAL --format json \
    --output $TMP/raw.json --no-progress --disable-telemetry \
    --skip-version-check \
    docker.io/tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3 \
    2> $TMP/raw-scan.log
  ```

- Exact normalized repository-VEX scan:

  ```bash
  HOME=$TMP/home XDG_CONFIG_HOME=$TMP/config XDG_CACHE_HOME=$TMP/cache \
    TMPDIR=$TMP/tmp \
    /opt/homebrew/bin/trivy --debug --cache-dir $TMP/trivy-cache \
    --timeout 20m image --image-src remote --platform linux/amd64 \
    --username "$registry_user" --password "$registry_secret" \
    --scanners vuln --severity HIGH,CRITICAL --format json \
    --output $TMP/repo-vex.json --no-progress --disable-telemetry \
    --skip-version-check --vex repo --show-suppressed \
    --skip-vex-repo-update \
    docker.io/tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3 \
    2> $TMP/repo-vex-scan.log
  ```

- Results:

  | Scan | Active Critical | Active High | Suppressed Critical | Suppressed High |
  | --- | ---: | ---: | ---: | ---: |
  | Raw | 17 | 75 | 0 | 0 |
  | Docker repository VEX | 9 | 50 | 8 | 25 |

- The 59 active repository-VEX findings reconcile exactly:
  - `linux-libc-dev@6.12.96-1+dhi0`: 1 Critical / 39 High;
  - `libperl5.40`, `perl`, `perl-base`, and `perl-modules-5.40`, all
    `5.40.1-6+dhi6`: 8 Critical / 8 High;
  - `libexpat1@2.7.1-2+dhi5`: 3 High.
- Change from EXP-005:
  - six util-linux packages were removed, eliminating six High occurrences;
  - `gpgv` was removed, eliminating `CVE-2026-24882`;
  - no EXP-005 finding changed because of a package upgrade or VEX
    suppression;
  - all 16 prior Perl occurrences remain active.
- The active Perl CVEs on each of the four binary packages are Critical
  `CVE-2026-13221` and `CVE-2026-57433`, plus High `CVE-2026-48962` and
  `CVE-2026-57432`. Their effective current public-repository status is
  `under_investigation`.
- The repository contains older DHI-specific `not_affected` Perl statements,
  but newer broadly matching `under_investigation` statements supersede them
  by timestamp in Trivy 0.72. This is a current non-suppressing decision, not a
  package-name matching failure.
- `linux-libc-dev` remains a package-index mapping gap: its binary package PURL
  is absent from Docker's VEX index.
- The three active `libexpat1` CVEs (`CVE-2026-56131`, `CVE-2026-56407`, and
  `CVE-2026-56408`) are absent from its selected VEX document.
- Report hashes:
  - raw JSON:
    `14dd8c57688b7d96437d459db8b044a54df8e3dda4aaa69f10fa4486aa311bdb`;
  - repository-VEX JSON:
    `bc55f1450b85e919b06dfd461a36e6723a26ba35481cd0732e59615bb8adb61c`.
- Result: PARTIAL. The moving customization removed seven High occurrences,
  but Docker's current public repository does not produce zero. A compatibility
  repository must never convert the current Perl `under_investigation`
  decision or absent `libexpat1` statements into `not_affected`.
- An independent subagent audited the arithmetic and conclusions and reported
  no material issues.
- Cleanup: the validated isolated root
  `/private/tmp/trivy-acceptance-TBi3ti7l` and all 3.3 GB of scan/cache/report
  data were removed and verified absent. No Docker CLI/daemon, publication,
  repository edit, or `../tyk` access occurred.

### EXP-008 - Current generated repository fails private-mirror discovery

- Date: 2026-07-29.
- Method: independent read-only audit of the uncommitted Gromit publisher
  against Trivy 0.72.0 production source. No Docker, network publication,
  repository edit, or `../tyk` access occurred.
- Trivy repository lookup removes version and qualifiers from package PURLs,
  except that it intentionally retains `repository_url` for OCI products:
  `pkg/vex/repo.go`.
- The current Gromit publisher indexes its document under an OCI PURL that
  contains the original image `repository_url`:
  `vexpublish/repository.go`.
- The generated portable product removes `repository_url`, but that product is
  evaluated only after Trivy has already selected and loaded a document:
  `vexpublish/publication.go`.
- Result: FAIL. After mirroring an image to another registry, Trivy computes a
  different OCI index key and never loads the current generated document.
- The documented apparent private-mirror success used
  `--vex "$OFFICIAL_VEX"` with a local file. It bypassed `--vex repo` and is
  not evidence for the north-star command.
- The current publisher records `network_publication_performed: false`; it
  plans a future Cosign `.att` tag but neither produces nor pushes one.
- Decision:
  - remove image-repository-indexed compatibility output;
  - generate a live composite repository indexed by exact Debian binary
    package PURLs, which are independent of the image registry namespace;
  - for every package key the composite repository owns, copy Docker's complete
    current binary-package document before adding verified source-to-binary or
    image-specific statements;
  - configure the Tyk composite first for its covered package keys and Docker's
    live repository second for every other package;
  - copy every current Docker status mechanically, including non-suppressing
    statuses, order conflicting statements deterministically from their source
    timestamps, and never invent `not_affected`;
  - continuously regenerate the composite from current Docker inputs; a static
    snapshot is not a customer delivery mechanism;
  - require zero OCI suppressions in Trivy 0.72 acceptance tests until
    EXP-003/006's conflict behavior is fixed.

### EXP-009 - Trivy 0.72 repository package-key precedence

- Date: 2026-07-29.
- Status: **PASS / DESIGN CONSTRAINT CONFIRMED**.
- Trivy source: tag `v0.72.0`, commit
  `8a32853686209a428179bb3a1688802b25691564`.
- Trivy `pkg/vex/repo.go` SHA-256:
  `65257a8c7ba089bdbd3ccd763ddc5575e041366d4dc8b442f7b26d882476f2e9`.
- Reproduction test SHA-256:
  `decc96ce219e340195f1c8480308d8d873f4b82728b95f71e334ee43a618e78c`.
- Go: `go1.26.3 darwin/arm64`.
- Exact command:

  ```bash
  gofmt -w pkg/vex/repository_precedence_repro_test.go
  go test ./pkg/vex \
    -run 'TestRepositorySetPrecedenceReproduction|TestRepositorySet_NotAffected' \
    -count=1 -v
  ```

- The production `RepositorySet.NotAffected` path was exercised with two
  configured repositories and a Debian package PURL carrying `arch`, `distro`,
  and `repository_url` qualifiers.
- Results:
  - first repository contains the package key but no matching CVE: the second
    repository is not consulted and the finding remains active;
  - first repository contains the package key and an
    `under_investigation` statement: the second repository is not consulted and
    the finding remains active;
  - first repository does not contain the package key: Trivy consults the
    second repository and applies its exact `not_affected` statement;
  - Debian package lookup removed version and all qualifiers, so its
    package-index key was `pkg:deb/debian/bash` and was independent of image
    registry identity.
- The complete test output SHA-256 was
  `31c24ed5f8919f1abf590b786faf7b5e21bdcb88982268ea44c629d12ea71772`.
- Result: a Tyk repository cannot be a sparse overlay for a package key also
  present in Docker's repository. Any Tyk-owned key must contain the complete
  applicable decision set that Trivy should evaluate for that key.
- Cleanup: the temporary test source and captured output were deleted; the
  verified Trivy checkout is clean. No Docker, registry, publication, Gromit,
  or Tyk repository write occurred.

### EXP-010 - Captured Docker-signed image VEX covers all 59 findings

- Observation window ended: `2026-07-29T11:40:48Z`.
- Status: **PASS FOR DOCKER EVIDENCE / FAIL FOR STOCK TRIVY CONSUMPTION**.
- Image:
  `docker.io/tykio/dhi-busybox-plugin-compiler`.
- Index:
  `sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885`.
- Linux/amd64 manifest:
  `sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`.
- Config:
  `sha256:3588cfaca2d08544e614ff378d3ebdf333fa0ae1406f22c30a4d4b5a94608ba8`.
- Public advisories commit and tree:
  `b42ed4ce6b2862a3ebf4112d474d748d37a5b033` /
  `67774157f4fe00a45fbc42bec038c502ca20d47b`.
- Eight repeated authenticated referrer enumerations returned the same
  `15`-descriptor AMD64 set.
- Final referrer response SHA-256:
  `80ba328b291761698c0fb2b1621fc8cb3b6a6616dc94c4d4a21a7e57bc8a2112`.
- Sorted referrer-set SHA-256:
  `b4114efb100f649b9b38e665d4f999a464e5c9e3ee973a3c926e950ddfacdb85`.
- Docker DHI public key SHA-256:
  - PEM:
    `1d02bbccf149283ae6288d96264dcad3fb23ee1911d90324a48eab28e4cb8a5f`;
  - DER:
    `118ba556dd52f4aec67018efd316c285c783cd3e54cc0f4527605715c643887c`.
- Three OpenVEX artifacts were stable and verified:

  | Scope | Manifest | Statement | Predicate | Signature artifact | DER signature |
  | --- | --- | --- | --- | --- | --- |
  | Base | `8bd1bff6ab64efaca98bc6dceb6a1ef37d01ca6b8a56c11fe9e4e87f211c32de` | `779111810e8ef5dee71353255781b4925f5228b365b2959d3a9a9dc96249353a` | `cf3d4a5d35c903e69ce5794ea0eaa9e2e9caa0929289ebe5445b72607bfe60f0` | `f9c5eb4d276af8316b012ad04ec4609851d438c070dd45ac0ea7426e8cebd0ec` | `a0066213e9fb4c1c30245ecabcae6051776c25b5f5c7473798a8a4e1ee7e42b2` |
  | Exception, Hub subject | `85efb50640a8749a726687e9102d827faaf07fdf7f1ae778f09ec654e8f0a26e` | `2476fdfae749263ddef9fe7c7a31f4dd2711efdbe30b5411e000f25aa978f8d1` | `f3da649741055652f73289ea3ff479dce65e91f60346dd918d9d8cece100d181` | `8e668f228d4be0ddd39d3497840c242847470187a945adc7c4641ab56c7ac2ab` | `9956b63ca48d32c8fc863eaf972cd2942c593f493004b84a52f5d644cf0bf353` |
  | Exception, internal subject | `6ce3cf17b1aafcac5c4d4b297499c294cbe3cbc220e84d60262e97d78f005f82` | `f7af6c08481844742a75fd62fd35c350c699c5f236b05e7a2b4a0c3bb4a5345f` | `f3da649741055652f73289ea3ff479dce65e91f60346dd918d9d8cece100d181` | `707abd8a4901cfea1996ec35b1c72a39cb9aae018bd2e5d37b0f03b66f2ad629` | `912413b0082e967956394288d5976c514663ed6f0e6a8797af7517ad459b0755` |

- The two exception predicates are byte-identical. Their OpenVEX document:
  - ID:
    `https://scout.docker.com/public/vex-97324fde87e7e5396cf0c147e3ae16a91592de0e36788577d20e01f895b61bed`;
  - author: `Docker Hardened Images <dhi@docker.com>`;
  - created: `2025-03-12T14:51:32Z`;
  - `last_updated`: `2026-07-29T04:48:10Z`;
  - statement count: `792`.
- `cosign verify --key <Docker-DHI-key> --insecure-ignore-tlog=true` passed
  claims validation for all three immutable VEX artifact digests. Direct
  `openssl dgst -sha256 -verify` passed for all three separately retrieved DER
  signatures and signed payloads.
- These are in-toto Statement v0.1 documents with separate Cosign
  simple-signing artifacts, not DSSE envelopes. They have no certificate,
  Fulcio chain, Rekor inclusion, or bundle. Docker-key authorship and exact
  AMD64 subject binding are verified; referrer-list freshness and transparency
  inclusion are not.
- The captured image-bound VEX says `not_affected` for the source-package
  versions behind every one of EXP-007's 59 active occurrences:

  | Trivy binary findings | Docker-signed source product | Captured image VEX | Why public repository leaves active |
  | --- | --- | --- | --- |
  | 40 `linux-libc-dev` | `pkg:deb/debian/linux@6.12.96-1%2Bdhi0?os_distro=trixie&os_name=debian&os_version=13` | all 40 `not_affected` | no `linux-libc-dev` index key |
  | 16 across four Perl binaries | `pkg:deb/debian/perl@5.40.1-6%2Bdhi6?os_distro=trixie&os_name=debian&os_version=13` | all 16 `not_affected` | newer broad `under_investigation` statements win Trivy timestamp ordering |
  | 3 `libexpat1` | `pkg:deb/debian/expat@2.7.1-2%2Bdhi5?os_distro=trixie&os_name=debian&os_version=13` | all 3 `not_affected` | selected `libexpat1` document lacks the CVEs |

- The public source-package documents still contain the exact DHI-version
  `not_affected` statements. No explicit exact-version revocation was found.
  The public Perl and `expat` documents also contain later broad
  `under_investigation` statements that Trivy treats as overriding.
- Docker's signed CycloneDX SBOM provides exact parent mappings:
  `linux-libc-dev -> linux`, `libexpat1 -> expat`, and the four Perl binaries
  `-> perl`.
- Docker Hub exposes no matching VEX referrers or legacy `.att` tags for the
  image index or platform digest. The verified VEX artifacts live in
  `registry.scout.docker.com`; an ordinary image mirror does not copy that
  external referrer graph, and Trivy does not query it automatically.
- Result: the DHI VEX record itself does not report these 59 occurrences as
  affected. Stock Trivy leaves them active because of package-identity,
  repository-selection, and statement-precedence interoperability gaps.
- A compatibility document must be described as Tyk-authored derived VEX based
  on Docker-signed VEX and SBOM evidence. Docker signatures do not cover the
  translated binary-PURL statements.
- Command retention limitation: exact enumeration and signature-verification
  commands and exact content digests were retained. The original artifact
  retrieval loop and temporary authentication-file construction were not
  retained byte-for-byte; only their exact ORAS command forms and digests were
  retained. Do not claim otherwise.
- Cleanup: both isolated evidence roots and nested auditor processes were
  removed. No Docker CLI/daemon, Trivy scan, publication, repository edit, or
  `../tyk` access occurred.

### EXP-011 - Real packageurl dependency invalidated mocked builder tests

- Date: 2026-07-29.
- Status: **FAIL / TEST GAP IDENTIFIED**.
- Candidate:
  `/Users/buger/go/src/tyk-vex-records-live/scripts/build_dhi_composite.py`.
- Python: Homebrew Python `3.14.6`.
- Dependency: `packageurl-python==0.17.5`, installed into an isolated temporary
  virtual environment from `requirements-dhi.txt`.
- Exact normalized commands:

  ```bash
  VENV=$(mktemp -d /private/tmp/tyk-vex-python.XXXXXX)
  python3 -m venv "$VENV/venv"
  "$VENV/venv/bin/pip" install \
    --disable-pip-version-check --no-input -r requirements-dhi.txt
  "$VENV/venv/bin/python" -B -m unittest \
    tests.test_build_dhi_composite -v
  ```

- Result: `11` tests ran; `8` errored, `1` failed, and `2` passed.
- Root cause: when `packageurl-python` was absent, the test suite replaced PURL
  parsing with a fixture-specific mock. The real dependency normalizes the
  signed-style OCI product
  `@sha256%3A...?repository_url=docker.io%2F...` to literal `sha256:` and `/`.
  The builder's canonical-string equality check therefore rejected the input
  before testing any Debian mapping behavior.
- Safety conclusion: a Docker-signed non-Debian product PURL must be validated
  and preserved byte-for-byte, not rewritten or rejected merely because a
  library has a different canonical spelling. Strict canonicalization remains
  appropriate for generated Debian package keys and mapped binary PURLs.
- Test conclusion: the dependency-present path is mandatory in CI. A fallback
  parser or mock must not be allowed to make the main builder suite pass.
- Cleanup: `/private/tmp/tyk-vex-python.9q2p3R` was removed and verified absent;
  no package was installed into the system interpreter.

### EXP-012 - Dependency-present tests passed but trust-boundary audit failed

- Date: 2026-07-29.
- Status: **FAIL / NO-SHIP**.
- Candidate hashes before remediation:
  - `requirements-dhi.txt`, pinned to `packageurl-python==0.17.6`:
    `410689e669905e6e1177160b30597586b6cc8ce7fa995d57df73f00b9525a591`;
  - `scripts/build_dhi_composite.py`:
    `f63c4b5d385f81a68798b00d7aff4fd50356460bce402d764428b3ad0712697b`;
  - `tests/test_build_dhi_composite.py`:
    `0370807c318b545b9a2a96ce509cffaffde97416773cfa04a0043f90c309f808`;
  - sorted fixture hash set:
    `16a7f0d221471edf75f1302fb0f623123e1d1aac29764f8615a6cd40ae65b66a`.
- Python: Homebrew Python `3.14.6`.
- Exact normalized command:

  ```bash
  VENV=$(mktemp -d /private/tmp/tyk-vex-python.XXXXXX)
  trap 'rm -rf "$VENV"' EXIT
  python3 -m venv "$VENV/venv"
  "$VENV/venv/bin/pip" install \
    --disable-pip-version-check --no-input -r requirements-dhi.txt
  "$VENV/venv/bin/python" -B -m unittest \
    tests.test_build_dhi_composite -v
  ```

- Result: all `12` tests passed with `packageurl-python 0.17.6`.
- Independent adversarial audit result: **NO-SHIP** despite the green suite:
  1. document `last_updated` was incorrectly allowed to make an older
     image-specific statement supersede a newer conflicting statement;
  2. image subcomponents were flattened and projected without requiring their
     signed parent OCI subject and SBOM root to match an expected exact image;
  3. Debian source/binary matching compared name and version but ignored OS and
     architecture qualifiers; and
  4. public repository documents were not required to have Docker's author.
- Missing tests included Debian epoch and `+bN` binNMU versions, unrelated OCI
  parents, qualifier mismatches, non-Docker public authors, multiple image
  documents/revocation, and complete same-name `perl` semantics.
- Decision: a passing deterministic-output suite is insufficient when the
  builder can broaden a signed `not_affected` decision outside its subject or
  Debian distribution. Fix every trust-boundary issue and rerun the
  dependency-present and adversarial suites before publication work.
- Cleanup: both temporary 0.17.5 and 0.17.6 virtual environments were removed;
  no `__pycache__` or matching `/private/tmp/tyk-vex-python*` directory
  remained. No Docker, registry, publication, tag, Gromit source, or `../tyk`
  write occurred during the builder test.

### EXP-013 - Subject-bound fail-closed builder passes adversarial audit

- Date: 2026-07-29.
- Status: **PASS FOR BOUNDED OFFLINE BUILDER / LIVE REPLAY PENDING**.
- Candidate hashes:
  - `requirements-dhi.txt`, pinned to `packageurl-python==0.17.6`:
    `410689e669905e6e1177160b30597586b6cc8ce7fa995d57df73f00b9525a591`;
  - executable `scripts/build_dhi_composite.py`:
    `99d962bc228e7974c1b32812f4638a6a3227c47141551b6497ceef656daf9ef3`;
  - `tests/test_build_dhi_composite.py`:
    `e8e19aeeacee3a4ef5223c844a7592ca923427de3345f77696d01eb1bffec00b`;
  - sorted fixture hash set:
    `f5ae4bde174c7638435a48e2929f10131afdeb026b560293acba37017f756358`.
- Python: Homebrew Python `3.14.6`.
- Dependency: `packageurl-python==0.17.6` in an isolated virtual environment.
- Exact normalized test command remained the EXP-012 command.
- Result: all `28` dependency-present tests passed. Ruff lint and formatting
  checks passed. A direct CLI build produced `7` valid JSON files and a
  deterministic five-member sorted archive.
- Fixed trust boundaries:
  1. statement timestamps, never document `last_updated`, determine history;
  2. every applicable generic or exact image/source/binary statement
     participates, so specificity cannot hide a newer decision;
  3. a same/newer conflicting source or binary statement fails closed;
  4. the expected immutable OCI PURL must match both the image-VEX parent and
     CycloneDX SBOM root byte-for-byte;
  5. image subcomponents are considered only under that exact parent;
  6. Debian SBOM qualifiers are restricted to `os_distro`, `os_name`,
     `os_version`, and optional `arch`, and every statement qualifier must be
     proven by the installed mapping;
  7. public and image VEX documents require Docker's exact author;
  8. signed-style non-Debian OCI PURLs are validated and preserved without
     canonical rewriting; and
  9. epoch, `+bN`, same-name `perl`, multiple image documents, revocation,
     output paths, and byte-identical archives have focused tests.
- Independent final audit result: **SHIP for the bounded builder**; no remaining
  unsafe `not_affected` path was found in the tested trust boundary.
- Important limitation: EXP-010's real Docker inputs include broad public
  statements whose statement timestamps are newer than older exact image
  statements. Docker's later image-document `last_updated` value is not a
  statement revocation or precedence rule under OpenVEX. The corrected builder
  must fail closed on such a conflict. A live replay may therefore remain
  non-zero or abort until Docker publishes chronologically unambiguous
  statements or an authenticated precedence contract.
- This experiment does not verify signatures, fetch Docker evidence, publish a
  repository, run stock Trivy, mirror an image, or meet the north star.
- Cleanup: all `/private/tmp/tyk-vex-*` directories and repository
  `__pycache__` directories were removed. No Docker, registry, publication,
  tag, Gromit source, or `../tyk` write occurred during builder validation.

### EXP-014 - Corrected builder consumes live Docker inputs and fails closed

- Date: 2026-07-29.
- Status: **FAIL CLOSED / NO CUSTOMER VEX OUTPUT**.
- Purpose: replay the bounded builder against Docker's actual signed
  CycloneDX and OpenVEX shapes and the current public advisory checkout,
  without Docker daemon use.
- Moving input:
  `docker.io/tykio/dhi-busybox-plugin-compiler:1.37.0-debian13-fips_plugin-compiler-ng-toolchain`.
- Remote `skopeo` resolution, using an ephemeral auth file populated through
  Docker's credential helper:
  - index:
    `sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885`;
  - linux/amd64:
    `sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`;
  - linux/arm64:
    `sha256:5d4ead5e820063b7fe3c0acaadfffcd331f6b37f667d74e2dc5bff8b922e851b`.
- Docker Scout: `1.20.4`.
- The live amd64 enumeration contained `15` descriptors. The normalized
  `{digest,predicateType,subject}` set SHA-256 was
  `cd08d0a5fc87c04b2114985fb057d95fa51ca9e8367dd08dd2afe32d5a346cfb`.
  The full Scout list is unsigned and is not proof of completeness or a
  monotonic current head.
- Selected Docker artifact descriptors:
  - CycloneDX:
    `sha256:29c2c61f0542d9b8ec6845451c7629b59510d3323d43fc6329b9f0269fa6543f`;
  - OpenVEX:
    `sha256:6ce3cf17b1aafcac5c4d4b297499c294cbe3cbc220e84d60262e97d78f005f82`,
    `sha256:85efb50640a8749a726687e9102d827faaf07fdf7f1ae778f09ec654e8f0a26e`,
    and
    `sha256:8bd1bff6ab64efaca98bc6dceb6a1ef37d01ca6b8a56c11fe9e4e87f211c32de`.
- `docker scout attestation get --verify --skip-tlog` passed Cosign claims and
  Docker-key verification for the SBOM and each VEX artifact. All four in-toto
  subjects contained the exact amd64 digest. Transparency-log inclusion was
  intentionally not proved.
- Docker DHI public key SHA-256:
  `1d02bbccf149283ae6288d96264dcad3fb23ee1911d90324a48eab28e4cb8a5f`.
  The key was retrieved from mutable `latest.pub` and checked against this
  previously recorded pin.
- Extracted predicate hashes:
  - CycloneDX:
    `b01db312bc18598b0b588224f4ab7f4b82d74cd5cf93026c98d051ddad43bb08`;
  - the two duplicate customization OpenVEX predicates:
    `4bd14393bc86548216c7de358dcc885399aef738201784f87878262305592846`;
  - the base OpenVEX predicate:
    `047babae48b013c3237f4cb9c4147a399596cc6f4dadefc7e92d16a4b5c06158`.
- Current Docker advisories checkout:
  - commit:
    `7df8eab421f5210155bcb1880486902a2d1da1bc`;
  - tree:
    `9a3836ffcf0af9e863933bd77c87fde2af486448`;
  - index `updated_at`: `2026-07-29T04:48:27Z`;
  - signed index SHA-256:
    `1e6a00f8a8b89bb2be82eb4748ba6ca4c6d4eb8c0d5b8aba8a47e845dcd64168`;
  - signed `binutils` document SHA-256:
    `26d2faf2297d32ba6090519b5e87540c207207cba98afc2efff301b8fad78bf2`.
- `cosign verify-blob` with Docker's pinned public key returned `Verified OK`
  for the index and the consumed `binutils` document. The separately examined
  `linux`, `perl`, and `expat` documents also passed.
- Live input corrections made before acceptance:
  1. require the exact digest-bound `pkg:docker` SBOM root separately from the
     exact tag-based OpenVEX parent;
  2. consume Docker's signed CycloneDX source-to-binary dependency edges,
     including source self-mapping, while rejecting missing, ambiguous,
     cross-release, duplicate, or malformed edges;
  3. consume Docker's direct Git checkout layout without an external rewrite
     and reject conflicting direct/`0.1` layouts;
  4. let genuinely unqualified and partially qualified compatible Docker
     products participate in chronology;
  5. let image statements naming the installed binary veto older source-level
     projections; and
  6. preserve Docker's original package PURL spelling in provenance while
     canonicalizing only internal index identity.
- Bounded-builder hashes before the independent ecosystem-shape audit
  (superseded by EXP-016):
  - `requirements-dhi.txt`:
    `410689e669905e6e1177160b30597586b6cc8ce7fa995d57df73f00b9525a591`;
  - `scripts/build_dhi_composite.py`:
    `e77434db5b5c026a701999cfc5e8ba882f8168cea9fd04e4efaa7a4baf187d88`;
  - `tests/test_build_dhi_composite.py`:
    `4e43665dfc96ea9e8cf40ab8602de62a195fd85964fe544bd1be64e3b4f5ae03`;
  - sorted fixture hash set:
    `9e1c28ae5a045fe9724c43aa8fbba293a1c43deb9a90d9b15bc54922e8cacf50`.
- Validation: all `42` dependency-present unit tests passed. Ruff lint and
  formatting checks passed.
- Exact live result:

  ```text
  error: conflicting public statement for CVE-2017-13716 is not older than exact image statement at 2026-05-10T11:35:31Z
  ```

- The image-specific Docker document contains a `not_affected` statement for
  the installed `binutils@2.44-3+dhi6` at `2026-05-10T11:35:31Z`. The current
  public `binutils` document contains a broader `under_investigation`
  statement at `2026-05-21T19:59:41.074Z`; its products include the
  unversioned `pkg:deb/debian/binutils`, so it overlaps the installed package
  even though the same statement also names Debian 12 `binutils@2.46.0`.
- CLI exit: `1`. No output directory or Trivy repository was created.
  Stderr SHA-256:
  `2eb4f7d719107c5b6ee9fc49fcf8b9ff0dfb18f73d78cebc75b3853330a3738b`.
- Safety conclusion: the captured Docker image VEX still records
  `not_affected`, but the signed public feed contains a later overlapping
  non-suppressing statement. Without a Docker-authenticated specificity,
  current-head, or revocation contract, Tyk cannot legitimately choose the
  older suppressing statement merely to reproduce the DHI console's zero.
- Remaining external trust boundary: the production retriever must verify
  signatures and prove that the SBOM and every VEX in-toto subject bind to the
  same platform digest before predicate extraction. The offline builder hashes
  inputs but intentionally does not perform network or signature operations.
- North-star consequence: repository, OCI, and private-mirror Trivy tests
  cannot proceed because the fail-closed builder correctly produced no VEX
  repository. No zero-Critical/High customer claim is available from this
  experiment.
- Cleanup: the `506M` evidence directory, both replay directories, the `36M`
  isolated virtual environment, and repository `__pycache__` directories were
  removed and verified absent. No Docker daemon data, image layer, registry
  mutation, tag, Gromit source, or `../tyk` write was created by this replay.

### EXP-015 - Unsupported VEX identity and product shapes fail closed

- Date: 2026-07-29.
- Status: **PASS FOR BOUNDED BUILDER HARDENING / LIVE NO-SHIP UNCHANGED**.
- Purpose: close the remaining accepted-input paths that could silently omit a
  newer Docker statement while preserving Docker input shapes already proved
  by the EXP-014 replay.
- Changes:
  1. every image-specific statement must use the exact expected image parent as
     its sole top-level product, with a non-empty set of Debian package-URL
     subcomponents;
  2. versioned, qualified, or subpath-bearing `pkg:deb/debian` public index keys
     fail closed because lookup uses Docker's generic package keys;
  3. public VEX products with nested subcomponents fail closed rather than
     being accepted and then ignored; and
  4. normal Docker vulnerability aliases remain accepted and preserved, while
     the consumed document set fails closed if one alias links more than one
     primary vulnerability identity.
- Regression controls cover parent-only, sibling-product, non-package and
  non-Debian subcomponent, versioned/qualified/subpath Debian index,
  public-subcomponent, and cross-primary alias inputs.
- Validation: all `46` dependency-present unit tests passed. Ruff lint and
  formatting checks passed. `git diff --check` passed.
- Final bounded-builder hashes:
  - `requirements-dhi.txt`:
    `410689e669905e6e1177160b30597586b6cc8ce7fa995d57df73f00b9525a591`;
  - `scripts/build_dhi_composite.py`:
    `c026bad8d6c4255a748ffb2f6a19e21b71a83b4250251239d436feda8f6e8dc1`;
  - `tests/test_build_dhi_composite.py`:
    `adcd31e562b1de1c73a9683fec47ecbeeecbd5487ca23e17d02e4e8ce589cd5f`;
  - sorted fixture hash set:
    `9e1c28ae5a045fe9724c43aa8fbba293a1c43deb9a90d9b15bc54922e8cacf50`.
- EXP-014's live evidence was intentionally not downloaded or replayed again:
  its immutable inputs were cleaned, and these validation changes cannot
  resolve Docker's signed `CVE-2017-13716` chronology conflict. EXP-014
  therefore remains the controlling live result: exit `1`, no compatibility
  repository, no stock-Trivy private-mirror proof, and no customer-ready zero
  claim.
- Cleanup: the isolated test virtual environments and all repository
  `__pycache__` directories were removed. Docker daemon state and `../tyk`
  remained untouched.

### EXP-016 - Debian image scope verified against signed Docker artifacts

- Date: 2026-07-29.
- Status: **PASS FOR INPUT-SCOPE HARDENING / LIVE NO-SHIP UNCHANGED**.
- Purpose: resolve an independent review finding that image subcomponents with
  a valid non-Debian package URL could pass structural validation and then be
  ignored by the Debian compatibility builder.
- Docker's immutable public index at EXP-014 commit
  `7df8eab421f5210155bcb1880486902a2d1da1bc` is intentionally
  multi-ecosystem: `33` APK, `27` Cargo, `812` Debian, `2` DHI, `15` Gem,
  `86` Go, `89` Maven, `112` npm, `4` NuGet, `576` OCI, and `21` PyPI keys.
  Those valid non-Debian public keys must remain accepted but are outside this
  builder's Debian lookup domain and cannot contribute output decisions.
- The three exact image-specific VEX artifact digests from EXP-014 were fetched
  again with Docker Scout `1.20.4` using the immutable index digest and
  `--verify --skip-tlog --predicate`. Docker-key/Cosign verification passed,
  and every in-toto subject again bound exact platform digest
  `sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`.
- Observed image subcomponent package types:
  - `sha256:6ce3cf17b1aafcac5c4d4b297499c294cbe3cbc220e84d60262e97d78f005f82`:
    `2620` Debian references;
  - `sha256:85efb50640a8749a726687e9102d827faaf07fdf7f1ae778f09ec654e8f0a26e`:
    `2620` Debian references;
  - `sha256:8bd1bff6ab64efaca98bc6dceb6a1ef37d01ca6b8a56c11fe9e4e87f211c32de`:
    `47` Debian references.
- No non-package or non-Debian image subcomponent was observed. The builder now
  requires every image subcomponent to be a `pkg:deb/debian` package URL with
  no subpath. The existing unsupported-image-shape test now also covers a
  valid `pkg:apk/alpine` subcomponent.
- Validation: all `46` dependency-present unit tests passed. Ruff lint and
  formatting checks passed. `git diff --check` passed.
- Current bounded-builder hashes:
  - `requirements-dhi.txt`:
    `410689e669905e6e1177160b30597586b6cc8ce7fa995d57df73f00b9525a591`;
  - `scripts/build_dhi_composite.py`:
    `91087b13d77ea20c1589365c2a51cb30d7fe8c64a2fe9a9f460b627c234946d5`;
  - `tests/test_build_dhi_composite.py`:
    `54992bf78bf7705f2238a5034eb5667e40c5c7474659e3d81c346469fb9d28b8`;
  - sorted fixture hash set:
    `9e1c28ae5a045fe9724c43aa8fbba293a1c43deb9a90d9b15bc54922e8cacf50`.
- This experiment validates input scope only. It did not rerun the
  compatibility build, publish a repository, or alter EXP-014's
  `CVE-2017-13716` conflict and no-output result.
- Cleanup: the temporary predicate directory and test virtual environment were
  removed and verified absent. No image, layer, tag, registry mutation, or
  `../tyk` write was created.

### EXP-017 - Trivy cross-source precedence remains unfixed upstream

- Date: 2026-07-29.
- Status: **REPRODUCED IN CURRENT RELEASE AND MAIN SOURCE / REPORT PREPARED**.
- Trivy `v0.72.0`, published 2026-06-30, remains the latest non-prerelease
  release. EXP-004's production-client test at release commit
  `8a32853686209a428179bb3a1688802b25691564` proves that source order cannot
  make a non-suppressing repository result veto a suppressing OCI result.
- Trivy `main` at
  `990d76568ecab5583381facd112bfd5ac6f4266b` retains the same
  `Client.NotAffected` behavior: it returns on the first source that grants
  suppression, while a non-suppressing result does not veto later sources.
- GitHub searches in `aquasecurity/trivy` returned no direct matching issue for
  `VEX OCI repository conflict`, `VEX precedence`, `stale VEX`, or
  `under_investigation not_affected VEX`.
- Prepared upstream report:
  `docs/trivy-vex-source-precedence-bug.md`, SHA-256
  `6a53becada98943852c0224ded11b6c9ba77a30489684f67fd1335293f7d12f8`.
- No Trivy issue was posted. This result does not resolve Docker's independent
  EXP-014 precedence/currentness conflict and does not permit a suppressive OCI
  fallback in the north-star command.

### EXP-018 - Live DHI composite repository passes stock-Trivy gates

- Date: 2026-07-29.
- Status: **LOCAL GENERATION AND SCANNER PROOF PASSED / MAIN DEPLOY PENDING**.
- Candidate implementation:
  `TykTechnologies/tyk-vex-records`, branch `releng/live-dhi-vex`, draft PR
  <https://github.com/TykTechnologies/tyk-vex-records/pull/5>.
- Moving compiler input:
  `tykio/dhi-busybox-plugin-compiler:1.37.0-debian13-fips_plugin-compiler-ng-toolchain`.
- Captured linux/amd64 subject:
  `tykio/dhi-busybox-plugin-compiler@sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`.
- Captured tag index:
  `sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885`.
- Selected signed Scout inputs:
  - CycloneDX SBOM:
    `sha256:29c2c61f0542d9b8ec6845451c7629b59510d3323d43fc6329b9f0269fa6543f`;
  - OpenVEX:
    `sha256:6ce3cf17b1aafcac5c4d4b297499c294cbe3cbc220e84d60262e97d78f005f82`,
    `sha256:85efb50640a8749a726687e9102d827faaf07fdf7f1ae778f09ec654e8f0a26e`,
    and
    `sha256:8bd1bff6ab64efaca98bc6dceb6a1ef37d01ca6b8a56c11fe9e4e87f211c32de`.
- Docker advisory input:
  - signed commit `a4e598c5a4bcd529bb28ed63d23b1caca33f3404`;
  - exact signing fingerprint
    `CFD34F826B30EDDC2DB4BB68566FA2B8D8312EB4`;
  - checked-in key SHA-256
    `01c4ecb52afdf9ee8b9b9f8d4b4cb657fe517119290e075d71459f72f80a34ad`.
- Docker attestation key SHA-256:
  `1d02bbccf149283ae6288d96264dcad3fb23ee1911d90324a48eab28e4cb8a5f`.
- The refresh verified each selected Scout statement with Docker's pinned key,
  bound it to the exact platform digest, verified the advisory commit in an
  isolated GPG home, verified signed advisory blobs where present, reconciled
  the signed SBOM with stock Trivy's exact package inventory, and rechecked the
  moving tag plus selected descriptor set before staging.
- Builder result: `162` installed package mappings, `73` consumed source
  documents, `3,532` emitted decisions, and `0` quarantines. The archive has
  `63` files, SHA-256
  `0a57af7c3cf2c9a3af667f57e598e1a110eb2c0d107c3ad37088e0560dd1d5bc`.
- Stock Trivy `0.72.0` compiler proof:
  - baseline: `17 Critical / 75 High` active;
  - generated repository: `0 Critical / 0 High` active;
  - all `92` baseline rows were matched and suppressed, with no extra or
    unexplained suppression.
- Stock Trivy `0.72.0` current Gateway DHI-base proof:
  - subject:
    `tykio/dhi-busybox@sha256:2b6d255c525e892dfcf1bac05ab3ca7910013ab87195c64b3cb7dd70745fe696`;
  - baseline: `0 Critical / 5 High` active;
  - generated repository: `0 Critical / 0 High` active;
  - all `5` rows were matched and suppressed.
- The customer path used a local HTTP VEX repository and unmodified Trivy
  repository loading. It did not use a Trivy fork, wrapper, local VEX file,
  `--ignore-unfixed`, Docker Scout's vulnerability summary, or the Docker
  daemon.
- Validation: `104` unit tests passed independently on Python `3.12` and
  `3.14`; Ruff lint/format, Python compilation, `actionlint`, embedded Bash
  syntax checks, workflow Action-tag resolution, and official tool/checksum
  checks passed.
- Local end-to-end evidence used Cosign `3.1.2`. The production workflow pins
  Cosign `3.1.2`, Docker Scout `1.23.1`, and Trivy `0.72.0`; their official
  release hashes were independently confirmed. The exact Linux toolchain will
  be exercised by the mandatory main-branch publication run.
- One initial full refresh failed closed at a Python builder invocation and
  staged no output. A diagnostic repeat with the same implementation and live
  subject completed successfully. The main workflow remains the authoritative
  repeatability gate and cannot deploy if any child command fails.
- GitHub Pages is enabled as public workflow deployment. Environments
  `vex-generation` and `github-pages` allow only `main`; the generation
  environment contains the two required DHI read credentials. The URL remains
  unproven until the main workflow deploys and anonymously verifies this
  run's exact four file hashes plus a stock-Trivy repository download.
- This is not yet the final private-registry mirror proof and does not make a
  suppressive OCI snapshot safe. Docker issue #1827 remains the residual
  unsigned-current-head limitation.

### EXP-019 - Hardened live refresh remains deterministic and fail closed

- Date: 2026-07-29.
- Status: **FINAL LOCAL GATES PASSED / MAIN DEPLOY PENDING**.
- Candidate implementation: the staged net diff of
  `TykTechnologies/tyk-vex-records` branch `releng/live-dhi-vex`, containing
  only the 18 intended Pages publication files relative to `origin/main`.
- The refresh copies both checked-in Docker trust roots into its private
  `0700` workspace as `0400` files before use, rechecks the Docker Cosign key
  hash, and verifies the captured advisory commit and tree rather than a
  mutable `HEAD` reference.
- The final full refresh consumed Docker advisory signed commit
  `4c3ac209a21006879e8b21dea11a4da5fdf14eb3` with tree
  `b95af46ac1a338970f1269021708d42556d5f82f` and the same signed compiler
  image index, platform, SBOM, and OpenVEX descriptors recorded in EXP-018.
- Builder result: `162` installed package mappings, `73` consumed source
  documents, `3,532` emitted decisions, and `0` quarantines.
- Stock Trivy `0.72.0` smoke result remained exactly `92` baseline rows to
  `0` active rows, with all `92` rows matched and suppressed and no extra or
  unexplained suppression.
- Final generated hashes:
  - repository archive:
    `4290a731568588563bf8df6c2f44e316410ed3902a11ae2ebdcfa7c6b7e1e5c8`;
  - root manifest:
    `8695b8f62337fd0d114094d1c8762a7d986066c0aceb8968197c43a519504b90`;
  - provenance:
    `fa24a2b263f865e39ed6ba5c66bf5b44bbad4e962141d3cad2dae2a6211e8c65`.
- The advisory repository advanced between EXP-018 and EXP-019. A normalized
  comparison found no semantic publication difference after removing only
  top-level generated timestamps; the archive changed for those timestamps.
- Validation: all `106` unit tests passed on Python `3.12` and `3.14`; Ruff
  lint and format checks, Python compilation, `actionlint`, every embedded
  workflow Bash block, staged-diff whitespace checks, and staged-diff secret
  pattern checks passed.
- Two independent bounded reviews found no deployment blocker after the
  private trust-copy and captured-commit fixes. The residual operational risk
  remains Docker issue #1827 and normal GitHub Pages cache propagation; the
  deployment job compares exact per-run hashes and therefore fails closed on
  stale Pages content.

### EXP-020 - Public Pages deployment and customer-path scan pass

- Date: 2026-07-29.
- Status: **PUBLIC DEPLOYMENT AND STOCK-TRIVY CUSTOMER PATH PASSED**.
- VEX PR #5 was squash-merged to `main` as
  `da695f92fc3117df611871c6af8d21bc22ac23ff`:
  <https://github.com/TykTechnologies/tyk-vex-records/pull/5>.
- Authoritative main workflow run `30485780304` passed both jobs:
  <https://github.com/TykTechnologies/tyk-vex-records/actions/runs/30485780304>.
  Live evidence generation, stock-Trivy smoke, four-file artifact validation,
  Docker credential cleanup, Pages deployment, exact-hash propagation wait,
  and anonymous stock-Trivy repository download all succeeded.
- Public repository:
  <https://tyktechnologies.github.io/tyk-vex-records>.
- Exact deployed SHA-256 values verified independently over anonymous HTTPS:
  - `index.html`:
    `a80d24016cd24351394776a512f0265c9a5fe480938b3d555ea652f66f2c335e`;
  - `.well-known/vex-repository.json`:
    `8695b8f62337fd0d114094d1c8762a7d986066c0aceb8968197c43a519504b90`;
  - `vex-repository-0.1.tar.gz`:
    `c6083059fc19fe183c409c3c1d4e08044a1c6a54888a37b69e017b4b50a450fb`;
  - `provenance.json`:
    `5d4a61317fb99977185a7387da46151623bb35ee2e785a566c44ec3b76726579`.
- Deployed provenance records Docker advisory signed commit
  `4c3ac209a21006879e8b21dea11a4da5fdf14eb3`, verified tree
  `b95af46ac1a338970f1269021708d42556d5f82f`, `162` exact installed-package
  mappings, `73` source documents, `3,532` decisions, and `0` quarantines.
  It explicitly records `vulnerability_statuses_authored: false`.
- Linux Actions stock Trivy `0.72.0` proof with the workflow's freshly
  downloaded database found `94` baseline High/Critical rows, `0` active rows
  with the generated repository, and all `94` exact baseline rows suppressed.
- Independent macOS stock Trivy `0.72.0` public-URL proof used the same immutable
  `linux/amd64` compiler digest and `--image-src remote`:
  - baseline: `17 Critical / 75 High` (`92` total);
  - public repository: `0` active, `92` suppressed;
  - baseline and suppression tuple sets were exactly equal;
  - every suppression source was
    `VEX Repository: tyk-dhi (https://tyktechnologies.github.io/tyk-vex-records)`.
- The Linux and macOS scanner environments reported different baseline counts
  (`94` and `92`) while both strict transition checks matched every baseline
  row and left zero active. The publication contract is finding-set based, not
  count based, so database/runtime drift cannot pass unless each exact current
  baseline row is represented by a repository suppression.
- The customer-path verification used no Trivy fork or wrapper, no local VEX
  file, no `--ignore-unfixed`, no Docker Scout summary, and no Docker daemon.
- The workflow emitted only GitHub's Node 20 deprecation notices for pinned
  official Actions running under Node 24. These were non-blocking compatibility
  annotations; all security and publication gates passed.
- An additional anonymous archive audit reproduced all four deployed hashes,
  confirmed that the manifest contains only the expected HTTPS archive URL,
  matched the manifest and archive hashes in provenance, and found no absolute
  or traversal paths, links, duplicate names, or unexpected archive entry
  types.
- Cleanup removed approximately `3.58 GiB` of task-specific temporary scanner
  databases, scan caches, tool archives, virtual environments, refresh output,
  and registry credentials. No task Trivy, refresh, or local HTTP process
  remains.
- No Tyk or Gromit product code was modified by this deployment, and no alpha
  tag was created.

## Customer Outcome

Customers must be able to mirror a Tyk Gateway or plugin-compiler image into a
private registry and scan its immutable digest with unmodified stock Trivy:

```bash
trivy image --scanners vuln --severity HIGH,CRITICAL \
  --vex repo --vex oci --show-suppressed \
  private.registry.example/namespace/tyk-plugin-compiler-ee@sha256:...
```

The configured live repositories must make Trivy apply every applicable
current Docker DHI decision. They may report zero inherited DHI OS Critical or
High findings only when the revision-bound Docker evidence leaves none active.
Digest-bound image evidence must be preserved in the Trivy-version-compatible
transport, but it must not independently suppress until EXP-003/006's
stale-source conflict is fixed. This exact command shape is the north-star
acceptance test.

The VEX integration must not hide:

- a package added or changed by a Tyk child layer;
- an application dependency finding;
- an unmatched CVE, package, version, image digest, or platform;
- a Docker decision other than a verified `not_affected` decision;
- stale, malformed, unsigned, revoked, or otherwise untrusted evidence.

Any remaining Tyk child-layer or application Critical or High finding must be
fixed by Tyk, not suppressed. Findings in Docker's retained package closure
remain visible until Docker publishes an updated closure or a current
applicable decision.

## Delivery Scope

- Gromit owns the generated Tyk workflows and image files.
- Keep the generated Tyk diff minimal and portable to `release-5.13`,
  `release-5.8`, and other supported branches through policy variables.
- Publish the NG compiler in parallel under the `-ng` tag family.
- Preserve CE, EE, and FIPS compiler variants.
- Build and test images sequentially. Do not test competing tag writers.
- Use the moving custom DHI tag, but resolve it once to an immutable digest in
  each build and retain that digest in evidence and image metadata.
- Publish SBOM and provenance attestations.
- Build a test-only Gateway helper for the plugin compile/load gate, and prove
  that it is absent from the published compiler image.
- Keep the existing compiler tags and workflow behavior intact.
- Never modify or revert `/Users/buger/go/src/tyk`. Generate only into
  `/Users/buger/go/src/gromit/tyk`.

## VEX Design

Docker is the affectedness authority. Current inputs can come from both
Docker's public Trivy-compatible advisory repository:

```text
https://github.com/docker-hardened-images/advisories
```

and Docker-signed image-specific VEX and SBOM artifacts for the exact DHI
subject. EXP-010 proved that the signed image VEX contains `not_affected`
decisions for every one of the 59 inherited findings left active by the public
repository alone. The public repository is therefore not a complete substitute
for the captured signed image evidence.

Trivy 0.72 stops at the first configured repository containing a package key,
even when that repository's document has no matching CVE or has a
non-suppressing status. A Tyk repository therefore cannot be a sparse
lower-priority overlay. For package keys requiring compatibility statements,
the proposed public Tyk repository must be a continuously refreshed composite
placed first. Each such key may contain only the applicable Docker-authored
statements obtained from verified inputs, plus verified mechanical
source-to-binary projections from Docker's signed SBOM. Publication must fail
closed until the workflow can establish that the selected input set is
complete enough for that image revision. Docker's repository remains second.

The compatibility output is Tyk-authored because Docker's signatures do not
cover translated binary PURLs or the generated archive. It must never be
presented as Docker-signed output or as a Tyk exploitability decision. Every
status, CVE, timestamp, rationale, and source product remains traceable to the
current Docker-authored input.

Each published Tyk image may retain digest-bound VEX evidence, but Trivy 0.72
discovers only Cosign's legacy digest-tagged `.att` transport, not a generic
OCI 1.1 referrer. If both are published for current and future clients,
registry-copy instructions and release validation must preserve and prove both
forms explicitly. Neither form may act as an independent stale suppressive
fallback: Trivy gives an OCI `not_affected` statement logical-OR behavior even
when the live repository has a newer non-suppressing decision. Suppression is
allowed only after a stock-Trivy conflict/veto design is proved safe or the
live repository completely accounts for the image and the OCI document
contains no independent suppressive decisions.

The abandoned custom `gromit dhi-vex`, schema-6 scanner, descriptor-retention,
and publisher design is historical experiment material, not the selected
implementation.

The implementation lives in the `tyk-vex-records` worktree. Its builder is an
offline deterministic transform. A separate fail-closed refresh orchestrator
retrieves and verifies Docker's current signed inputs, runs stock Trivy against
the exact immutable platform, invokes the builder, rechecks moving inputs, and
stages exactly the Pages manifest, archive, and provenance document.

The source repository remains private, while GitHub Pages is configured to be
public. The live workflow and protected environments now exist, but the first
main-branch deployment has not completed. Until that run succeeds and the
anonymous stock-Trivy check passes, there is no customer-ready VEX URL.

Docker's signed DHI records may identify a Debian source package while Trivy
reports its installed binary packages. Any compatibility alias must prove that
source-to-binary relationship from authenticated package/SBOM evidence. Never
describe a translated binary-package PURL as an exact Docker-signed product.

Trivy must be configured with the Docker repository and invoked with
`--vex repo`. For Trivy 0.72, the additional `--vex oci` proves discovery only
when the mirrored registry preserved the legacy Cosign `.att` tag; a generic
OCI 1.1 referrer alone is insufficient. That OCI evidence must not
independently suppress a decision that the live repository can later revoke.
`--show-suppressed` is required so customers can audit each applied decision
instead of seeing only a zero count.

Individual signature verification cannot prove that a registry response is a
complete current set rather than an older signed subset. Docker issue #1827
tracks that residual limitation. The selected implementation polls the live
tag and referrer inventory, verifies every selected input, rejects selected-set
or platform drift, polls signed advisory `main`, publishes only a successful
fresh generation, and exposes detailed provenance. It does not claim a signed
monotonic Docker checkpoint that Docker has not supplied.

The production refresh must remain sequential and bounded:

1. resolve each moving DHI input once;
2. retrieve and verify Docker's current evidence for that exact subject;
3. build the deterministic compatibility repository;
4. run repository-only, OCI-only, and combined stock-Trivy checks;
5. run the north-star test against a same-basename private mirror;
6. retain immutable inputs, raw/suppressed/active reports, and hashes; and
7. publish a reviewed immutable repository revision before updating one
   customer-facing current pointer.

Do not add a database, queue, custom vulnerability scanner, image runtime
helper, or concurrent publication writer.

## Required Proof

For each Gateway and NG compiler CE/EE/FIPS image:

1. build the final candidate;
2. push or copy it to a temporary private registry under a different namespace
   while preserving the final basename;
3. configure and freshly download Docker's current public Trivy VEX
   repository;
4. prove `--vex repo` and `--vex oci` independently, then scan the mirrored
   immutable reference with the exact north-star command from the top of this
   file;
5. verify the expected signed VEX attestation and exact subject separately
   because Trivy 0.72 does not verify Cosign signatures and exits zero when no
   OCI VEX is found;
6. parse machine-readable output or run a separate `--exit-code 1` gate to
   verify that active inherited DHI findings exactly match Docker's current
   applicable decisions, and verify zero only when that evidence leaves none
   active; command success alone is not proof;
7. retain the complete `--show-suppressed` output;
8. seed or retain negative controls proving unmatched OS and application
   findings remain active;
9. verify the original and mirrored manifest/platform digests are identical;
10. retain the commands, Trivy version and executable hash, database metadata,
   report hashes, signed attestation verification, and VEX repository commit
   as test evidence.

Routine unit and template tests must not require Docker. Use one bounded final
registry-mirror integration proof after the non-Docker gates pass.

## Evidence Record - Current And Superseded

This section preserves results produced before the 2026-07-29 scope decision.
Any item referring to `dhi-vex`, schema-6 evidence, descriptor execution,
`memfd`, the old `tyk-vex-records` publisher, or a direct local VEX file is a
superseded experiment, not current release architecture. The current VEX
evidence boundary is EXP-009 through EXP-017 and the offline compatibility
builder described in `VEX Design`; EXP-011 and EXP-012 remain recorded failed
tests, EXP-013 records fixture-level progress, EXP-014 controls the live
no-output result, EXP-015/016 record bounded fail-closed hardening without
superseding that live result, and EXP-017 confirms the separate Trivy
cross-source precedence defect remains current.

- Regenerated Tyk PR workflow run
  `https://github.com/TykTechnologies/tyk/actions/runs/30440534058` passed CE,
  EE, and FIPS native plugin load gates on 2026-07-29 with the import/module
  rewrite functionality removed. CE passed arm64 and s390x cross-builds;
  EE/FIPS passed arm64.
- Hosted compiler workflow run `30364329407` passed CE, EE, and FIPS native
  compile/load, cross-validation, and image-push gates.
- The published FIPS compiler candidate contains no production `tyk` binary;
  the Gateway helper is limited to the local test stage.
- The current master FIPS compiler uses native Go FIPS with
  `GOFIPS140=v1.0.0`; older branches can select their required mode through
  Gromit variables.
- The rebuilt FIPS compiler and its custom DHI base had the same 92 severe
  Trivy OS tuples: 17 Critical and 75 High. The previous alpha had seven
  additional child-image High findings.
- Historical R12 applied a locally generated, Tyk-authored compatibility
  projection of revision-bound Docker decisions to the exact DHI repository
  identity. It accounted for all 92 inherited findings and produced
  0 Critical / 0 High with the R12 target, Trivy database, authorization set,
  and report hashes recorded in the R12 evidence. Docker's signatures do not
  cover the translated bytes. This local-file proof does not pass the stock
  repository/OCI north star.
- The superseded Gromit pipeline's
  `go test ./vexpublish ./dhivex ./cmd -count=1` passed without Docker at that
  revision. Those packages are not part of the selected implementation.
- The superseded `tyk-vex-records` implementation passed 52 tests on Python 3.14
  and 3.9, including a stock Trivy 0.72 repository download and real scan.
- That superseded implementation generated an exact immutable product and a
  second portable product derived by removing only `repository_url`; its
  package tests passed with that pair.
- Stock Trivy 0.72 was reproduced without Docker using the locally generated
  compatibility VEX: its portable product matches a different registry and
  namespace when the final basename, digest, architecture, CVE, and package
  PURL are unchanged. Different basename, digest, or package identity remains
  active. This did not test live repository or OCI discovery.
- The Gateway's remaining High is `GHSA-hrxh-6v49-42gf` in the direct
  `google.golang.org/grpc v1.80.0` dependency. A clean isolated upgrade to
  `v1.82.1` changed only `go.mod` and `go.sum`; focused Gateway, coprocess,
  and OpenTelemetry tests passed.
- The local Docker engine is currently empty after the user restarted Docker.
- The superseded publisher experiment selected Docker Hub VEX from the
  canonical `pkg/oci/index.docker.io/...` archive path and rejected
  `pkg/oci/docker.io/...`.
- Tyk PR 8518's dependency-guard failures are the expected review gate:
  `go.mod` and `go.sum` require a `deps-reviewed` label. Replaying the change
  from clean `master` proved the 27-line module update is the minimal
  `go mod tidy` result for gRPC 1.82.1; the new gRPC module directly requires
  the updated xDS, Envoy, GCP detector, genproto, and validation modules.
- Independent review of the superseded scanner found that it recorded Trivy's
  binary hash without requiring a separately reviewed expected hash/version.
  That finding contributed to abandoning the custom scanner path.
- Independent compiler review found that policy sync did not yet list the
  deleted standalone NG base workflow, so a real downstream sync would leave
  that stale workflow active. It also found that the mutable Go image was not
  resolved once per run and that pushed digests/attestations lacked an explicit
  post-push check. Those compiler issues were fixed and covered by focused
  policy tests.
- Independent review of the superseded `tyk-vex-records` publisher found it
  was not protected by the documented Product Security
  environment/CODEOWNERS/validator contract, and that its fixtures did not
  require or test the portable second product against a different private
  registry namespace. Those findings were fixed in that experiment before the
  broader publisher was removed.
- The second compiler review found no high or medium issues. Its one low
  evidence-retention finding was fixed by creating metadata before validating
  the published digest; targeted policy tests and `actionlint` pass.
- Review of the superseded custom scanner found a replace-then-restore window between
  hashing and invoking the external Trivy executable, plus scanner/publisher
  compatibility-schema and CLI-hook validation gaps.
- That experiment then copied the regular
  independently pinned executable while hashing into a private `0700`
  directory, executed only the `0500` controlled copy, rejected duplicate
  version JSON keys, validated pins before the CLI scan hook, and verified
  compatibility parity for supplier, action-statement time, and statement
  version. Its tests passed before the custom scanner was abandoned.
- Product rejected NG import/module rewriting. The NG `rewrite-imports.go`
  helper, Dockerfile copy, invocation, and existing-module `go mod edit
  -module` are removed from both template and golden output. Existing module
  paths and Go imports are preserved; `plugin_id` isolates workspace/output
  and names the synthetic module only for module-less input.
- A later audit of that experiment found that the private Trivy copy was still
  executed through a mutable pathname and could be swapped/restored by a
  same-UID process. It also found case-folded `Version` aliases. Descriptor-
  bound execution and case-insensitive semantic duplicate rejection were then
  investigated.
- A final descriptor audit found that an unlinked ordinary filesystem inode
  remains mutable through another same-UID `/proc/<pid>/fd` handle. A sealed
  Linux `memfd` experiment was implemented and then removed: defending against
  a hostile same-UID process is outside the isolated release-runner threat
  model and a memfd does not address all same-UID process tampering. The
  practical controls remain an isolated runner, independently pinned Trivy
  version/SHA, and descriptor-bound execution of the verified copy.
- The superseded compatibility decoder rejected recursive case-folded field
  aliases and canonical-plus-alias collisions before typed decoding; its
  focused, normal, race, and vet checks passed.
- Gromit policy sync regenerated Tyk PR 8379 from an isolated temporary clone.
  The remote PR no longer contains `rewrite-imports.go`, `rewrite-imports`, or
  `go mod edit -module`; existing plugin modules and imports remain untouched.

## Failed Or Rejected Experiments

- Exact repository-name VEX matching works for the original remote image but
  fails after a Docker-local alias or private-registry mirror changes the first
  `RepoDigest`. This does not meet the customer outcome.
- `--ignore-unfixed` reduces scanner output but violates the customer policy
  and is not an acceptable fix.
- Bundling a static VEX file only into an image becomes stale and cannot be the
  sole distribution mechanism. A current public repository is required.
- The current private `tyk-vex-records` repository has no current release,
  release asset, or offline-verifiable GitHub attestation. Its expiring Actions
  artifacts are not a customer delivery mechanism.
- `trivy sbom --vex` did not apply image-product OpenVEX to the retained
  CycloneDX scan even though its root component carried the exact image PURL.
  An SBOM replay is therefore not an acceptable substitute for the final
  private-registry image scan.
- Image-bound VEX remains useful as signed evidence, but EXP-003 rejects a
  suppressive static fallback for Trivy 0.72. Continuously updated decisions
  must come through the live repository path until conflict veto semantics
  exist.
- Broad package or repository wildcards are unsafe because they can suppress a
  finding for unrelated content.
- The complex multi-writer publication-coordination experiment was rejected as
  unnecessary. The workflows are being restored to simple sequential
  build/test/publish behavior.
- The sealed-`memfd` Trivy experiment was rejected as unrelated hardening for
  the customer scan path. It added Linux-kernel complexity without making the
  stock customer command consume VEX more correctly.
- Removing Perl or replacing Debian Git with a Tyk-maintained minimal Git build
  was rejected. Docker's pre-provisioned Git package and its complete declared
  dependency closure remain part of the compiler toolchain. The customer issue
  is whether stock Trivy consumes the applicable signed DHI VEX decisions, not
  whether scanner-visible package metadata can be removed.

## Current Work

1. Keep Docker issue #1827 open as the residual authenticated-current-head
   limitation. Do not add a static suppressive OCI fallback.
2. Complete the same-basename private-registry mirror test before calling the
   combined repository-plus-OCI north star complete or promoting the alpha.
3. Add separately verified platform publications before claiming support
   beyond the current `linux/amd64` repository.
4. Monitor the scheduled main workflow and update pinned Actions when upstream
   releases remove the Node 20 compatibility annotations.

## External Gates

- Docker does not publish an authenticated, monotonic current head for each
  complete DHI decision-bearing referrer set. The scheduled publisher can
  verify observed signed inputs and detect drift during one run, but it cannot
  cryptographically prove that Scout did not omit an older signed descriptor.
  Docker issue:
  <https://github.com/docker-hardened-images/advisories/issues/1827>.
- The exact specificity/currentness concern and requested machine-readable
  rule are recorded in:
  <https://github.com/docker-hardened-images/advisories/issues/1827#issuecomment-5118790554>.
- The public repository is live and its main-branch publication run passed.
  Future scheduled runs must continue to pass the same signed-input,
  exact-transition, and post-deploy hash gates.
- The actual private-registry mirror proof remains required before alpha
  promotion and before declaring the combined north star complete.

## Branches And Pull Requests

- Gromit branch: `releng/plugin-compiler-ng-supply-chain`
- Gromit PR: <https://github.com/TykTechnologies/gromit/pull/517>
- Current pushed Gromit head: `c2f6b9bf9ef3b6c1c8d25265066887362ee5b8b1`
- Generated Tyk PR: <https://github.com/TykTechnologies/tyk/pull/8379>
- Current pushed Tyk head: `7ce44616d74c5dfbf286315e20884ae2b90dfc13`
- Separate Gateway gRPC security PR:
  <https://github.com/TykTechnologies/tyk/pull/8518>
  (`go.mod` and `go.sum` only, commit `6610948`)
- VEX publication infrastructure PR:
  <https://github.com/TykTechnologies/tyk-vex-records/pull/5>
  (merged as `da695f92fc3117df611871c6af8d21bc22ac23ff`)
- Intended next alpha tag: `v5.15.0-alpha9`

The public VEX deployment is complete. Do not create the alpha tag before the
remaining private-registry mirror proof and the separately requested release
decision.
