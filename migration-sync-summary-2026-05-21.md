# Migration Sync Summary

Date: 2026-05-21

Source: `/Volumes/Macintosh HD-1/Users/leonidbugaev`

Destination: `/Users/leonidbugaev`

## Completed

- Copied all remote-only top-level `~/go/src` entries identified in `migration-actions-2026-05-21.csv`.
- Copied the `reqforge` repo and related reqforge worktrees by chunked rsync.
- Included repo-local `.claude` state for reqforge/worktrees.
- Verified `reqforge/.claude/worktrees` count matches remote: `22/22`.
- Copied nested missing `github.com` repos:
  - `github.com/asaskevich/govalidator`
  - `github.com/spf13/cast`
  - `github.com/tidwall/gjson`
- Copied divergent existing repos side-by-side:
  - `/Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac`
  - `/Users/leonidbugaev/go/src/tyk-sink-from-broken-mac`
- Created local `tyk` worktree:
  - `/Users/leonidbugaev/go/src/tyk-formal-requirements-policy`
- Copied remote-only `~/Projects` to:
  - `/Users/leonidbugaev/Projects`
- Copied selected work/research Downloads, including:
  - `AI SE Handbook Volume 1.pdf`
  - Tyk AI/governance/token-exchange notes
  - Tyk R&D/questionnaire files
  - FRET/NASA research references
- Staged non-go config files for review at:
  - `/Users/leonidbugaev/migration-from-broken-mac-2026-05-21/config-review`

## Verification

- All remote-only top-level `~/go/src` entries are present locally.
- Key paths verified present:
  - `/Users/leonidbugaev/go/src/reqforge`
  - `/Users/leonidbugaev/go/src/reqforge-wt-signals-core`
  - `/Users/leonidbugaev/go/src/reqforge-worktrees`
  - `/Users/leonidbugaev/go/src/testkube-proof`
  - `/Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac`
  - `/Users/leonidbugaev/go/src/tyk-sink-from-broken-mac`
  - `/Users/leonidbugaev/go/src/tyk-formal-requirements-policy`
  - `/Users/leonidbugaev/Projects`
  - `/Users/leonidbugaev/Downloads/AI SE Handbook Volume 1.pdf`
- Git status sanity checks work for the copied repos/worktrees.
- Tracked files accidentally skipped by broad cache excludes were restored from the broken Mac source.

## Notes

- The first broad rsync attempts were interrupted and are visible in older log files as `SIGINT`/`Broken pipe`. Those were intentional stops during strategy changes.
- Final successful logs are under `migration-sync-logs/`, especially:
  - `reqforge-chunked.log`
  - `sync-tail.log`
- Broad excludes were narrowed after verification so tracked source paths named `build`, `dist`, or `coverage` are not skipped.
- Repo-local `.codex` directories were copied back where tracked files were reported as missing, but global `.codex` history migration was not run because this Codex session is active.
- Claude/Codex global history still requires running:
  - `/Users/leonidbugaev/go/src/gromit/migrate-ai-history-2026-05-21.sh`
  after closing Claude Code and Codex.

## Current Size Snapshot

- `/Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac`: about `17G`
- `/Users/leonidbugaev/go/src/reqforge`: about `6.1G`
- `/Users/leonidbugaev/Projects`: about `4.1G`
- Current root free space after sync: about `118G`
