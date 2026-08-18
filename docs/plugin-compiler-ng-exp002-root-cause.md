# Plugin Compiler NG EXP-002 Root-Cause Classification

Date: 2026-07-29

## Result

This section is a historical EXP-002 repository-only snapshot. EXP-010 and
EXP-014 supersede its affectedness interpretation and currentness.

At the recorded EXP-002 revisions, Docker's public Trivy VEX repository
suppressed 33 of 99 Critical/High findings in the Plugin Compiler NG EE alpha8
image. The 66 active findings divided into:

| Classification | Critical | High | Total |
|---|---:|---:|---:|
| Product/subcomponent PURL mismatch | 1 | 43 | 44 |
| Effective status is not `not_affected` | 8 | 14 | 22 |
| No statement anywhere in repository | 0 | 0 | 0 |
| Installed-package version mismatch | 0 | 0 | 0 |
| **Active** | **9** | **57** | **66** |
| Suppressed by repository VEX | 8 | 25 | 33 |
| **Total represented** | **17** | **82** | **99** |

The 44 PURL mismatches were scanner/repository interoperability gaps. Under
that repository-only snapshot, the 22 newer `under_investigation` decisions
were intentionally non-suppressing and had to remain active.

## Immutable Inputs

- Trivy: `/opt/homebrew/bin/trivy` `0.72.0`
- Trivy SHA-256:
  `242c86fa4fb1304014631cd6b89638bb342dea11a1bf9fa56687fed5d2c18d90`
- Image:
  `docker.io/tykio/tyk-plugin-compiler-ee@sha256:443a2f8496232d41023007d7228df5a105556565f4ea36c759f47a69001f0a62`
- Image config:
  `sha256:7007e09773192f6df46c8fd970f2b716931ed9c32015c36b2fe2821eae4e8dac`
- Platform: `linux/amd64`
- OS: Debian `13.6`
- Docker advisories commit:
  `b42ed4ce6b2862a3ebf4112d474d748d37a5b033`
- Docker advisories tree:
  `67774157f4fe00a45fbc42bec038c502ca20d47b`
- VEX cache ETag:
  `W/"758c4f3c814cf9533db678fff534b94d8a243abdb4f5a279b0b7fa72c4ecc1e7"`

The Trivy-downloaded VEX directory was byte-identical to the checked-out
advisories tree above.

## Trivy Database

- Repository: `mirror.gcr.io/aquasec/trivy-db:2`
- Manifest:
  `sha256:43f675dddccf9888eb9034f785a134c4f1f6446e53cd77136f5fbd6688eede34`
- Layer:
  `sha256:728a29c1190b0fcbeb6cf6eb2c77b544c43fa6470744524d6246efdd3eb6caab`
- Updated: `2026-07-29T07:47:23.42987139Z`
- Downloaded: `2026-07-29T10:22:57.669483Z`

## Command

```bash
/opt/homebrew/bin/trivy \
  --cache-dir "$TMP/cache/trivy" \
  --timeout 15m \
  --debug image \
  --image-src remote \
  --platform linux/amd64 \
  --scanners vuln \
  --severity HIGH,CRITICAL \
  --vex repo \
  --show-suppressed \
  --format json \
  --output "$TMP/work/report.json" \
  docker.io/tykio/tyk-plugin-compiler-ee@sha256:443a2f8496232d41023007d7228df5a105556565f4ea36c759f47a69001f0a62
```

No Docker command or daemon was used.

## Product/Subcomponent PURL Mismatches

### gpgv

- Installed package: `gpgv@2.4.7-21+deb13u1+dhi3`
- PURL:
  `pkg:deb/debian/gpgv@2.4.7-21%2Bdeb13u1%2Bdhi3?arch=amd64&distro=debian-13.6`
- High: `CVE-2026-24882`
- Cause: Docker has statements for source package `pkg:deb/debian/gnupg2`,
  but no `gpgv` index entry/product.

### libexpat1

- Installed package: `libexpat1@2.7.1-2+dhi5`
- PURL:
  `pkg:deb/debian/libexpat1@2.7.1-2%2Bdhi5?arch=amd64&distro=debian-13.6`
- High: `CVE-2026-56131`, `CVE-2026-56407`, `CVE-2026-56408`
- Cause: Docker has statements for source package `pkg:deb/debian/expat`.
  The indexed `libexpat1` document has no matching statements.

### linux-libc-dev

- Installed package: `linux-libc-dev@6.12.96-1+dhi0`
- PURL:
  `pkg:deb/debian/linux-libc-dev@6.12.96-1%2Bdhi0?arch=all&distro=debian-13.6`
- Critical: `CVE-2026-43185`
- High:
  - `CVE-2013-7445`
  - `CVE-2019-19449`
  - `CVE-2019-19814`
  - `CVE-2021-3847`
  - `CVE-2021-3864`
  - `CVE-2024-21803`
  - `CVE-2024-58015`
  - `CVE-2025-22104`
  - `CVE-2025-38137`
  - `CVE-2025-38187`
  - `CVE-2025-38204`
  - `CVE-2025-38206`
  - `CVE-2025-38421`
  - `CVE-2025-38636`
  - `CVE-2025-39859`
  - `CVE-2025-39862`
  - `CVE-2025-39958`
  - `CVE-2026-23102`
  - `CVE-2026-23208`
  - `CVE-2026-23327`
  - `CVE-2026-31493`
  - `CVE-2026-31536`
  - `CVE-2026-31568`
  - `CVE-2026-43198`
  - `CVE-2026-43263`
  - `CVE-2026-46130`
  - `CVE-2026-46181`
  - `CVE-2026-46279`
  - `CVE-2026-52991`
  - `CVE-2026-53000`
  - `CVE-2026-53010`
  - `CVE-2026-53089`
  - `CVE-2026-53091`
  - `CVE-2026-53109`
  - `CVE-2026-53118`
  - `CVE-2026-53277`
  - `CVE-2026-53330`
  - `CVE-2026-63970`
  - `CVE-2026-64017`
- Cause: statements target source package `pkg:deb/debian/linux`,
  `libcpupower*`, or image products. There is no `linux-libc-dev` index entry.

## Newer Non-Suppressing Decisions

This section records the historical public-repository-only interpretation.
EXP-010's signed image-specific VEX evidence supersedes it for the immutable
DHI revision examined later. It is retained to explain stock Trivy's
repository behavior, not as an open package-removal recommendation.

### util-linux binary packages

The following `2.41-5+dhi3` packages each retain High
`CVE-2026-53615`:

- `pkg:deb/debian/libblkid1@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/liblastlog2-2@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/libmount1@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/libsmartcols1@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/libuuid1@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/util-linux@2.41-5%2Bdhi3?arch=amd64&distro=debian-13.6`

The effective statement is `under_investigation`, dated
`2026-07-13T20:11:42.146Z`. It is newer than the matching
`not_affected` statements.

### Perl binary packages

The following `5.40.1-6+dhi6` packages each retain:

- Critical: `CVE-2026-13221`, `CVE-2026-57433`
- High: `CVE-2026-48962`, `CVE-2026-57432`

Packages:

- `pkg:deb/debian/libperl5.40@5.40.1-6%2Bdhi6?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/perl@5.40.1-6%2Bdhi6?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/perl-base@5.40.1-6%2Bdhi6?arch=amd64&distro=debian-13.6`
- `pkg:deb/debian/perl-modules-5.40@5.40.1-6%2Bdhi6?arch=all&distro=debian-13.6`

Newer matching `under_investigation` statements override older DHI-specific
`not_affected` statements. Trivy/OpenVEX orders matching statements by
timestamp and selects the latest effective statement.

## Consequences

1. The 44 PURL mismatches are valid candidates for a Docker advisories
   interoperability fix or a carefully evidenced source-to-binary projection.
2. The 22 `under_investigation` findings must not be suppressed by a Tyk VEX
   projection.
3. A new image may reduce the 22 only if it removes the affected packages or
   contains versions no longer reported by the current Trivy database.
4. The public VEX repository, by itself, cannot legitimately produce zero for
   this immutable alpha8 image at these database/advisory revisions.

Item 3 describes a scanner-count mechanism, not the selected remediation.
Product scope now accepts Docker's pre-provisioned Git and Perl dependency
closure. No package-removal or custom Git work remains; the open issue is
faithful discovery and matching of Docker's applicable DHI VEX decisions.

## Current Moving Base Follow-up

This section also records the public-repository-only view. The
image-specific evidence in the following section supersedes its affectedness
interpretation for the same immutable platform digest.

EXP-007 scanned the refreshed moving customization at:

```text
index: sha256:dac3425c548dc62ef0b99f2484ba24df9f02443a5292c5afb2410fa6776d7885
amd64: sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3
config: sha256:3588cfaca2d08544e614ff378d3ebdf333fa0ae1406f22c30a4d4b5a94608ba8
```

With Trivy 0.72.0 and Docker advisories commit
`b42ed4ce6b2862a3ebf4112d474d748d37a5b033`, the current counts are:

| View | Active Critical | Active High | Suppressed Critical | Suppressed High |
| --- | ---: | ---: | ---: | ---: |
| Raw | 17 | 75 | 0 | 0 |
| Docker public repository | 9 | 50 | 8 | 25 |

Docker removed the six util-linux binary packages and `gpgv`, eliminating
seven High occurrences without VEX suppression. The remaining 59 active
findings are:

- `linux-libc-dev`: 1 Critical / 39 High due to the unresolved package-index
  mapping gap;
- four Perl binary packages: 8 Critical / 8 High with effective current
  `under_investigation` decisions;
- `libexpat1`: 3 High CVEs absent from its selected VEX document.

The refreshed result narrows the work but does not change the safety boundary.
A compatibility repository may add evidenced binary-package aliases, but it
must not turn a current `under_investigation` decision or an absent statement
into `not_affected`.

## Signed Image VEX Supersedes The Repository-Only Interpretation

The preceding conclusion is correct for Docker public advisories commit
`b42ed4ce6b2862a3ebf4112d474d748d37a5b033` as stock Trivy 0.72 evaluates it.
It is not the complete current Docker DHI decision set.

EXP-010 subsequently retrieved image-bound VEX observed on 2026-07-29 for the
same
linux/amd64 manifest,
`sha256:832d86c084f84ce83a14404847e8da2f3642a72633d03868565982d8a315b4d3`,
from `registry.scout.docker.com`. Three immutable OpenVEX artifacts passed
Cosign claims validation and direct P-256 verification with Docker's DHI
public key. The exception document declared `last_updated` as
`2026-07-29T04:48:10Z`; that field does not prove registry completeness,
current-head selection, or precedence over a later statement.

That captured signed image VEX says `not_affected` for the exact
source-package versions behind all 59 findings:

| Binary findings reported by Trivy | Docker-signed source package | Signed status |
| --- | --- | --- |
| 40 `linux-libc-dev` | `linux@6.12.96-1+dhi0` | `not_affected` |
| 16 across the four Perl binaries | `perl@5.40.1-6+dhi6` | `not_affected` |
| 3 `libexpat1` | `expat@2.7.1-2+dhi5` | `not_affected` |

Docker's signed CycloneDX SBOM records the corresponding binary-to-source
parent relationships. The public source-package documents also retain the
exact DHI-version `not_affected` statements; no explicit exact-version
revocation was found. Stock Trivy nevertheless leaves the findings active
because:

1. it indexes the scanner-observed Debian binary package, not Docker's source
   package;
2. it does not follow the signed SBOM parent relationship;
3. `linux-libc-dev` has no repository index key and `libexpat1` lacks the three
   CVEs in its selected binary document;
4. for Perl, newer broad `under_investigation` statements win timestamp
   ordering over the more specific exact-DHI-version statements;
5. Trivy scans the mirrored image repository and does not automatically query
   Docker Scout's separate referrer graph.

The selected remediation is therefore not package removal and not a new Tyk
exploitability assessment. It is a continuously refreshed, Tyk-authored
mechanical compatibility projection of current Docker-signed source decisions
and Docker-signed SBOM mappings to the exact installed binary PURLs. Docker's
signatures authenticate the retained inputs, not the translated output bytes.
Any explicit exact-version Docker revocation or unsupported mapping must fail
closed and remain visible.

## Closed Decision: Retain Docker's Git And Perl Closure

Debian 13's `git 1:2.47.3-0+deb13u1` binary package declares `perl` and
`liberror-perl` as hard runtime `Depends`, not `Recommends` and not source
`Build-Depends` ([Debian package metadata](https://packages.debian.org/en/trixie/git)).
The compiler installs Git with `--no-install-recommends`, but that cannot omit
required dependencies.

Go's direct/private-module path may use only Git's C-built core commands, but
removing Perl while retaining Debian's `git` package would leave a broken
package closure. A Perl-free design requires a separately built and maintained
minimal Git package, such as a reviewed Git build with Perl features disabled,
and corresponding DHI customization, SBOM, update, and compatibility gates.

### Decision

Retain Docker's pre-provisioned Debian Git package and its complete declared
dependency closure, including Perl. Do not force-remove Perl, copy selected Git
binaries out of the package, or introduce a Tyk-maintained minimal Git build.
The remaining work is VEX interoperability: determine why stock Trivy does not
apply the relevant signed DHI decisions to the customized compiler image and
its private-registry mirror.

For the immutable DHI revision examined in EXP-010, Docker's signed
image-specific VEX already marks the source-package versions behind all 59
inherited Critical and High occurrences `not_affected`. Future Docker image or
VEX revisions require fresh evidence, but they do not reopen package removal
as the selected remediation.

The retained R12 evidence confirms that this is not merely an assumption. Two
Docker-authored DHI OpenVEX documents, last updated on 2026-07-28, contain
`not_affected` statements for the Debian source package
`perl@5.40.1-6+dhi6`. The Docker-signed products do not directly name all four
installed binary packages reported by Trivy.

The R12 compatibility projection mapped the Docker source-package decision to
the installed `libperl5.40`, `perl`, `perl-base`, and
`perl-modules-5.40` binary package PURLs. That mapping covered all eight
Critical/High findings per package present in the R12 raw Trivy report,
including the four CVEs discussed above. A direct package-name join matches
only the eight findings reported against the `perl` binary; the other 24
require the evidenced Debian source-to-binary relationship.

The Trivy-compatible R12 document and its binary-package aliases are a
Tyk-authored mechanical translation of those verified Docker source-package
statements. Its bytes are not Docker-signed and it must not be presented as a
new Tyk exploitability assessment or as if Docker signed the translated
document. The original Docker statements, subjects, signatures, authenticated
source-to-binary mapping, and translation record remain the decision evidence.
The open product issue is making stock Trivy discover and match that evidence
faithfully after a same-basename private-registry mirror, while retaining a
current source for later Docker decision changes.

## EXP-014 Live Replay Boundary

The corrected builder subsequently consumed Docker's actual signed input
shapes for the same immutable amd64 digest: a `pkg:docker` CycloneDX root,
source-to-binary dependency edges, the exact tag-based OpenVEX parent, and the
direct public advisory checkout. Its `42` tests passed, but the live run
correctly created no repository.

EXP-015 subsequently added fail-closed controls for unsupported image product
structure, non-generic Debian index keys, nested public products, and
cross-primary vulnerability aliases. The bounded suite now passes `46` tests;
EXP-016 verifies the stricter Debian image scope against all three exact signed
VEX artifacts while retaining Docker's unrelated public-index ecosystems. The
EXP-014 live result remains unchanged.

The first conflict was `CVE-2017-13716` for `binutils@2.44-3+dhi6`:

```text
image-specific VEX  2026-05-10T11:35:31Z      not_affected
public VEX          2026-05-21T19:59:41.074Z  under_investigation
```

The later public statement includes unversioned
`pkg:deb/debian/binutils`, so it overlaps the installed DHI package. Docker's
image document `last_updated` value does not provide a safe OpenVEX
statement-precedence rule. Until Docker publishes authenticated
specificity/currentness semantics or corrected statements, the translator
must fail closed rather than emit the older suppressing decision.

The exact immutable evidence is recorded in `goal.md` EXP-014 and in the
[Docker issue follow-up](https://github.com/docker-hardened-images/advisories/issues/1827#issuecomment-5118790554).

## Cleanup

`/private/tmp/exp002-vex.v2h9E8am` and all `exp002-vex.*` artifacts were
removed. No repository files were edited during the classification.
