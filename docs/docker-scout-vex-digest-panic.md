# Docker Scout Bug Report: Digest-Only VEX Export Panic

Last updated: 2026-07-29

## Proposed Title

`docker scout vex get` panics for a digest-only custom-DHI reference

## Scope

This report records behavior reproduced on 2026-07-28. It does not claim that
the defect persists in a newer Scout version and does not establish a
source-level root cause.

The accepted Gromit R12 directory contains independently retrieved and
verified DHI evidence plus Trivy reports. It contains no Scout command output,
Scout version record, or Scout-exported VEX file. Do not cite R12 as the Scout
reproduction.

## Observed Versions

| Scout    | Host platform  | Failure location                            |
| -------- | -------------- | ------------------------------------------- |
| `1.20.4` | `darwin/arm64` | `internal/attestations/vex_processor.go:80` |
| `1.23.1` | `linux/amd64`  | `internal/attestations/vex_processor.go:78` |

Retest with the current Scout version before relying on the workaround.

## Immutable Subject

The moving custom-DHI tag was resolved to this immutable R12 snapshot:

| Object               | Identity                                                                              |
| -------------------- | ------------------------------------------------------------------------------------- |
| Moving tag           | `tykio/dhi-busybox-plugin-compiler:1.37.0-debian13-fips_plugin-compiler-ng-toolchain` |
| Image index          | `sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733`             |
| Linux/amd64 image    | `sha256:58369d0f3051eaf1c7478465ddd2c36aa582c437f6d66fbc933e25de5a7dc0df`             |
| Example VEX referrer | `sha256:6143c27c1a559ce719b6b6ff1d09c1ff86ae2ddbe95dbbe6272f7b56b31565a0`             |

Resolve a moving tag once and retain the exact index for each new test. Do not
assume that the tag still points to the R12 digest.

Use Docker credentials when the repository or Scout authority requires them.
Retain version, command, stdout, stderr, exit code, and output-file state for
every reproduction.

## Digest-Only Reproduction

```bash
set -euo pipefail

IMAGE='tykio/dhi-busybox-plugin-compiler'
INDEX='sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733'
OUT='/tmp/dhi-vex-digest-only.json'

rm -f "$OUT"
docker scout version

set +e
docker scout vex get \
  --platform linux/amd64 \
  --verify \
  --skip-tlog \
  --output "$OUT" \
  "registry://${IMAGE}@${INDEX}" \
  >'/tmp/dhi-vex-digest-only.stdout' \
  2>'/tmp/dhi-vex-digest-only.stderr'
status=$?
set -e

printf '%s\n' "$status" >'/tmp/dhi-vex-digest-only.exit-code'
ls -l "$OUT" >'/tmp/dhi-vex-digest-only.output-state' 2>&1 || true
```

The dated runs reached key/signature verification and then printed:

```text
panic: runtime error: index out of range [0] with length 0
github.com/docker/scout-cli-plugin/internal/attestations.(*VEXExportProcessor).Process
```

The command exited non-zero and did not leave a usable aggregate output in the
captured runs. A new reproduction must inspect output state rather than assume
that cleanup behavior is unchanged.

The corresponding single-referrer path was also observed to reach the same
processor:

```bash
set -euo pipefail

IMAGE='tykio/dhi-busybox-plugin-compiler'
INDEX='sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733'
VEX='sha256:6143c27c1a559ce719b6b6ff1d09c1ff86ae2ddbe95dbbe6272f7b56b31565a0'

docker scout attestation get \
  --platform linux/amd64 \
  --verify \
  --skip-tlog \
  --predicate \
  --output /tmp/dhi-vex-attestation.json \
  "registry://${IMAGE}@${INDEX}" \
  "$VEX"
```

## Observed Tag-at-Digest Workaround

In the same dated environments, retaining the known tag while constraining the
same immutable index avoided the panic:

```bash
set -euo pipefail

IMAGE='tykio/dhi-busybox-plugin-compiler'
TAG='1.37.0-debian13-fips_plugin-compiler-ng-toolchain'
INDEX='sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733'

docker scout vex get \
  --platform linux/amd64 \
  --verify \
  --skip-tlog \
  --output /tmp/dhi-vex-tag-at-digest.json \
  "registry://${IMAGE}:${TAG}@${INDEX}"
```

The digest still constrains selection. This is an observed client workaround,
not a compatibility guarantee for another Scout version, registry, or
command. It must not replace a product fix.

## Verification Limitations

The reproduction uses `--skip-tlog`. It exercises the configured
key/signature path while explicitly skipping transparency-log verification. A
success message does not prove log inclusion or freshness.

If transparency evidence is required, rerun without `--skip-tlog` or retain an
equivalent verified bundle and log proof. Report a transparency failure
separately from the export panic.

The raw Scout referrer response is also unsigned. Signatures prove the
retrieved artifacts, not completeness or freshness of the list. The historical
Gromit R12 experiment recorded the exact one-provenance/three-VEX descriptor
set with SHA-256:

```text
0deab0a66b918ee2814b91cf0e1f8d8207930842d7850829306d425225dcff80
```

That historical checkpoint neither authorizes the set as current nor proves
that Scout export succeeded.

## Root-Cause Signal

The stack trace names
`github.com/docker/scout-cli-plugin/internal/attestations.VEXExportProcessor`.
Inspection of the shipped 1.20.4 binary found a branch that checks a slice
length and can call `runtime.goPanicIndex` before formatting a `%s:%s`
reference. This is consistent with indexing the first tag of a tagless
reference.

It is not definitive source-level proof. Another parser or state transition
could provide the empty slice. The defect should be described as consistent
with a missing empty-list guard, not as a confirmed root cause.

## Expected Behavior

Both immutable forms should export the same predicate set or return a normal,
actionable error:

```text
registry://repository@sha256:<index>
registry://repository:tag@sha256:<same-index>
```

The CLI must not panic when no tag is present. Display logic can use the digest
or another explicit fallback.

## Regression Coverage

- digest-only, tag-only, and `tag@digest` references;
- single-platform manifests and multi-platform indexes;
- empty and non-empty tag metadata;
- `attestation get` and `vex get`;
- output-file cleanup on failure;
- verification with and without `--skip-tlog`;
- authenticated and anonymous registry access;
- a moving tag resolved once, followed by repeated immutable requests.

## Relationship to Trivy, Grype, Prisma, and FIPS

Scout export behavior is separate from the historical stock-Trivy 0.72
repository experiments. No repository result currently satisfies the customer
north star. Customer Trivy instructions must scan an immutable remote
repository identity; Docker-local aliases can prevent product matching.
Active counts must come from a normal report, not be inferred from
`--show-suppressed`, and no accepted test uses `--ignore-unfixed`.

Scout export does not promise Grype or Prisma VEX ingestion. It also does not
establish FIPS compliance; FIPS requires separate predicate, CMVP, provider,
Go module, runtime, TLS, and plugin-load evidence.
