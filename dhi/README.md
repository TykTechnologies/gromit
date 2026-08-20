# Docker Hardened Image Customizations

Gromit owns the Docker-managed base used by the next-generation Tyk plugin
compiler. The customization installs the compiler's direct package requirements
on the non-dev Debian 13 FIPS BusyBox image. Docker resolves the transitive
closure and publishes the customized image with its own signed SBOM,
provenance, and VEX attestations.

The Docker token remains in the local or CI Docker/DHI credential store. It
must not be added to this repository.

## FIPS scopes

The customized base and the generated plugin use separate cryptographic
modules:

- the customized DHI contains Docker's OpenSSL FIPS Provider 3.1.2, which is
  active at runtime;
- the current Tyk FIPS plugin compiler sets `GOFIPS140=v1.0.0`, linking the
  native Go Cryptographic Module covered by CMVP Certificate #5247 into the
  plugin; it does not use legacy Go+BoringCrypto;
- older release branches can still select Go+BoringCrypto through their
  Gromit branch variables.

Docker has not published its FIPS predicate for the exact customized image
subject. That is an evidence gap for claiming that the customized compiler
container is Docker-FIPS-attested. It does not mean the generated Go plugin is
non-FIPS, and it does not block an alpha unless that exact container claim is
an alpha requirement.

## Apply

```bash
./dhi/apply-plugin-compiler-customization.sh
```

The wrapper creates the customization when it does not exist and edits the
existing customization only when its normalized manifest differs. Override
the organization with `DHI_ORG`; `DHI_DESTINATION` defaults to
`${DHI_ORG}/dhi-busybox-plugin-compiler`.

Current remote customization:

```text
organization: tykio
destination:  tykio/dhi-busybox-plugin-compiler
name:         plugin compiler ng toolchain
id:           cz_3gxyrz8ca7x0l
tag:          1.37.0-debian13-fips_plugin-compiler-ng-toolchain
index:        sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733
amd64:        sha256:58369d0f3051eaf1c7478465ddd2c36aa582c437f6d66fbc933e25de5a7dc0df
arm64:        sha256:603c7940ecc930f763cd869e98285c1952e40885a2bf549ce7aadf9325a3381f
```

## Monitor

```bash
docker dhi customization build list cz_3gxyrz8ca7x0l --org tykio
docker dhi customization build get cz_3gxyrz8ca7x0l <build-id> --org tykio
docker dhi customization build logs cz_3gxyrz8ca7x0l <build-id> --org tykio
```

Do not update Gromit's plugin compiler source-base reference until both amd64
and arm64 customization builds succeed.

## Verify

Resolve the rolling customization tag on every promotion and pin the resulting
multi-platform index digest in `config/config.yaml`.

```bash
REPO=tykio/dhi-busybox-plugin-compiler
TAG=1.37.0-debian13-fips_plugin-compiler-ng-toolchain
INDEX=sha256:8a8967f03d2243d88659256e8a3ca3f5a7b009a4b522e5608f1facfed9be3733
REF="${REPO}:${TAG}@${INDEX}"
REMOTE="registry://${REPO}@${INDEX}"

docker buildx imagetools inspect "$REF"
docker scout attestation list --platform linux/amd64 "$REMOTE"
docker scout cves --platform linux/amd64 --only-severity critical,high "$REMOTE"

trivy image \
  --image-src remote \
  --platform linux/amd64 \
  --scanners vuln \
  --severity HIGH,CRITICAL \
  "${REPO}@${INDEX}"

grype \
  "registry:${REPO}@${INDEX}" \
  --platform linux/amd64 \
  --fail-on high
```

Do not add `docker scout vex get` to the customer gate yet. Docker Scout
`1.20.4` and `1.23.1` both verify the first DHI signature and then panic in
`VEXExportProcessor`, leaving no VEX output. The three underlying OpenVEX OCI
artifacts are present and independently verify against Docker's DHI key, but
passing those raw in-toto predicates directly to Trivy does not reproduce
Scout's required package-product projection.

Package inclusion makes findings eligible for Docker's customized-image
assessment; it does not imply that every finding is `not_affected`. Verify that
the SBOM contains the complete Git and compiler dependency closures and review
every VEX disposition before promotion.

The Go toolchain, Gateway source, and compatibility glibc sysroots are added by
the downstream compiler build. They are outside this Debian package boundary
and require their own SBOM and vulnerability checks.

## Verification status

The final 2026-07-28 build passed these gates:

- amd64 and arm64 customization builds completed successfully;
- 207 packages are present in Docker's signed SBOM;
- native and cross C/C++ compilers for amd64, arm64, and s390x execute;
- the OpenSSL FIPS provider is active;
- Trivy 0.72 reports 17 Critical and 75 High without VEX;
- the downstream FIPS compiler builds and loads an amd64 native-Go-FIPS plugin
  with `GOFIPS140=v1.0.0`, validates an arm64 plugin, and excludes the
  test-only Gateway binary from the production image.

The following are open upstream gates and must not be represented as passing:

- the custom image has the `com.docker.dhi.compliance=fips,stig,cis` label and
  an active OpenSSL FIPS provider, but lacks Docker's signed
  `https://docker.com/dhi/fips/v0.1` attestation for the customized subject;
  this limits the compiler-container claim, not the generated plugin's native
  Go FIPS mode;
- anonymous Scout VEX retrieval for the custom repository returns
  `UNAUTHORIZED`, so customers currently need Docker registry access to fetch
  the live custom-product document;
- authenticated `docker scout vex get` panics in both Scout `1.20.4` and
  `1.23.1`, so there is currently no supported customer command that exports
  the live custom-product VEX for Trivy;
- Grype 0.116 reports four active High libssh2 issues
  (`CVE-2026-66032` through `CVE-2026-66035`; eight matches). Debian currently
  marks them vulnerable, Docker VEX has no disposition for them, and
  `1.11.1-1+dhi3` is the newest DHI package. They must be fixed upstream or
  explicitly assessed; they cannot be legitimately suppressed.
