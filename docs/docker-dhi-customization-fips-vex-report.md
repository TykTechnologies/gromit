# Docker DHI Customization Report: FIPS Evidence, VEX Export, and libssh2

## Proposed title

Customized FIPS DHI omits FIPS attestation, cannot export live VEX, and
currently installs vulnerable libssh2

## Summary

Docker successfully built a multi-platform customization from the non-dev
Debian 13 BusyBox FIPS DHI. The resulting image has a signed SBOM, provenance,
STIG evidence, and three signed OpenVEX attestations. Runtime OpenSSL loads the
FIPS provider, and the image carries Docker's `fips,stig,cis` compliance label.

Four independent issues prevent the customized image from providing the same
customer-verifiable security posture as its source FIPS DHI:

1. The source image has a signed `https://docker.com/dhi/fips/v0.1`
   attestation, but the customization does not.
2. The custom-product VEX artifacts require registry authentication;
   anonymous retrieval returns `UNAUTHORIZED`.
3. Docker Scout `vex get` panics after signature verification in both tested
   CLI versions and leaves no usable VEX file.
4. The newest DHI `libssh2-1t64` package is affected by four newly published
   High CVEs and the custom-product VEX does not disposition them.

No downstream VEX or scanner ignore is used in this reproduction.

## Immutable inputs

| Input | Value |
| --- | --- |
| Customization | `cz_3gxyrz8ca7x0l` |
| Repository | `tykio/dhi-busybox-plugin-compiler` |
| Tag | `1.37.0-debian13-fips_plugin-compiler-ng-toolchain` |
| Index | `sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733` |
| amd64 subject | `sha256:58369d0f3051eaf1c7478465ddd2c36aa582c437f6d66fbc933e25de5a7dc0df` |
| arm64 subject | `sha256:603c7940ecc930f763cd869e98285c1952e40885a2bf549ce7aadf9325a3381f` |
| Source FIPS DHI | `tykio/dhi-busybox:1.37-fips` |
| Docker Scout | `1.20.4` and `1.23.1` |
| Trivy | `0.72.0` |
| Grype | `0.116.0` |
| Test date | `2026-07-28` |

## Issue 1: missing signed FIPS evidence

The source image exposes Docker's FIPS predicate:

```bash
docker scout attestation list \
  --platform linux/amd64 \
  tykio/dhi-busybox:1.37-fips
```

Expected and observed on the source:

```text
https://docker.com/dhi/fips/v0.1  FIPS compliance
```

The same command for the customized image lists CycloneDX, SPDX, Scout SBOM,
SLSA provenance, STIG, vulnerability, malware, secret, and OpenVEX
attestations, but no FIPS predicate:

```bash
docker scout attestation list \
  --platform linux/amd64 \
  'registry://tykio/dhi-busybox-plugin-compiler@sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733'
```

Runtime evidence is positive but is not a substitute for the missing signed
predicate:

```text
com.docker.dhi.compliance=fips,stig,cis
OpenSSL 3.5.6
OpenSSL FIPS Provider 3.1.2: active
```

Requested behavior: either generate a FIPS predicate that accurately explains
the customized image's compliance boundary, or make the customization result
explicitly fail/non-FIPS instead of retaining a FIPS compliance label without
the corresponding signed evidence.

## Issue 2: live VEX is not anonymously retrievable

The custom repository and its Scout attestation registry return
`UNAUTHORIZED` without Docker credentials. This prevents customers scanning a
public downstream Tyk image from fetching the current custom-base VEX unless
they also have suitable Docker registry access.

Attaching a point-in-time VEX to every downstream image is not a suitable
replacement because Docker's assessments change independently.

Requested behavior: make the signed VEX predicate for a downstream-public DHI
lineage anonymously retrievable, or provide a public live advisory endpoint
for organization customizations.

## Issue 3: Docker Scout panics while exporting valid VEX artifacts

Authenticated attestation discovery finds three OpenVEX artifacts. Their OCI
layer digests match their content, and Cosign verifies all three against
Docker's DHI public key. Together they contain 1,595 statements: 1,553
`not_affected` and 42 `under_investigation`.

The supported export command fails in both Scout `1.20.4` and `1.23.1`:

```bash
docker scout vex get \
  --platform linux/amd64 \
  --verify \
  --skip-tlog \
  --output /tmp/custom.vex.json \
  'registry://tykio/dhi-busybox-plugin-compiler@sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733'
```

Observed after the first signature is verified:

```text
panic: runtime error: index out of range [0] with length 0
github.com/docker/scout-cli-plugin/internal/attestations.(*VEXExportProcessor).Process
```

The command exits `2` and creates no usable output. Passing the raw in-toto
predicates directly to Trivy is not equivalent because Scout normally projects
Docker's package identifiers into the scanner product identifiers during
export.

Requested behavior: make the exporter handle Docker's own custom-image
attestation shape and return a verified OpenVEX document instead of panicking.

## Issue 4: current libssh2 package remains vulnerable

The customization installs Git. Its DHI dependency closure includes:

```text
libssh2-1t64 1.11.1-1+dhi3
```

The current DHI package repository has no newer candidate:

```text
Candidate: 1.11.1-1+dhi3
```

An earlier successfully exported equivalent VEX caused Grype to report zero
Critical and eight High matches. The eight matches represent four CVEs
duplicated across catalog paths:

```text
CVE-2026-66032
CVE-2026-66033
CVE-2026-66034
CVE-2026-66035
```

Debian's security tracker currently marks Trixie vulnerable for all four. The
Docker VEX document contains no statements for these CVEs, so the active result
is correct. Trivy 0.72 and Docker Scout currently report zero because their
databases do not yet contain or expose these findings.

Requested behavior: publish a DHI package containing the upstream fixes and
rebuild affected customizations. Until then, Docker Scout should avoid
presenting the image as having no vulnerable packages if its advisory database
has not ingested these public CVEs.

## Scanner matrix

| Scanner and policy | Critical | High | Interpretation |
| --- | ---: | ---: | --- |
| Trivy, no VEX | 17 | 75 | Raw Debian records |
| Trivy, current signed custom VEX | blocked | blocked | Scout exporter panics; raw predicates do not reproduce Scout's product projection |
| Docker Scout with attached VEX | 0 | 0 | Scout database lacks or does not expose the four new libssh2 CVEs |
| Grype, previously exported equivalent VEX | 0 | 8 | Four currently active libssh2 CVEs |

## Security expectation

The requested resolution is not a blanket VEX statement. A finding should be
suppressed only when Docker has assessed the exact customized product and can
justify `not_affected`. Findings that reach vulnerable code must remain active
until a patched package is published.
