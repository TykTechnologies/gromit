# DHI VEX-Aware Trivy Scanning

The `gromit dhi-vex` command applies Docker's current signed DHI VEX to an
immutable Tyk image without bundling a point-in-time VEX file in that image.
It is intended for derived DHI images such as the Tyk Gateway and
next-generation plugin compiler.

## Requirements

- Trivy 0.72.0 or later
- Cosign 3.0.6 or later
- Anonymous HTTPS access to Docker Hub, GitHub, and
  `registry.scout.docker.com`
- An immutable `repository@sha256:...` image reference

`GITHUB_TOKEN` is optional and is used only to increase the GitHub API rate
limit. No Docker Scout API key or DHI registry credential is required.

## Customer command

Pin Gromit to the reviewed commit rather than a moving branch:

```bash
IMAGE='tykio/tyk-plugin-compiler-fips@sha256:e2f9b5e5c867026e6e921d65c02c30595c3477f97651e4b766e36d58a0819cd6'
GROMIT_COMMIT='<reviewed-gromit-commit>'
OUT="$PWD/trivy-dhi-vex-$(date -u +%Y%m%dT%H%M%SZ)"

go run "github.com/TykTechnologies/gromit@${GROMIT_COMMIT}" dhi-vex \
  --platform linux/amd64 \
  --severity HIGH,CRITICAL \
  --output-dir "$OUT" \
  "$IMAGE"
```

The command returns zero only when no selected-severity finding remains
active. Any scanner, network, signature, lineage, database, projection, or
accounting failure returns non-zero and does not publish a partial evidence
directory.

## Security model

The command performs these checks on every invocation:

1. Rejects tags and scans the requested digest directly from the registry.
2. Uses private `HOME`, `XDG_*`, Docker configuration, Trivy configuration,
   ignore file, and cache directories. Inherited Trivy policy variables cannot
   enable `ignore-unfixed` or change the severity gate.
3. Downloads a fresh Trivy database, records its metadata and file hashes, and
   prevents the final scan from updating it.
4. Verifies the requested OCI index digest in Trivy's repository metadata and
   requires the same selected platform image ID in the raw and final reports.
5. Recomputes `com.docker.dhi.chain-id` from layer DiffIDs. Only SBOM
   components attributed to a layer inside that DHI base boundary are eligible
   for Docker VEX.
6. Derives the VEX product from the image's DHI labels. It does not combine
   decisions from unrelated Docker products.
7. Resolves Docker's public advisory branch once to an immutable commit,
   downloads that product's VEX and signature from the commit, verifies the
   signature with Cosign, and checks Docker's public-key SHA-256 against the
   fingerprint pinned in Gromit.
8. Requires the same CVE and exact Debian source name, epoch, version, release,
   and `+dhiN` suffix in a current `not_affected` Docker statement.
9. Generates a run-local OpenVEX compatibility document containing only the
   exact Trivy binary PURL. The source assertion remains in the audit ledger
   but cannot cause broader scanner suppression.
10. Requires exact multiset accounting: raw findings must equal active plus
    suppressed findings, suppressed findings must equal the projection ledger,
    and active findings must equal the unmatched ledger.

No compatibility VEX is attached to or published with the Tyk image. The
run-local document is derived from the current verified Docker feed and is
discardable evidence for that scan.

## Evidence

The output directory contains:

- `trivy-raw.json`: complete selected-severity Trivy findings
- `trivy-sbom.cdx.json`: package and layer evidence used for source mapping
- `dhi-<product>.vex.json`, `.sig`, and `.cosign-verify.txt`: exact Docker
  inputs and retained signature-verification output
- `dhi-public-key.pem`: Docker verification key used for the run
- `dhi-trivy-compat.vex.json`: exact binary-PURL projection for this run
- `trivy-vex.json`: active and explicitly suppressed findings
- `trivy-version.json`: scanner and vulnerability database metadata
- `projection-manifest.json`: hashes, lineage, source statements, projections,
  and unmatched reasons
- `summary.json`: gate counts and immutable identities

## Current alpha results

The values below use Trivy 0.72.0, Docker advisory commit
`01b0f4f4f4c247221005d0c8b96e381af34ab291`, and the alpha8 image digests.
They are observations, not permanent allowlists.

| Image | Raw | Docker VEX suppressed | Active |
| --- | ---: | ---: | ---: |
| Plugin compiler FIPS NG | 17 Critical / 82 High | 4 Critical / 18 High | 13 Critical / 64 High |
| Gateway FIPS | 0 Critical / 6 High | 0 Critical / 5 High | 0 Critical / 1 High |

The compiler cannot legitimately report zero from Docker VEX alone. Its DHI
BusyBox base accounts for 22 severe records, but 77 records belong to packages
installed by Tyk in child layers. Docker's current-production integration
rules prohibit applying DHI base VEX to those packages. Applying Golang DHI
decisions merely because source versions match would be a cross-product false
suppression.

The Gateway's active High is `GHSA-hrxh-6v49-42gf` in
`google.golang.org/grpc v1.80.0`; it is an application dependency and remains
outside Docker's Debian-package VEX.

For a customer policy requiring raw zero High/Critical, the remaining compiler
records need one of:

- package upgrades or an architecture that removes the affected package;
- a documented customer risk exception;
- a Tyk-authored, Product Security-approved VEX assessment for the exact Tyk
  image and finding.

`--ignore-unfixed`, cross-product Docker VEX, and removing package metadata are
not equivalent controls and must not be used to manufacture zero.

## Key rotation

Docker's DHI signing key fingerprint is pinned in the command. A legitimate
Docker key rotation will fail closed until the fingerprint change is reviewed,
tested, and released in Gromit. It does not require rebuilding Tyk images.
