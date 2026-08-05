# Tyk Plugin Compiler NG Security Scanning

Last updated: 2026-07-29

## Purpose

This document defines the security-scanning contract for Plugin Compiler NG.
It separates four different facts that must not be collapsed into one claim:

1. the findings produced by an unmodified scanner;
2. Docker's current signed DHI affectedness decisions;
3. any Tyk-authored package-identity translation needed by that scanner; and
4. the remaining findings in Tyk-added layers or application dependencies.

The Git and Perl package-removal investigation is closed. The compiler retains
Docker's pre-provisioned Debian Git package and its declared Perl dependency
closure. Their presence in an SBOM or raw scanner report is not the problem
being solved.

## Customer Contract

An enterprise customer must be able to copy a released Gateway or Plugin
Compiler NG image into a private registry and use stock Trivy:

```bash
trivy image --scanners vuln --severity HIGH,CRITICAL \
  --vex repo --vex oci --show-suppressed \
  private.registry.example/namespace/tyk-plugin-compiler-ee@sha256:...
```

The acceptance test uses the mirrored immutable digest and preserves the
released image basename. It does not use:

- a Tyk Trivy fork, patch, wrapper, or custom vulnerability scanner;
- `--ignore-unfixed`, an ignore file, or a global suppression;
- a local-only image alias or source-DHI scan as a substitute for the mirror;
- a static VEX file presented as a permanently current feed; or
- a zero-only summary without the raw and suppressed records.

The expected active DHI OS result is zero Critical and zero High only when the
current Docker evidence actually reaches that result. Tyk child-layer,
application, unmatched, `affected`, and `under_investigation` findings remain
active.

## VEX Delivery Question

Docker remains the vulnerability-decision authority for inherited DHI
packages. For the immutable DHI revision in `goal.md` EXP-010, Docker's signed
image-specific VEX records the source-package versions behind all 59 inherited
Critical and High occurrences as `not_affected`. EXP-014 found a later
overlapping `under_investigation` statement in Docker's signed public feed, so
those captured image-specific records alone do not settle current
authoritative affectedness.

The remaining engineering question is how to deliver those decisions to stock
Trivy:

1. make Docker's current applicable image-specific decisions discoverable
   after the child image is mirrored;
2. project Docker's signed SBOM source-to-binary relationships to the exact
   package PURLs Trivy reports as explicitly Tyk-authored compatibility bytes
   that preserve Docker's decision fields and provenance without claiming
   Docker signed the translation; and
3. define a live freshness, completeness, and revocation contract so stale
   repository or image-bound records cannot hide a newer Docker `affected` or
   `under_investigation` decision.

The signed image VEX and SBOM are the machine-consumable evidence to transport
for that revision. The DHI console is corroborating UI evidence. None of these
are permission to create blanket `not_affected` statements, and each later
image or VEX revision must be evaluated independently.

## Current Evidence

The current custom compiler base observed on 2026-07-29 was:

```text
docker.io/tykio/dhi-busybox-plugin-compiler
index:    sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885
amd64:    sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3
config:   sha256:3588cfaca2d08544e614ff378d3ebdf333fa0ae1406f22c30a4d4b5a94608ba8
```

Raw Trivy reported 17 Critical and 75 High. Docker's public Trivy repository
left 9 Critical and 50 High active. Of those active findings, 59 occurrences
were inherited DHI OS packages:

| Trivy binary package | Occurrences | Docker-signed source package |
| --- | ---: | --- |
| `linux-libc-dev` | 40 | `linux@6.12.96-1+dhi0` |
| four Perl binary packages | 16 | `perl@5.40.1-6+dhi6` |
| `libexpat1` | 3 | `expat@2.7.1-2+dhi5` |

For the same immutable amd64 image, the Docker-signed image VEX captured in
EXP-010 marks all 59 occurrences `not_affected` through those source package
products.
Docker's signed CycloneDX SBOM contains the corresponding binary-to-source
relationships:

```text
linux-libc-dev -> linux
libexpat1       -> expat
Perl binaries  -> perl
```

The VEX artifacts and their Cosign signatures were verified with Docker's DHI
public key. Exact artifact, signature, statement, SBOM, image, and public
advisory digests are retained in `goal.md` EXP-010.

This proves that removing Git or Perl is not required to reconcile the DHI
console. The unresolved defect is stock Trivy interoperability.

## Why Stock Trivy Misses The Decisions

Three independent behaviors currently matter:

1. Docker's signed image-specific VEX graph is hosted under
   `registry.scout.docker.com`. It is not a Docker Hub referrer graph copied by
   a normal image mirror, and Trivy does not query it automatically.
2. Docker's image VEX identifies Debian source packages while Trivy reports
   installed binary packages. Stock Trivy does not perform the signed SBOM's
   source-to-binary reconciliation.
3. Trivy 0.72 stops repository lookup at the first repository containing a
   package key. A later repository cannot replace an older broad
   `under_investigation` selection or fill a missing CVE in that key.

Trivy also applies suppressions from multiple VEX sources with logical-OR
behavior. A stale OCI `not_affected` statement can therefore suppress even if
a live repository contains a newer non-suppressing decision. Until this is
fixed upstream or protected by an equivalent current-state contract, OCI VEX
must not act as an independent suppressive fallback.

The source-level reproductions and exact Trivy revision are retained in
`goal.md` EXP-003, EXP-006, and EXP-009.

## Compatibility Boundary

The proposed Tyk repository is a scanner-compatibility projection, not a Tyk
security assessment.

For every projected statement:

- the affectedness status, CVE, source product, timestamp, and rationale come
  from current Docker-authored evidence;
- source-to-binary mapping comes from Docker's signed SBOM;
- generated Debian package keys and product PURLs are canonical;
- the original Docker statement and input hashes remain traceable;
- non-Debian signed product PURLs are validated and preserved byte-for-byte;
- no `under_investigation` or `affected` decision becomes `not_affected`;
- missing, ambiguous, stale, conflicting, or unverifiable input fails closed;
- the generated compatibility bytes are identified as Tyk-authored; and
- Docker signatures are never claimed to cover the translated bytes.

Docker remains authoritative for all inherited DHI decisions. Tyk-authored VEX
for Gateway application dependencies is a separate publication stream and
must not be mixed with Docker-derived OS-package compatibility records.

## Current Implementation State

The old experimental `gromit dhi-vex`, schema-6 scanner, descriptor-retention,
and offline publisher design is not the selected implementation. Its R12
outputs remain historical experiment evidence only.

The current prototype is an offline deterministic builder in the
`TykTechnologies/tyk-vex-records` worktree. It accepts:

- a separately verified Docker advisories checkout;
- a separately verified Docker CycloneDX SBOM; and
- separately verified Docker image VEX documents.

It emits a Trivy 0.1 package-index repository, deterministic archive, and
provenance describing source resolution. It performs no network access,
signature verification, registry mutation, image build, GitHub write, tag
push, or vulnerability detection.

The builder is not yet a customer service. The repository is private, no
anonymous release URL exists, and the complete live verification/publication
workflow has not been implemented or approved.

The first 2026-07-29 adversarial audit was no-ship. EXP-012 records unsafe
statement supersession, missing parent-image/SBOM binding, ignored qualifiers,
and missing Docker-author enforcement. EXP-013 fixed those fixture-level
findings. EXP-014 then corrected the builder for Docker's actual signed input
shapes:

- a digest-bound `pkg:docker` SBOM root;
- a separate exact tag-based OpenVEX parent;
- CycloneDX source-to-binary dependency edges;
- direct GitHub repository layout;
- unqualified and partially qualified Docker products; and
- image statements that name only the installed binary.

The resulting `42` dependency-present tests and Ruff checks pass. The live
replay validates the real input shapes and then deliberately exits `1` without
creating a repository:

```text
error: conflicting public statement for CVE-2017-13716 is not older than exact image statement at 2026-05-10T11:35:31Z
```

Docker's image VEX says `not_affected` for the installed
`binutils@2.44-3+dhi6` on May 10. Docker's current signed public document has a
broader overlapping `under_investigation` statement dated May 21. A timestamp
does not prove authorship, completeness, or currentness, and document
`last_updated` cannot safely override statement chronology.

EXP-015 adds fail-closed validation for unsupported image-product structure,
non-generic Debian index keys, nested public products, and cross-primary
vulnerability aliases. Its `46` dependency-present tests and Ruff checks pass.
EXP-016 additionally verifies that all three exact signed image VEX artifacts
use Debian subcomponents while Docker's public index is intentionally
multi-ecosystem; image subcomponents are now restricted to `pkg:deb/debian`
without rejecting unrelated public index ecosystems. This hardening does not
supersede EXP-014's live no-output result.

The bounded builder is therefore behaving correctly, but it remains no-ship:
without a Docker-authenticated specificity/currentness rule, it must not emit
the older suppressing decision merely to reproduce the DHI console's zero.

## Required Live Publication Design

A production refresh must be sequential and fail closed:

1. resolve each configured DHI tag once to an index and platform digest;
2. retrieve Docker's current advisory repository, signed image VEX, and signed
   SBOM for that exact subject;
3. verify Docker signatures, exact subject binding, and the approved evidence
   completeness/currentness contract;
4. generate the deterministic compatibility package index;
5. run stock Trivy against raw, repository-only, OCI-only, and combined modes;
6. run the north-star scan against a same-basename private-registry mirror;
7. retain raw, suppressed, and active counts plus all input and output hashes;
8. publish an immutable reviewed repository revision and then update one
   customer-facing current pointer.

Do not run concurrent writers for the same publication pointer. Do not add a
database, queue, custom scanner, image runtime helper, or hidden policy engine.

Docker's signed artifacts prove authenticity of the selected documents, but
the observed Scout referrer list does not itself provide a signed monotonic
current-head value. The release design must not claim automatic freshness
until that completeness/currentness contract is resolved.

## Build Inputs

The workflow configuration intentionally uses the moving custom DHI tag:

```text
tykio/dhi-busybox-plugin-compiler:
1.37.0-debian13-fips_plugin-compiler-ng-toolchain
```

Resolve it once at the start of each workflow run and use only the resulting
`tag@sha256:...` identity for that run. A later Docker rebuild is picked up by
a later run; the base must not change midway through a build.

Resolve the branch-selected `tykio/golang-cross:<GOLANG_CROSS>` tag in the same
way. Record both immutable inputs in image metadata and retained evidence.

Every build -- pull request and release alike -- uses Docker's pre-provisioned
closure. The compiler toolchain is never installed at build time and there is no
fallback base: a base image missing any expected toolchain component fails the
build. This is a security requirement, not a convenience. Only packages Docker
provisions into the customization are covered by Docker's maintenance
obligation, carry `+dhi` versions, and have published DHI vulnerability
decisions. A package APT-installed at build time would be an ordinary upstream
build that nobody is obliged to patch and that no DHI advisory covers.

The consequence is deliberate: changing the toolchain package list requires
Docker to rebuild the customization first. In exchange, the pull-request
compile/load gate validates the same artifact that ships.

## Plugin Compiler Functional Gate

Security scanning does not replace the compiler's functional proof. For CE,
EE, and FIPS variants:

1. build a local gate image with the test-only Gateway enabled;
2. compile native and supported cross-platform plugins;
3. validate ELF architecture, Go version, edition tags, FIPS mode, and the
   GLIBC ceiling;
4. load a real plugin through the matching test-only Gateway and exercise its
   HTTP behavior;
5. build the publish form from the same immutable inputs; and
6. prove the publish form contains no Gateway executable or symlink.

The compiler preserves an existing plugin module path and imports. `plugin_id`
only isolates the workspace and names a synthetic module when no `go.mod`
exists.

## Other Scanners

Grype and Prisma results remain independent evidence:

- Grype does not use Aqua's VEX repository protocol and currently does not
  perform the same source-to-binary package reconciliation.
- Prisma Cloud acceptance must run in the customer's Console with the current
  Intelligence Stream and the customer's actual policy.
- No Trivy compatibility result may be presented as a Grype or Prisma result.

Raw scanner output must always be retained. Scanner-specific duplicate
cataloging, package identity, or VEX ingestion differences must be explained,
not hidden with global ignores.

## FIPS Boundary

VEX does not establish FIPS compliance. The FIPS gate separately verifies:

- Docker's signed FIPS evidence for the exact image subject;
- the OpenSSL FIPS provider, configuration, and approved boundary;
- native `GOFIPS140` or explicitly supported legacy BoringCrypto behavior;
- Gateway startup and TLS behavior; and
- plugin compilation and loading under the selected branch policy.

`release-5.8` and `release-5.8.15` retain
`ee,fips,boringcrypto` with `GOEXPERIMENT=boringcrypto`. Master and
`release-5.13` use `ee,fips` with native `GOFIPS140=v1.0.0`.

## Release Gate

A Plugin Compiler NG image is not ready for customer VEX validation until:

1. Gromit-generated CE, EE, and FIPS workflows pass their functional gates;
2. the final immutable customer-facing digests, SBOMs, and provenance exist;
3. Docker evidence is retrieved and verified for those exact DHI subjects;
4. the compatibility builder passes dependency-present unit and negative
   tests;
5. a public anonymous VEX repository revision is available;
6. stock Trivy passes the private-mirror north-star test without ignores;
7. raw, suppressed, and active findings reconcile exactly;
8. child-layer and unresolved findings remain active;
9. Grype, Prisma, and FIPS evidence is reported independently; and
10. Product Security approves the publication and freshness controls.

As of 2026-07-29, items 5 through 10 are not complete. No alpha or existing
local R12 result should be presented as satisfying this final customer gate.
EXP-014 also blocks item 4's live-input portion: the tested builder passes its
unit suite but correctly produces no repository from the conflicting current
Docker inputs.
