# Docker Bug Report: Public DHI VEX Omits Trivy Binary Aliases

## Proposed title

Public DHI VEX omits Debian binary aliases present in Scout registry exports

## Summary

The signed public GitHub VEX for the immutable BusyBox DHI image omits Debian
binary-package aliases that are present in the authenticated
`docker scout vex get` export for the same product.

With Trivy 0.72.0:

- no VEX reports 4 Critical and 18 High records;
- the public signed BusyBox VEX leaves 6 High records active;
- the authenticated Scout registry VEX suppresses all 22 records.

This reproduction scans the DHI image directly. It does not apply VEX to a
derived image, merge products, translate PURLs, or modify either VEX document.

## Immutable inputs

| Input | Value |
| --- | --- |
| BusyBox DHI index | `dhi.io/busybox:1-debian-fips-dev@sha256:d383017cc5b4984c4088845242dbb70c075102aae43c16c45038d5750c411f42` |
| `linux/amd64` subject | `sha256:4086ad19166e954afdff09ffa27a9fa1daa8c57ca747d4ce76bd67bc93626621` |
| Scout VEX artifact | `sha256:f7d210856281e20551d0ab79815aae46d8338701229253ed06e62520fa780b88` |
| Scout VEX SHA-256 | `e69b1d472b83314fbc557096e150a10bf1523ad51f7101c3ca23f48b504764d3` |
| Public advisory commit | `01b0f4f4f4c247221005d0c8b96e381af34ab291` |
| Public BusyBox VEX SHA-256 | `fe0376035e64e2d0772913bc35ec665ab3ada42de498e6f8a5624bd2c1fa6c9d` |
| Trivy | `0.72.0` |
| Trivy DB updated | `2026-07-27T07:48:00.123872744Z` |

## Result

| Input | Active Critical | Active High | Suppressed |
| --- | ---: | ---: | ---: |
| No VEX | 4 | 18 | 0 |
| Public signed product VEX | 0 | 6 | 16 |
| Authenticated Scout export | 0 | 0 | 22 |

The six residual public-feed records are:

| Binary package | Residual |
| --- | ---: |
| `pkg:deb/debian/busybox` | 5 High |
| `pkg:deb/debian/gpgv` | 1 High |

The public VEX has exact source-package decisions, while the Scout export also
contains scanner-matchable binary aliases. Trivy does not infer the missing
source-to-binary relationship when it consumes the public document directly.

## Reproduction

Docker login is required only for retrieving the DHI image and registry VEX in
this comparison.

```bash
set -euo pipefail

IMAGE='dhi.io/busybox:1-debian-fips-dev@sha256:d383017cc5b4984c4088845242dbb70c075102aae43c16c45038d5750c411f42'
COMMIT='01b0f4f4f4c247221005d0c8b96e381af34ab291'

docker scout vex get \
  --platform linux/amd64 \
  --verify \
  --skip-tlog \
  "registry://${IMAGE}" \
  --output /tmp/dhi-busybox-registry.vex.json

curl -fsSLo /tmp/dhi-busybox-public.vex.json \
  "https://raw.githubusercontent.com/docker-hardened-images/advisories/${COMMIT}/vex/busybox/dhi-busybox.vex.json"
curl -fsSLo /tmp/dhi-busybox-public.vex.json.sig \
  "https://raw.githubusercontent.com/docker-hardened-images/advisories/${COMMIT}/vex/busybox/dhi-busybox.vex.json.sig"
curl -fsSLo /tmp/dhi-public-key.pem \
  'https://registry.scout.docker.com/keyring/dhi/latest'

printf '%s  %s\n' \
  '1d02bbccf149283ae6288d96264dcad3fb23ee1911d90324a48eab28e4cb8a5f' \
  '/tmp/dhi-public-key.pem' | sha256sum -c -

cosign verify-blob \
  --bundle /tmp/dhi-busybox-public.vex.json.sig \
  --key /tmp/dhi-public-key.pem \
  /tmp/dhi-busybox-public.vex.json

common=(
  image
  --image-src remote
  --platform linux/amd64
  --scanners vuln
  --severity HIGH,CRITICAL
  --show-suppressed
  --format json
  --no-progress
  --timeout 45m
)

trivy "${common[@]}" \
  --output /tmp/no-vex.json \
  "$IMAGE"

trivy "${common[@]}" \
  --vex /tmp/dhi-busybox-public.vex.json \
  --output /tmp/public-vex.json \
  "$IMAGE"

trivy "${common[@]}" \
  --vex /tmp/dhi-busybox-registry.vex.json \
  --output /tmp/registry-vex.json \
  "$IMAGE"
```

Count the results:

```bash
for report in /tmp/no-vex.json /tmp/public-vex.json /tmp/registry-vex.json; do
  jq '{
    active_critical: ([.Results[].Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length),
    active_high: ([.Results[].Vulnerabilities[]? | select(.Severity == "HIGH")] | length),
    suppressed: ([.Results[].ExperimentalModifiedFindings[]?] | length)
  }' "$report"
done
```

## Anonymous public-access issue

The GitHub VEX and signatures are anonymously downloadable, but the captured
registry predicate with the additional observed aliases is not. With Docker
Scout 1.23.1 and an empty Docker configuration, registry retrieval returns:

```text
GET https://dhi.io/token?scope=repository%3Abusybox%3Apull&service=registry.docker.io:
unexpected status code 401 Unauthorized
```

This prevents a credential-free scanner from using the document that already
contains the required aliases.

## Expected behavior

The public VEX publication pipeline should include the scanner-matchable
Debian binary aliases present in the captured signed registry export, or an
authenticated complete current predicate should be anonymously retrievable.

Docker remains the decision authority. Customers need an anonymously
retrievable, machine-consumable view of Docker's complete current applicable
evidence, whether that is the public GitHub feed, signed image-specific
artifacts, or a documented combination. Neither transport alone should be
called permanently complete. The requested fix is not a static VEX mirror in
downstream images.
