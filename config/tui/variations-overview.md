# TUI Test Variations Overview

This document summarizes when each cache/db/config combination is used, based on
[`test-variations.yml`](test-variations.yml) and [`prod-variations.yml`](prod-variations.yml).

Structure of the source files: `testsuite (api/ui/lts-version) → branch → trigger (event) → repo → envfiles`.
Each level can override `envfiles` from its parent; if a level has no `envfiles` override, it inherits from the
closest ancestor that defines one (ultimately falling back to the top-level default:
`cache: redis7, config: sha256, db: mongo7`).

## test-variations.yml

| Testsuite | Branch | Event | Repo | Cache | DB | Config | Notes |
|---|---|---|---|---|---|---|---|
| api | default | workflow_dispatch / pull_request / push | any | valkey8 | postgres15 | murmur128 | Default envfile for all branches not explicitly listed; kept on old versions for compatibility with old Tyk versions |
| api | master | pull_request / push (repos other than tyk-pro) | any | valkey9_1 | postgres15 | murmur128 | Branch-level default for `master` |
| api | master | pull_request | tyk-pro | redis8 | mongo7 | sha256 | 1st override matrix entry |
| api | master | pull_request | tyk-pro | valkey8 | postgres16 | murmur128 | 2nd override matrix entry |
| api | master | schedule | tyk-analytics | redis8 | postgres15 | sha256 | 1st scheduled matrix entry (uses master gwdash) |
| api | master | schedule | tyk-analytics | redis8 | mongo8 | murmur128 | 2nd scheduled matrix entry (gwdash release-5.13) |
| api | master | schedule | tyk-analytics | redis8 | postgres14 | murmur128 | 3rd scheduled matrix entry |
| api | master | schedule | tyk-analytics | valkey7 | postgres16 | sha256 | 4th scheduled matrix entry (gwdash release-5.8) |
| api | master | schedule | tyk-analytics | valkey8 | postgres17 | murmur128 | 5th scheduled matrix entry |
| api | master | schedule | tyk-analytics | valkey8 | mongo8 | sha256 | 6th scheduled matrix entry |
| api | master | schedule | tyk-pro | redis8 | postgres18 | sha256 | New entry (gwdash master) |
| ui | default | workflow_dispatch / pull_request / push | tyk-analytics, tyk-pro | redis7 | mongo7 | sha256 | Falls back to global top-level default (no branch/trigger override) |
| ui | master | pull_request | tyk-analytics | redis7 | mongo7 | sha256 | Falls back to global default (no override at this level) |
| ui | master | pull_request | tyk-pro | redis8 | mongo8 | sha256 | 1st override matrix entry (gwdash release-5.12) |
| ui | master | pull_request | tyk-pro | valkey7 | postgres17 | murmur128 | 2nd override matrix entry (gwdash release-5.8) |
| ui | master | schedule | tyk-analytics | valkey8 | mongo7 | murmur128 | 1st scheduled matrix entry (gwdash release-5.12) |
| ui | master | schedule | tyk-analytics | redis7 | mongo8 | murmur128 | 2nd scheduled matrix entry (gwdash release-5.8) |
| ui | master | push | tyk-analytics, tyk-pro | redis7 | mongo7 | sha256 | Falls back to global default (no override) |

## prod-variations.yml

| Testsuite | Branch | Event | Repo | Cache | DB | Config | Notes |
|---|---|---|---|---|---|---|---|
| api | default | workflow_dispatch / pull_request / push | any | valkey8 | postgres15 | murmur128 | Default envfile for all branches not explicitly listed; kept on old versions for compatibility |
| api | master | pull_request / push (repos other than tyk-pro) | any | valkey9 | postgres15 | murmur128 | Branch-level default for `master` (note: `valkey9`, not `valkey9_1` as in test-variations) |
| api | master | pull_request | tyk-pro | redis8 | mongo7 | sha256 | 1st override matrix entry |
| api | master | pull_request | tyk-pro | valkey8 | postgres16 | murmur128 | 2nd override matrix entry |
| api | master | schedule | tyk-analytics | redis8 | postgres15 | sha256 | 1st scheduled matrix entry (uses master gwdash) |
| api | master | schedule | tyk-analytics | redis8 | mongo8 | murmur128 | 2nd scheduled matrix entry (gwdash release-5.13) |
| api | master | schedule | tyk-analytics | redis8 | postgres17 | murmur128 | 3rd scheduled matrix entry |
| api | master | schedule | tyk-analytics | valkey7 | postgres16 | sha256 | 4th scheduled matrix entry (gwdash release-5.8) |
| api | master | schedule | tyk-analytics | valkey8 | postgres17 | murmur128 | 5th scheduled matrix entry |
| api | master | schedule | tyk-analytics | valkey8 | mongo8 | sha256 | 6th scheduled matrix entry |
| api | master | schedule | tyk-pro | redis8 | postgres18 | sha256 | New entry (gwdash master) |
| lts-version | master | schedule | tyk-analytics | — (inherited) | — (inherited) | — (inherited) | Only overrides `gwdash` (master / release-5.8 / release-5.13); cache/db/config inherited from ancestor default |
| ui | default | workflow_dispatch / pull_request / push | tyk-analytics, tyk-pro | redis7 | mongo7 | sha256 | Falls back to global top-level default |
| ui | master | pull_request | tyk-analytics | redis7 | mongo7 | sha256 | Falls back to global default (no override) |
| ui | master | pull_request | tyk-pro | redis8 | mongo8 | sha256 | 1st override matrix entry (gwdash release-5.12) |
| ui | master | pull_request | tyk-pro | valkey7 | postgres17 | murmur128 | 2nd override matrix entry (gwdash release-5.8) |
| ui | master | schedule | tyk-analytics | valkey8 | postgres15 | murmur128 | 1st scheduled matrix entry (gwdash release-5.13) |
| ui | master | schedule | tyk-analytics | redis7 | mongo8 | murmur128 | 2nd scheduled matrix entry (gwdash release-5.8) |
| ui | master | schedule | tyk-analytics | redis8 | postgres17 | sha256 | 3rd scheduled matrix entry (gwdash master) |
| ui | master | push | tyk-analytics, tyk-pro | redis7 | mongo7 | sha256 | Falls back to global default (no override) |

## Key differences between test- and prod-variations

- `api / master` default branch envfile: **test** uses `valkey9_1`, **prod** uses `valkey9` (otherwise identical: postgres15/murmur128).
- `api / master / schedule / tyk-analytics` 3rd matrix entry: **test** uses `db: postgres14`, **prod** uses `db: postgres17` (both `redis8`/`murmur128`).
- `prod-variations.yml` has an extra testsuite, **`lts-version`**, absent from `test-variations.yml`. It only exercises the `gwdash` version (master, release-5.8, release-5.13) against `tyk-analytics` on `master`/`schedule`, inheriting cache/db/config from the ancestor default.
- `ui / master / schedule / tyk-analytics`: **prod** has an extra 3rd envfile entry (`redis8/postgres17/sha256`, gwdash master) that **test** does not have.
