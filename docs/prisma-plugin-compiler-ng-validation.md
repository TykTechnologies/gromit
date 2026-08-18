# Prisma Cloud Validation for Plugin Compiler NG

Last updated: 2026-07-29

## Purpose

This runbook is the customer-environment Prisma Cloud acceptance test for a
plugin compiler NG release. It does not infer a Prisma result from Trivy,
Docker Scout, Grype, Docker OpenVEX, Tyk OpenVEX, or Gromit evidence.

No supported contract has been established for Prisma Cloud to ingest an
arbitrary external OpenVEX repository or OCI referrer for this workflow. Do
not promise VEX ingestion or a zero-active Prisma result.

## Historical Evidence Boundary

The superseded Gromit R12 experiment produced local engineering evidence for:

```text
localhost:5001/plugin-compiler-fips-candidate@
sha256:c04b2abcb35d5c3e176f6b451f58e3f922da3d6da7e9d9831306d3b552341188
```

Stock Trivy reported raw 17 Critical / 75 High and active 0 Critical / 0 High
after an exact Tyk-authored mechanical projection derived from Docker-signed
DHI inputs. Those counts are not Prisma counts, and that custom
scanner/projection pipeline is not the selected implementation. R12 contains
no Prisma report and is not provenance for a future customer registry digest.

The Docker-signed image VEX and SBOM snapshot accounts for the inherited DHI
package findings described in `goal.md` EXP-010. EXP-014 subsequently found a
chronology conflict with the current public feed and emitted no repository.
Neither result proves what Prisma discovers, classifies, or filters.

Prisma acceptance begins only after the final customer image exists.

## Prisma Policy Boundary

Prisma's raw result is its own package/file/hash inventory correlated with the
customer's Console and Intelligence Stream. Preserve it unchanged.

A separately reviewed policy view may use supported Prisma controls:

- scan the exact DHI base;
- identify it as the base for the exact compiler descendant;
- apply a narrowly scoped `Exclude base image vulnerabilities` rule;
- use narrow, approved, expiring exceptions for residual findings;
- retain the raw and filtered reports independently.

Docker SLSA, signed SBOM, signed DHI VEX, and immutable Tyk build provenance
can corroborate a review. They do not prove that Prisma consumed VEX or that
Prisma filtered the same records.

## Required Identities

Record these values before scanning:

| Object             | Required value                                        |
| ------------------ | ----------------------------------------------------- |
| Compiler release   | Customer repository/tag and immutable index digest    |
| Compiler platform  | Exact platform-manifest digest                        |
| Local scan subject | Resolved Docker image ID                              |
| DHI base           | Run-resolved index and selected platform digest       |
| Build              | Source commit and workflow run                        |
| Promotion          | Destination and exact promoted digest                 |
| Prisma             | `twistcli`, Console, and Intelligence Stream versions |

The build workflow resolves the moving DHI tag once per run. Use the index and
platform digest recorded by that workflow; do not copy the R12 digest into a
future release policy unless it is the value that run actually resolved.

The R12 DHI values, for historical comparison only, were:

```text
index: sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733
amd64: sha256:58369d0f3051eaf1c7478465ddd2c36aa582c437f6d66fbc933e25de5a7dc0df
arm64: sha256:603c7940ecc930f763cd869e98285c1952e40885a2bf549ce7aadf9325a3381f
```

## Prepare the Immutable Local Subject

Use the `twistcli` binary supplied for the customer's Console version:

```bash
set -euo pipefail

IMAGE='registry.example.com/tyk-plugin-compiler-fips@sha256:<release-index>'
PLATFORM='linux/amd64'
RUN_ROOT="$PWD/prisma-plugin-compiler-fips"
RAW_DIR="$RUN_ROOT/raw"

test ! -e "$RUN_ROOT"
mkdir -p "$RAW_DIR"

docker pull --platform "$PLATFORM" "$IMAGE" \
  >"$RAW_DIR/docker-pull.stdout" \
  2>"$RAW_DIR/docker-pull.stderr"

LOCAL_IMAGE_ID="$(docker image inspect --format '{{.Id}}' "$IMAGE")"
case "$LOCAL_IMAGE_ID" in
  sha256:*) ;;
  *)
    printf 'Unexpected local image ID: %s\n' "$LOCAL_IMAGE_ID" >&2
    exit 2
    ;;
esac

printf '%s\n' "$IMAGE" >"$RAW_DIR/requested-image.txt"
printf '%s\n' "$PLATFORM" >"$RAW_DIR/platform.txt"
printf '%s\n' "$LOCAL_IMAGE_ID" >"$RAW_DIR/local-image-id.txt"
docker image inspect "$IMAGE" >"$RAW_DIR/docker-image-inspect.json"
docker version >"$RAW_DIR/docker-version.txt"
./twistcli --version >"$RAW_DIR/twistcli-version.stdout" \
  2>"$RAW_DIR/twistcli-version.stderr"
```

If `IMAGE` is a multi-platform index, compare the selected platform manifest
in registry evidence with `docker-image-inspect.json`. The scan target below
is the resolved local image ID, not a tag or Docker-local alias.

## Capture Raw Policy State

Before the raw scan, export or record:

- vulnerability rule ID, name, scope, order, effect, and revision time;
- current base-image configuration;
- every applicable CVE exception, package scope, image scope, owner, approval,
  and expiry;
- Console and Intelligence Stream versions;
- project, tenant, collection, and credential context.

Prefer machine-readable exports. If only UI state exists, retain screenshots
plus a text ledger with stable rule and exception IDs. Store this as
`raw-rule-state` under the raw directory.

Do not add a new base-image rule or exception before the raw scan.

## Raw Scan

Configure credentials according to the customer's `twistcli` distribution.
The common environment form avoids putting secrets in the command line:

```bash
export PCC_CONSOLE='https://<customer-console>'
export TWISTLOCK_USER='<customer-user-or-access-key-id>'
export TWISTLOCK_PASSWORD='<customer-password-or-secret>'
```

Run:

```bash
set -euo pipefail

RAW_DIR="$PWD/prisma-plugin-compiler-fips/raw"
LOCAL_IMAGE_ID="$(cat "$RAW_DIR/local-image-id.txt")"

set +e
./twistcli images scan \
  --address "$PCC_CONSOLE" \
  --details \
  --output-file "$RAW_DIR/report.json" \
  "$LOCAL_IMAGE_ID" \
  >"$RAW_DIR/scan.stdout" \
  2>"$RAW_DIR/scan.stderr"
scan_exit=$?
set -e

printf '%s\n' "$scan_exit" >"$RAW_DIR/scan.exit-code.txt"
test -s "$RAW_DIR/report.json"
sha256sum "$RAW_DIR/report.json" >"$RAW_DIR/report.json.sha256"
```

Retain the complete JSON even when policy fails. Record the scan/job ID,
start/end timestamps, Prisma-reported image digest, rule revision, and policy
result.

No accepted command uses a fixability filter or global vulnerability ignore.

## Exit-Code Interpretation

Treat exit code `1` as ambiguous. Depending on Console and CLI version, it can
represent a policy result, authentication problem, Console error, malformed
response, or another operational condition.

Classify the run only after reviewing:

- `report.json`;
- `scan.stdout` and `scan.stderr`;
- Prisma scan/job ID and Console state;
- reported image ID/digest;
- exact rule state;
- `twistcli`, Console, and Intelligence Stream versions.

If JSON is missing, incomplete, or not tied to the resolved local image ID,
the scan failed operationally regardless of exit code. Never convert an exit
code alone into a vulnerability count.

## Configure the Exact DHI Base

Use the DHI index and platform digest captured by the same release workflow:

```text
tykio/dhi-busybox-plugin-compiler@sha256:<resolved-platform-manifest>
```

If Console requires a tag for discovery, use the run's captured
`moving-tag@resolved-index` reference, then retain Prisma's selected platform
digest and prove it equals the workflow value.

Scan that exact DHI base in Prisma before enabling base filtering. Scope
`Exclude base image vulnerabilities` only to the intended plugin compiler NG
repositories, platforms, and release candidates. Capture the result as
`filtered-rule-state`.

Base filtering is acceptable only when:

1. Prisma scanned the exact DHI platform;
2. Prisma identifies the exact compiler image as its descendant;
3. every removed record belongs to inherited base content;
4. child-added or child-modified findings remain visible;
5. the scope does not affect unrelated repositories.

Docker's signed base SBOM and the final Tyk image's SBOM and provenance can
support the ancestry comparison, but Prisma evidence must show what Prisma
actually filtered.

## Filtered Scan

Use a new directory. Do not overwrite the raw report:

```bash
set -euo pipefail

RUN_ROOT="$PWD/prisma-plugin-compiler-fips"
RAW_DIR="$RUN_ROOT/raw"
FILTERED_DIR="$RUN_ROOT/filtered"
LOCAL_IMAGE_ID="$(cat "$RAW_DIR/local-image-id.txt")"

test ! -e "$FILTERED_DIR"
mkdir -p "$FILTERED_DIR"

printf '%s\n' "$LOCAL_IMAGE_ID" >"$FILTERED_DIR/local-image-id.txt"
./twistcli --version >"$FILTERED_DIR/twistcli-version.stdout" \
  2>"$FILTERED_DIR/twistcli-version.stderr"

set +e
./twistcli images scan \
  --address "$PCC_CONSOLE" \
  --details \
  --output-file "$FILTERED_DIR/report.json" \
  "$LOCAL_IMAGE_ID" \
  >"$FILTERED_DIR/scan.stdout" \
  2>"$FILTERED_DIR/scan.stderr"
scan_exit=$?
set -e

printf '%s\n' "$scan_exit" >"$FILTERED_DIR/scan.exit-code.txt"
test -s "$FILTERED_DIR/report.json"
sha256sum "$FILTERED_DIR/report.json" \
  >"$FILTERED_DIR/report.json.sha256"
```

Retain a new scan ID and the complete filtered rule state. Compare records by
CVE, package, installed version, file, layer, and image digest. A count-only
comparison is insufficient.

## Record-Level Comparison

For every record removed by the base-image rule:

1. identify the raw report record;
2. identify the exact DHI base record;
3. confirm package, installed version, file/layer, and platform match;
4. confirm Gromit ancestry evidence classifies it as inherited;
5. record the Prisma rule responsible for removal.

Any record that cannot be reconciled stays active. Do not infer removal from
Trivy's active 0/0 result or from a VEX status.

## Residual Exceptions

For every Critical or High finding remaining after base filtering:

1. confirm it belongs to the exact local image ID;
2. identify package/file and introducing layer;
3. determine whether a fixed dependency or rebuild removes it;
4. review signed Docker evidence only for the exact matching product,
   package, version, and platform;
5. obtain Product Security approval;
6. scope any exception to the exact repository, descendant set, package, and
   CVE;
7. assign an owner and expiry;
8. retain approval, rule IDs, exception IDs, and both scan IDs.

Do not create a global CVE allow list for an image-specific base decision. Do
not translate `under_investigation`, `affected`, `fix_deferred`, or
`will_not_fix` into `not_affected`.

Grype 0.116.1 found four High `libssh2-1t64` CVEs absent from the captured
signed Docker DHI VEX:

```text
CVE-2026-66032
CVE-2026-66033
CVE-2026-66034
CVE-2026-66035
```

This does not predict Prisma's inventory. If Prisma reports any of them, keep
them active unless the package is fixed or Product Security approves a narrow
exception based on current applicable evidence.

## Evidence Checklist

Retain:

- requested immutable remote image;
- selected platform manifest and local Docker image ID;
- Docker inspect JSON and repo digests;
- run-resolved DHI tag, index, and platform;
- source commit, workflow run, and promotion destination;
- `twistcli --version` stdout/stderr;
- Console and Intelligence Stream versions;
- complete raw and filtered JSON plus SHA-256;
- complete stdout, stderr, and process exit code;
- both Prisma scan/job IDs and timestamps;
- Prisma-reported image identities;
- exact raw and filtered rule/base/exception state;
- record-level raw-versus-filtered comparison;
- Product Security approvals and exception expiry.

Do not publish credentials or tokens in the evidence bundle.

## Acceptance Criteria

The Prisma gate passes only when:

- the local image ID corresponds to the final customer-published immutable
  platform;
- raw and filtered scans are separate, complete, and reproducible;
- exit status is interpreted with JSON, diagnostics, scan ID, and Console
  state;
- Prisma recognizes the exact run-resolved DHI platform;
- every removed record is proven inherited;
- no child-layer finding is hidden as a base finding;
- every residual exception is narrow, approved, owned, and expiring;
- no unexplained Critical or High remains under the approved customer policy;
- the matching plugin build/load and separate FIPS gates pass.

An attached VEX document, a Docker label, a Trivy 0/0 active result, or a
Grype result does not establish that this Prisma gate passed.

## Current and Planned State

No Prisma customer scan has been run for the final release image. R12,
EXP-010, and EXP-014 are Trivy/DHI engineering evidence, not Prisma evidence.
The customer still needs to provide Console access, run the immutable release
scan, and retain policy state. Public Tyk VEX hosting and GitHub automation are
separate future dependencies and do not change this Prisma acceptance path.
