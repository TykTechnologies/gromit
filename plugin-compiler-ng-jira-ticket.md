# Next-Generation Tyk Plugin Compiler And DHI VEX Interoperability

Last updated: 2026-07-29

## Proposed Jira Title

Deliver parallel CE, EE, and FIPS Plugin Compiler NG images with Docker DHI
VEX interoperability for stock Trivy

## Tracking Links

- [Gromit PR #517](https://github.com/TykTechnologies/gromit/pull/517)
- [Tyk PR #8379](https://github.com/TykTechnologies/tyk/pull/8379)
- [Tyk VEX Records PR #5](https://github.com/TykTechnologies/tyk-vex-records/pull/5)
- [Docker advisories issue #1827](https://github.com/docker-hardened-images/advisories/issues/1827)
- [EXP-014 Docker precedence follow-up](https://github.com/docker-hardened-images/advisories/issues/1827#issuecomment-5118790554)

The pushed pull request heads do not yet contain every local correction
described here. Gromit must be the source of generated Tyk changes, and both
pull requests must be regenerated and reviewed before an alpha is tagged.

## Problem Statement

Tyk needs a next-generation plugin compiler that can ship beside the legacy
compiler while it is proven across customer environments. It must:

- compile and load real plugins for CE, EE, and FIPS Gateway variants;
- use current Docker Hardened Images for EE/FIPS compliance requirements;
- retain the smaller BusyBox-based runtime and pre-provisioned toolchain;
- publish SBOM and provenance for immutable release images;
- support master and release branches through Gromit variables; and
- produce security results customers can reproduce after copying an image to a
  private registry.

The compiler implementation and the VEX interoperability problem are related
but separate. A compiler can function correctly while a scanner fails to
consume Docker's DHI affectedness decisions.

## Settled Scope

The Git and Perl package-removal investigation is closed.

Plugin Compiler NG retains Docker's pre-provisioned Debian Git package and its
complete declared Perl dependency closure. This is the same package-management
contract Docker uses for the custom DHI image. Tyk will not force-remove Perl,
copy selected Git binaries out of their package, or maintain a separate
minimal Git distribution.

For the immutable DHI revision recorded in `goal.md` EXP-010, Docker's signed
image VEX marks the source-package versions behind all 59 inherited Critical
and High occurrences `not_affected`. This settles the package-removal question,
not current authoritative affectedness: EXP-014 found a later overlapping
`under_investigation` statement in Docker's signed public feed. The open work
is Docker's precedence/currentness contract, live VEX discovery,
package-identity interoperability, and stale-decision protection after
mirroring.

This ticket does not:

- replace the legacy compiler or legacy tags;
- make Plugin Compiler NG the default;
- implement a vulnerability scanner or a Trivy fork;
- use `--ignore-unfixed`, global ignores, or scanner-specific policy to
  manufacture zero findings;
- claim that VEX establishes FIPS compliance;
- promise that Grype or Prisma consumes Trivy's VEX repository protocol; or
- allow hand-maintained generated changes in Tyk.

## User Outcomes

The opt-in `-ng` compiler family will:

1. build plugins with the branch-matching Gateway source, Go toolchain, module
   graph, edition tags, and FIPS mode;
2. preserve an existing plugin's `go.mod` module path and source imports;
3. validate ELF architecture, Go build metadata, edition, FIPS mode, and the
   supported GLIBC ceiling;
4. load a real plugin through a matching test-only Gateway and exercise the
   plugin's HTTP behavior;
5. publish CE, EE, and FIPS variants beside the existing compiler;
6. prove that the final compiler image contains no test Gateway executable;
7. retain immutable SBOM, provenance, build-input, functional-test, and
   scanner evidence; and
8. let a customer run stock Trivy against a same-basename private-registry
   mirror and see the current Docker DHI decisions applied.

## Product Variants

The parallel tag family is:

```text
tykio/tyk-plugin-compiler:<gateway-version>-ng
tykio/tyk-plugin-compiler-ee:<gateway-version>-ng
tykio/tyk-plugin-compiler-fips:<gateway-version>-ng
```

| Variant | Build policy | Plugin targets |
| --- | --- | --- |
| CE | `goplugin` | amd64, arm64, s390x |
| EE | `goplugin,ee` | amd64, arm64 |
| EE FIPS | `goplugin,ee,fips` plus branch FIPS mode | amd64, arm64 |

The compiler image platform is Linux/amd64 and contains the native and cross
toolchains needed for those plugin targets.

## Ownership And Branch Sync

Gromit owns:

- branch policy and variables;
- compiler Dockerfile and script templates;
- generated GitHub Actions;
- CE, EE, and FIPS variant configuration;
- functional validation; and
- the expected generated Tyk file set.

Tyk receives generated output only through Gromit's normal render/sync path.
The current Tyk worktree must not be edited or reset manually. The resulting
Tyk pull request should contain the smallest generated diff needed for Plugin
Compiler NG.

Branch policy keeps FIPS behavior explicit:

| Branch | FIPS build mode |
| --- | --- |
| master | `ee,fips` with native `GOFIPS140=v1.0.0` |
| `release-5.13` | `ee,fips` with native `GOFIPS140=v1.0.0` |
| `release-5.8`, `release-5.8.15` | `ee,fips,boringcrypto` with `GOEXPERIMENT=boringcrypto` |

Adding variables does not automatically enable NG on a release branch. Each
branch rollout remains a separate reviewed decision.

## Compiler Architecture

### Immutable Inputs

The configured custom DHI base is intentionally a moving tag:

```text
tykio/dhi-busybox-plugin-compiler:
1.37.0-debian13-fips_plugin-compiler-ng-toolchain
```

Each workflow resolves it once at the start of the run and records the index
and selected platform digests. Every build, label, provenance check, test,
scan, and push in that run uses the resolved `tag@sha256:...` identity. A
later Docker rebuild is consumed by a later workflow run, not midway through
the current run.

The branch-selected `tykio/golang-cross:<GOLANG_CROSS>` tag is resolved once
under the same rule.

### One Path: Docker's Pre-Provisioned Closure

Every build uses Docker's custom pre-provisioned package closure. The compiler
toolchain is never installed at build time, on any path, and there is no
fallback base. The Dockerfile verifies that each expected tool, the trust
bundle, and the package records are present in the base and fails the build if
any is missing.

This is a security requirement. Only packages Docker provisions into the
customization are covered by Docker's maintenance obligation, carry `+dhi`
versions, and have published DHI vulnerability decisions. A toolchain package
APT-installed at build time would be an ordinary upstream build that nobody is
obliged to patch and that no DHI advisory covers -- it would silently widen the
image's unmaintained surface.

Two consequences follow, both accepted deliberately:

- Changing the toolchain package list requires Docker to rebuild the
  customization first. Tyk cannot add a compiler dependency unilaterally.
- Pull requests build the same base the release builds, so the compile/load
  gate validates the artifact that actually ships rather than a Debian
  stand-in.

### Gate And Publish Forms

For each variant, the workflow runs sequentially:

1. build a local gate form containing the test-only Gateway;
2. compile native and cross-platform plugins;
3. validate architecture, Go, edition, FIPS, and GLIBC properties;
4. load and execute a real plugin through the matching Gateway;
5. stop before any push if a gate fails;
6. build the publish form from the same immutable inputs without the Gateway;
7. prove the Gateway is absent from the publish form;
8. push the immutable version and approved `-ng` aliases; and
9. inspect SBOM and provenance at the pushed digest.

CE, EE, and FIPS run in configured order. Tag-triggered workflows share one
non-cancelling concurrency group so competing releases do not write the same
alias concurrently.

`plugin_id` isolates compiler workspace and output. It names a synthetic
module only when the input has no `go.mod`; it never rewrites a supplied module
path or Go imports.

## Current Functional Evidence

Hosted Tyk workflow
[30440534058](https://github.com/TykTechnologies/tyk/actions/runs/30440534058)
passed:

- CE native compile/load plus arm64 and s390x cross-builds;
- EE native compile/load plus arm64 cross-build;
- FIPS native compile/load plus arm64 cross-build; and
- module/import preservation after removal of `rewrite-imports.go`.

The test Gateway is limited to the local gate stage. The final compiler form
contains no production `tyk` executable or symlink.

These results prove the compiler workflow, not the final VEX customer outcome.
The pull request still requires regeneration from the final Gromit source and
all normal repository checks.

## Current Docker VEX Evidence

The exact custom DHI revision examined on 2026-07-29 was:

```text
repository: docker.io/tykio/dhi-busybox-plugin-compiler
index:      sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885
amd64:      sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3
config:     sha256:3588cfaca2d08544e614ff378d3ebdf333fa0ae1406f22c30a4d4b5a94608ba8
```

Raw Trivy reported 17 Critical and 75 High. Docker's public Trivy repository
left 9 Critical and 50 High active. The inherited findings left active by the
public repository included:

| Binary findings | Docker-signed source product | Captured Docker image VEX |
| --- | --- | --- |
| 40 `linux-libc-dev` | `linux@6.12.96-1+dhi0` | all 40 `not_affected` |
| 16 across four Perl binaries | `perl@5.40.1-6+dhi6` | all 16 `not_affected` |
| 3 `libexpat1` | `expat@2.7.1-2+dhi5` | all 3 `not_affected` |

Docker's signed CycloneDX SBOM records the exact binary-to-source package
relationships. Docker's DHI public key verified the three selected VEX
artifacts and their exact image binding.

The captured image-specific DHI evidence accounts for all 59 occurrences as
`not_affected`, but it is not sufficient by itself for a current zero claim:
EXP-014 found a later overlapping `under_investigation` statement in Docker's
signed public feed and correctly produced no repository. Stock Trivy also
cannot consume the source-scoped image evidence in its current form. Exact
evidence digests and verification commands are recorded in `goal.md` EXP-010
and the live conflict is recorded in EXP-014.

## Trivy Interoperability Gap

This is a VEX delivery problem, not an open assessment of Git, Perl, or the
other inherited DHI packages. Docker remains the decision author. Tyk must
make the current Docker evidence consumable by stock Trivy without changing
its status, timestamp, rationale, or authority.

The root cause has three parts:

1. Docker's signed image-specific VEX artifacts live in
   `registry.scout.docker.com`, outside the Docker Hub referrer graph copied by
   a normal image mirror.
2. Docker identifies the Debian source package while Trivy reports the
   installed binary package. Trivy does not apply Docker's signed SBOM
   source-to-binary relationship.
3. Trivy 0.72 stops at the first VEX repository that contains a package key.
   A lower-priority repository cannot fill a missing CVE or supersede an
   inapplicable broad statement for that key.

Multiple VEX sources are also suppressive with logical-OR behavior. A stale
OCI `not_affected` statement can suppress a finding even when the current
repository contains a newer non-suppressing decision. OCI evidence must not
be used as an independent stale fallback until this behavior is fixed or an
equivalent current-state contract is proven.

## Minimal Compatibility Architecture

The selected direction is a minimal Docker-to-Trivy package-identity
projection in `TykTechnologies/tyk-vex-records`.

The transform accepts separately verified:

- Docker's current public advisory checkout;
- Docker's signed image-specific VEX for an exact DHI subject; and
- Docker's signed CycloneDX SBOM for the same subject.

It emits:

- a deterministic Trivy 0.1 package-index repository;
- a deterministic archive; and
- provenance recording source selection, input hashes, and package mappings.

The transform does not:

- scan an image or decide whether a vulnerability is exploitable;
- retrieve network data or Docker credentials;
- verify signatures itself;
- mutate a registry or attach an OCI artifact;
- write GitHub branches, releases, or tags; or
- rewrite Docker's status, CVE, source product, timestamp, or rationale.

For mapped binary package records, Docker's signed SBOM proves the mapping and
Docker's VEX supplies the decision. The generated bytes are Tyk-authored
compatibility data because Docker did not sign those translated binary PURLs.
They must never be labeled Docker-signed output or a Tyk vulnerability
assessment.

Missing, ambiguous, stale, conflicting, or unverifiable evidence fails closed.
An unresolved or affected finding remains active.

The previous experimental `gromit dhi-vex`, schema-6 scanner, anonymous
descriptor execution, and offline publisher architecture is not part of this
delivery.

EXP-014 proves that the current bounded implementation consumes Docker's real
`pkg:docker` SBOM root, CycloneDX dependency edges, tag-based OpenVEX parent,
and direct advisory checkout. Its `42` tests pass, but the live run produces no
repository because Docker's signed channels disagree chronologically:

```text
CVE-2017-13716:
image VEX  2026-05-10  not_affected
public VEX 2026-05-21  under_investigation
```

EXP-015 closes additional silent-input paths for image product structure,
non-generic Debian index keys, nested public products, and cross-primary
vulnerability aliases. The hardened bounded builder passes `46` tests, but
EXP-016 verifies the Debian-only image scope against all three exact signed VEX
artifacts while retaining Docker's unrelated public-index ecosystems.
EXP-014 remains the controlling live result and still emits no repository.

The later public statement includes unversioned
`pkg:deb/debian/binutils`, so it overlaps the installed DHI package. Choosing
the older suppressing statement requires a Docker-authenticated precedence or
currentness contract that does not exist today.

## Freshness And Publication

Image-bound VEX alone is not a sufficient live customer feed because it can
become stale while Docker's decisions change.

The production flow must be a small sequential refresh:

1. resolve the configured DHI input to exact digests;
2. retrieve and verify Docker's current signed image VEX and SBOM;
3. retrieve the current Docker advisory repository;
4. build the deterministic compatibility repository;
5. run unit, conflict, and deterministic-output tests;
6. run stock Trivy in repository-only, OCI-only, and combined modes;
7. test a same-basename immutable private-registry mirror;
8. publish an immutable reviewed repository revision; and
9. update one customer-facing current pointer only after all gates pass.

No concurrent tag writers, queue, database, custom scanner, or image runtime
helper is required.

Docker signatures authenticate selected artifacts, but the observed Scout
referrer list does not currently provide a signed monotonic current-head or
complete-set guarantee. That freshness/completeness boundary must be explicit
in the release control and is tracked with Docker.

`TykTechnologies/tyk-vex-records` is currently private. No anonymous customer
endpoint or live release exists yet.

## Customer North Star

After copying the released image into a private registry while preserving its
basename, the customer runs unmodified stock Trivy:

```bash
trivy image --scanners vuln --severity HIGH,CRITICAL \
  --vex repo --vex oci --show-suppressed \
  private.registry.example/namespace/tyk-plugin-compiler-ee@sha256:...
```

This test passes only when:

- the referenced digest is identical to the released immutable digest;
- Trivy uses the current Docker repository plus the Tyk compatibility
  repository in the tested order;
- every suppression traces to a current applicable Docker decision;
- source-to-binary mappings trace to Docker's signed SBOM;
- `--show-suppressed` exposes each applied decision;
- raw, suppressed, and active counts reconcile exactly; and
- child-layer, application, unmatched, `affected`, and
  `under_investigation` findings remain active.

Zero Critical and zero High is an expected result only when the current
Docker evidence leaves no inherited DHI OS finding active. Command exit status
alone is not proof.

## Test Plan

### Gromit And Generation

- run focused policy, branch-variable, template, and golden-file tests;
- run `go test ./...`;
- validate generated GitHub Actions syntax;
- render the intended Tyk branch only from Gromit;
- verify generated output is reproducible and minimal;
- confirm the legacy compiler files remain unchanged; and
- verify release-branch native FIPS and BoringCrypto variables.

### Compiler Functionality

- build CE, EE, and FIPS gate forms sequentially;
- compile native and supported cross-platform plugins;
- validate ELF, Go version, edition, FIPS, and GLIBC requirements;
- load a real plugin using the separate test Gateway;
- execute the HTTP behavior test;
- prove a failed gate prevents the publish build and push;
- build the publish form from the same immutable inputs; and
- prove no Gateway binary or symlink exists in the publish form.

### VEX Builder

- install the exact pinned Python dependency in an isolated environment;
- reject malformed and noncanonical generated Debian PURLs;
- validate and preserve signed non-Debian product PURLs;
- test exact-version, epoch, binNMU, architecture, and OS qualifiers;
- test same-name source and binary packages such as `perl`;
- test source-to-binary mappings from the signed SBOM;
- test newer conflicting Docker statements and fail-closed behavior;
- test missing source records and incomplete first-repository keys;
- prove repeated builds are byte-identical; and
- prove no Tyk-authored affectedness decision appears in output.

### Stock Trivy Integration

- record the Trivy binary version and hash plus vulnerability DB metadata;
- retain a raw scan without VEX;
- test Docker repository VEX alone;
- test OCI VEX alone and verify whether discovery occurred;
- test both sources and retain `--show-suppressed` output;
- mirror the image under a different namespace with the same basename;
- verify original and mirrored manifest/platform digests are identical;
- run the exact north-star command against the private mirror;
- retain raw, suppressed, and active machine-readable reports; and
- include negative controls for child and unresolved findings.

### Other Evidence

- run current Grype separately and retain its native package inventory;
- run Prisma in the customer environment with current Intelligence Stream and
  customer policy;
- retain Docker Scout/DHI console evidence separately; and
- perform the FIPS compliance gate independently of VEX.

## Rollout

1. Complete and independently audit the minimal VEX builder.
2. Complete the sequential verification and publication workflow.
3. Make `tyk-vex-records` anonymously readable only after repository and
   Product Security approval.
4. Run the stock-Trivy private-mirror proof for Gateway and compiler variants.
5. Finalize Gromit and regenerate Tyk.
6. Rerun all hosted compiler gates from the final generated Tyk revision.
7. Create the next master alpha tag only after every required gate passes.
8. Give customers immutable image digests, VEX repository configuration,
   scanner commands, and retained reports.
9. Evaluate `release-5.13` and `release-5.8` enablement separately.

The existing alpha8 images are historical test artifacts. They are not the
final customer VEX release.

## External Dependencies

- approval to make `TykTechnologies/tyk-vex-records` anonymously readable;
- reliable read access to current Docker-signed DHI VEX and SBOM artifacts;
- a reviewed freshness/completeness control for Docker's current artifact set;
- a minimal sequential publication workflow;
- Product Security review of the trust and fail-closed boundaries; and
- customer Prisma access for final customer-policy validation.

No Docker-heavy local workflow is required for routine unit tests. The final
private-registry integration proof should run once in bounded hosted
infrastructure after non-Docker checks pass.

## Risks

- Trivy repository precedence can let an incomplete first key block a correct
  later repository.
- Trivy's OCI discovery does not currently query Docker's external Scout
  artifact graph.
- Trivy combines suppressive sources without a revocation veto, so stale OCI
  VEX can hide a newer non-suppressing repository decision.
- A compatibility repository can become stale unless it is regenerated from
  current verified Docker evidence.
- Docker signatures cover source VEX and SBOM evidence, not the Tyk-generated
  compatibility bytes.
- A changed final image basename can prevent image-product matching.
- Grype and Prisma have different package inventories and VEX capabilities.
- VEX is not evidence of FIPS module status or runtime policy.
- A moving DHI tag can change between runs; every run must resolve it once and
  retain the exact digest.

## Acceptance Criteria

- Gromit remains the source of all generated Tyk compiler changes.
- The Tyk diff is minimal and contains no manual-only compiler files.
- CE, EE, and FIPS gate and publish forms pass all functional checks.
- Existing plugin module paths and imports are preserved.
- The published compiler image contains no test Gateway.
- Moving DHI and Go tags are resolved once per run and then used immutably.
- Trusted builds use the pre-provisioned DHI closure without APT installation.
- Docker's signed image VEX and SBOM are verified for the exact DHI subject.
- The Git and Perl package closure is retained; package deletion is not used to
  alter scanner output.
- The compatibility builder is deterministic and introduces no Tyk-authored
  vulnerability decision.
- Missing, conflicting, stale, or unverifiable evidence fails closed.
- `tyk-vex-records` is anonymously readable before a customer command is
  published.
- Stock Trivy passes the exact private-mirror north-star command without a
  fork, wrapper, ignore file, or fixability filter.
- Raw, suppressed, and active findings reconcile, and every suppression traces
  to current Docker evidence.
- Legitimate Tyk child-layer and unresolved findings remain active.
- Grype, Prisma, and FIPS results are reported as independent evidence.
- The final Gromit and regenerated Tyk pull requests pass normal checks before
  the next alpha tag is created.

## Current Status

As of 2026-07-29:

- the hosted compiler workflow has passed CE, EE, and FIPS functional gates;
- Docker's signed image VEX has been verified to cover the 59 inherited
  findings missed by the public repository;
- the Trivy package-identity and precedence failures have source-level
  reproductions;
- the Git/Perl removal question is closed;
- the minimal offline compatibility builder has passed its corrected
  46-test trust-boundary suite, including actual Docker input shapes and
  unsupported-shape negative controls;
- the live verification/publication workflow is not implemented;
- `tyk-vex-records` is private;
- the EXP-014 live replay did fail closed on Docker's conflicting
  `CVE-2017-13716` statements and created no VEX repository;
- document `last_updated` is not a safe precedence rule, and Docker has not
  published an authenticated alternative;
- the stock-Trivy private-mirror north-star proof is not complete; and
- no new alpha should be tagged yet.
