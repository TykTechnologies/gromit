# Broken MacBook Migration Gap Report

Date: 2026-05-21

Source, broken new MacBook: `/Volumes/Macintosh HD-1/Users/leonidbugaev`

Destination, current machine: `/Users/leonidbugaev`

Primary scope inspected: `~/go/src`

## Executive Summary

The SMB share is mounted and readable at `/Volumes/Macintosh HD-1`.

Top-level `~/go/src` comparison:

- Broken MacBook has 263 top-level entries.
- Current machine has 231 top-level entries.
- Broken MacBook has 32 top-level entries missing from the current machine.
- Current machine has 0 top-level entries missing from the broken MacBook.

The highest priority migration work is in `~/go/src`. Most of the new work on the broken MacBook is in proof/reqforge/libxml2/jsonparser/kubernetes related repositories and worktrees.

Do not blindly overwrite existing local repositories. `tyk`, `tyk-analytics`, `tyk-docs`, and `tyk-sink` already exist locally and need branch-aware handling.

## Top-Level Entries Present Only On Broken MacBook

These are present in `/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src` and absent from `/Users/leonidbugaev/go/src`:

```text
.claude
.wrangler
ShowProfit
customer-insights
dailydev_hackaton
encoding-json-v2-proof
fret
graphql-go-tools-proof
graphql-go-tools-proof-bin
graphql-go-tools-proof-workspace
jsonparser
jsonparser-proof
kubernetes
libxml2
libxml2-gnome-proof
libxml2-pr-compatible
license-analysis
list-docker-cves
probe
proof-action
proof-coverage
proof-solidity-demo
reqforge
reqforge-release-proof-fix
reqforge-worktrees
reqforge-wt-signals-cli
reqforge-wt-signals-core
reqforge-wt-signals-fixtures
reqforge-wt-signals-workflow
reqproof-proof
testkube-proof
tyk-proof
```

## Recently Active Top-Level Entries On Broken MacBook

These had top-level mtimes within the last 35 days:

```text
dailydev_hackaton
encoding-json-v2-proof
fret
github.com
graphql-go-tools-proof
graphql-go-tools-proof-bin
graphql-go-tools-proof-workspace
jsonparser
jsonparser-proof
kubernetes
libxml2
libxml2-gnome-proof
libxml2-pr-compatible
list-docker-cves
node_modules
probe
proof-action
proof-coverage
proof-solidity-demo
reqforge
reqforge-release-proof-fix
reqforge-worktrees
reqforge-wt-signals-cli
reqforge-wt-signals-core
reqforge-wt-signals-fixtures
reqforge-wt-signals-workflow
reqproof-proof
testkube-proof
tyk
tyk-analytics
tyk-docs
tyk-proof
tyk-sink
```

## Remote-Only Git State Summary

Copy these as whole directories, preserving `.git`, unless noted otherwise.

| Repo | Broken Mac branch | Broken Mac head | Dirty summary | Action |
|---|---:|---:|---|---|
| `ShowProfit` | `refactor/split-settlement-app` | `ae998e3` | 1 modified, 267 untracked | Copy entire repo to current machine. Review generated traces/db files after copy. |
| `customer-insights` | `main` | `4bbb543` | clean | Copy entire repo if needed. |
| `dailydev_hackaton` | `main` | no commits yet | 5 untracked | Copy entire repo. It has no committed history yet. |
| `encoding-json-v2-proof` | `main` | `d37b317` | 1 deleted, 1 modified, 3 untracked | Copy entire repo. |
| `fret` | `master` | `58db455` | 3 modified, 65 untracked | Copy entire repo. Watch for `tyk-org-secrets.txt` and related secret inventory files. |
| `graphql-go-tools-proof` | `audit-disclosure-panic-dos` | `9fcfffd9` | clean | Copy entire repo. |
| `jsonparser` | `fix-oss-fuzz-delete-leading-comma` | `b0adab0` | clean | Copy entire repo. |
| `kubernetes` | `master` | `8f511d81` | 2 modified | Copy entire repo. |
| `libxml2` | `main` | `c28108f8` | 1 modified, 10 untracked | Copy entire repo. |
| `libxml2-gnome-proof` | `feat/gnome-proof-dogfood` | `9b5849e9` | clean | Copy entire repo. |
| `libxml2-pr-compatible` | `feat/gnome-proof-dogfood-pr` | `60c588e4` | 15 modified, 16 untracked | Copy entire repo. |
| `license-analysis` | `main` | `13e4280` | clean | Copy entire repo. |
| `list-docker-cves` | `main` | `72e6d59` | 9 untracked | Copy entire repo. |
| `probe` | `feat/proof-rust-dogfood` | `cf18ab81` | 1 untracked | Copy entire repo. |
| `proof-action` | `main` | `09a843c` | clean | Copy entire repo. |
| `proof-coverage` | `main` | `048030e` | clean | Copy entire repo. |
| `proof-solidity-demo` | `main` | `13bad42` | 15 modified | Copy entire repo. |
| `reqforge` | `feat/solidity-support-71` | `3573ad69` | 121 modified, 26 untracked | Copy `reqforge` together with all `reqforge-*` worktrees. |
| `testkube-proof` | `main` | `5334924` | 705 untracked | Copy entire repo. This looks like generated proof/SRS material plus tests. |

## Reqforge Worktree Set

The following broken-Mac entries are git worktrees pointing into `/Users/leonidbugaev/go/src/reqforge/.git/worktrees/...` on that machine:

```text
reqforge-release-proof-fix
reqforge-wt-signals-cli
reqforge-wt-signals-core
reqforge-wt-signals-fixtures
reqforge-wt-signals-workflow
```

Because the `.git` files use absolute paths, copy the whole reqforge set into the same destination path layout:

```text
/Users/leonidbugaev/go/src/reqforge
/Users/leonidbugaev/go/src/reqforge-release-proof-fix
/Users/leonidbugaev/go/src/reqforge-worktrees
/Users/leonidbugaev/go/src/reqforge-wt-signals-cli
/Users/leonidbugaev/go/src/reqforge-wt-signals-core
/Users/leonidbugaev/go/src/reqforge-wt-signals-fixtures
/Users/leonidbugaev/go/src/reqforge-wt-signals-workflow
```

Observed worktree states:

| Worktree | Branch/head | Dirty summary |
|---|---|---|
| `reqforge-release-proof-fix` | detached `8cc433a5` | clean from targeted status |
| `reqforge-wt-signals-cli` | `feat/ext-signals-cli` @ `7981180b` | clean |
| `reqforge-wt-signals-core` | `feat/ext-signals-core` @ `a1402b56` | 2 modified requirement YAML files |
| `reqforge-wt-signals-fixtures` | `feat/ext-signals-fixtures` @ `966c1c27` | clean |
| `reqforge-wt-signals-workflow` | `feat/ext-signals-workflow` @ `7640cfe2` | clean |

## Nested `github.com` Additions

Broken MacBook has these nested repos under `~/go/src/github.com` that are missing locally:

| Repo | Broken Mac head | Status |
|---|---:|---|
| `github.com/asaskevich/govalidator` | `f21760c` | detached HEAD, clean |
| `github.com/spf13/cast` | `fc73346` | detached HEAD, clean |
| `github.com/tidwall/gjson` | `133f42c` | detached HEAD, clean |

Copy them if you need reproducible dependency checkouts or local experimentation state.

## Existing Local Repositories Requiring Care

These exist on both machines, so do not overwrite the local copies.

| Repo | Current machine | Broken MacBook | Recommendation |
|---|---|---|---|
| `tyk` | `fix/oas-security-scheme-race-condition-7573` @ `02c594c0d`, 14 untracked | `experiment/formal-requirements-policy` @ `f5b63df8e`, clean | The broken-Mac commit exists locally. Create a local worktree/branch for `f5b63df8e`; do not overwrite current `tyk` because it has untracked local files. |
| `tyk-analytics` | `master` @ `7991ffc8a`, 1 modified `webclient` | `fix/disable-fips-release-5.12` @ `6135f442a`, 1 modified `webclient` | Broken-Mac commit is missing locally. Copy broken repo to a side directory first, then reconcile `webclient`. |
| `tyk-docs` | `feature/ai-studio-install-updates` @ `287bcf018`, clean | same branch/head, 1 untracked `.claude/settings.local.json` | Only copy `.claude/settings.local.json` if wanted. |
| `tyk-sink` | `master` @ `ba9d77f`, 72 untracked `.desc`/output files | `TT-16932-cve-fixes` @ `a26a088`, similar untracked `.desc`/output files | Broken-Mac commit is missing locally. Copy broken repo to a side directory or import as bundle before merging. |

## Recommended Copy-Back Plan

1. Create a safety staging area on the current machine:

   ```sh
   mkdir -p /Users/leonidbugaev/migration-from-broken-mac-2026-05-21
   ```

2. Copy all top-level entries that are missing locally, preserving metadata and git data, while excluding reproducible dependency/build caches:

   ```sh
   rsync -aEH --info=progress2 \
     --exclude-from "/Users/leonidbugaev/go/src/gromit/migration-rsync-excludes-2026-05-21.txt" \
     "/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src/REPO_NAME/" \
     "/Users/leonidbugaev/go/src/REPO_NAME/"
   ```

   Use this for every row marked `copy-whole` in `migration-actions-2026-05-21.csv`.

3. Copy the reqforge set as one batch before running git commands inside any reqforge worktree:

   ```sh
   for name in reqforge reqforge-release-proof-fix reqforge-worktrees reqforge-wt-signals-cli reqforge-wt-signals-core reqforge-wt-signals-fixtures reqforge-wt-signals-workflow; do
     rsync -aEH --info=progress2 \
       --exclude-from "/Users/leonidbugaev/go/src/gromit/migration-rsync-excludes-2026-05-21.txt" \
       "/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src/$name/" \
       "/Users/leonidbugaev/go/src/$name/"
   done
   ```

4. For existing divergent repos, copy to side-by-side recovery directories first:

   ```sh
   rsync -aEH --info=progress2 \
     --exclude-from "/Users/leonidbugaev/go/src/gromit/migration-rsync-excludes-2026-05-21.txt" \
     "/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src/tyk-analytics/" \
     "/Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac/"

   rsync -aEH --info=progress2 \
     --exclude-from "/Users/leonidbugaev/go/src/gromit/migration-rsync-excludes-2026-05-21.txt" \
     "/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src/tyk-sink/" \
     "/Users/leonidbugaev/go/src/tyk-sink-from-broken-mac/"
   ```

   For `tyk`, prefer a local worktree from the existing commit:

   ```sh
   git -C /Users/leonidbugaev/go/src/tyk worktree add /Users/leonidbugaev/go/src/tyk-formal-requirements-policy f5b63df8e
   ```

5. Verify after copy:

   ```sh
   git -C /Users/leonidbugaev/go/src/reqforge status --short --branch
   git -C /Users/leonidbugaev/go/src/libxml2-pr-compatible status --short --branch
   git -C /Users/leonidbugaev/go/src/testkube-proof status --short --branch
   git -C /Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac status --short --branch
   git -C /Users/leonidbugaev/go/src/tyk-sink-from-broken-mac status --short --branch
   ```

6. After verification, decide which copied generated/cache artifacts to keep. Good candidates for review before committing or archiving:

   ```text
   .proof/index.db
   .proof/audit-cache.json
   .reqproof/index.db
   debug-artifacts/
   output/traces/
   *.sqlite
   *.db
   *.db-shm
   *.db-wal
   node_modules/
   ```

## Space Estimate

Current free space on `/`: about 162 GiB.

Measured AI history/config from the broken MacBook:

| Area | Size |
|---|---:|
| Full `.claude` | 2.72 GiB |
| `.claude/projects` | 2.31 GiB |
| Full `.codex` | 26.78 GiB |
| `.codex/sessions` | 13.23 GiB |
| `.codex/log` raw log directory | 12.81 GiB |
| `.codex/logs_2.sqlite` | 0.64 GiB |

Measured selected project directories that responded quickly over SMB total at least 1.39 GiB. Several large/high-file-count project directories timed out during a 20 second per-directory SMB size scan, including `ShowProfit`, `fret`, `kubernetes`, `libxml2*`, `reqforge*`, `probe`, and `tyk-analytics`.

Practical space guidance:

- Reserve at least 50 GiB for the project copy-back plus Claude and Codex history excluding raw Codex logs.
- Reserve about 80 GiB if you also want the raw `.codex/log` directory and all generated project artifacts.
- The current machine has enough free space for either target based on the measured 162 GiB available.

Detailed size results are stored in `migration-size-report-2026-05-21.csv`.

## Claude Code And Codex History Migration

Claude/Codex migration should not be performed while either tool is running. This current session is itself Codex, so I prepared a script instead of mutating live Codex state underneath the running process.

Prepared script:

```text
migrate-ai-history-2026-05-21.sh
```

The script backs up local history first, then:

- Merges Claude Code directories such as `.claude/projects`, `.claude/file-history`, `.claude/todos`, `.claude/tasks`, and `.claude/sessions` with `rsync --backup`.
- Merges `.claude/history.jsonl` by unique JSONL line.
- Replaces `.claude.json` after backup because a reliable semantic merge is not guaranteed.
- Merges Codex `.codex/sessions`, `.codex/shell_snapshots`, `.codex/memories`, and `.codex/rules` with `rsync --backup`.
- Merges `.codex/history.jsonl` by unique JSONL line.
- Replaces Codex SQLite history/state files after backup because SQLite merge is not practical.
- Skips the raw `.codex/log` directory by default because it is about 12.81 GiB. Run with `MIGRATE_CODEX_RAW_LOG=1` if you want it too.

Run after closing Claude Code and Codex:

```sh
chmod +x /Users/leonidbugaev/go/src/gromit/migrate-ai-history-2026-05-21.sh
/Users/leonidbugaev/go/src/gromit/migrate-ai-history-2026-05-21.sh
```

To include raw Codex logs:

```sh
MIGRATE_CODEX_RAW_LOG=1 /Users/leonidbugaev/go/src/gromit/migrate-ai-history-2026-05-21.sh
```

## Folder Structure Preservation

For projects copied from the broken MacBook, use `rsync -aEH` from the source directory with a trailing slash into the destination directory with the same top-level name. For remote-only projects, the destination does not exist yet, so the resulting folder structure will match the broken MacBook copy.

Use `migration-rsync-excludes-2026-05-21.txt` for code copies. This intentionally does not preserve reproducible generated folders such as `node_modules`, `.next`, `dist`, `build`, `.cache`, Go build cache style folders, Python caches, Rust `target`, and trace/debug outputs. That should substantially reduce copy size for repositories like `tyk-analytics` while keeping source, git history, configuration, and project layout.

For existing divergent repositories, keep side-by-side recovery directories first, for example:

```text
/Users/leonidbugaev/go/src/tyk-analytics-from-broken-mac
/Users/leonidbugaev/go/src/tyk-sink-from-broken-mac
```

This preserves the broken-Mac folder structure without overwriting current-machine work.

## Non-Go Src Candidates

Additional migration candidates outside `~/go/src` are tracked in:

```text
migration-non-go-candidates-2026-05-21.csv
```

Recommended high-value items:

- `.claude`, `.claude.json`, and `.codex`: already covered by `migrate-ai-history-2026-05-21.sh`.
- `~/Projects`: copy the whole tree with cache/build excludes. It is missing locally and contains `newsletter`, `sherpa-onnx`, `CosyVoice`, `fish-speech`, `silero-tts`, and related work folders.
- `.gitconfig`, `.zshrc`, `.zprofile`, `.config`: review and merge, because the broken Mac has newer developer environment config.
- `.ssh`: review carefully and copy only missing/new key or config files; do not overwrite local keys blindly.
- `~/bin`: merge personal scripts with backup.
- `~/conductor`: review as a likely project/work directory outside `go/src`.
- `Downloads` and `Desktop`: selectively copy recent work artifacts/screenshots/docs, not the whole folder.

Lower priority or skip:

- `.aws` and `.kube`: timestamps match.
- `.docker`: local copy is newer; only review contexts if something is missing.
- `Applications`: inventory/reinstall preferred over copying app bundles.

## Downloads And Desktop Review

Detailed review candidates are stored in:

```text
migration-downloads-desktop-review-2026-05-21.csv
```

Conclusion:

- Desktop: no copy needed from the current evidence. All 441 remote top-level Desktop entries are already present locally. Recent Desktop items are screenshots and game/app bundles.
- Downloads: there are important-looking remote-only files. Copy selected work artifacts, especially Tyk/AI governance notes, token exchange docs, agentic delivery docs, AI SE/FRET/NASA research references, and Tyk questionnaire/report files.
- Downloads also contains sensitive credential-looking files such as `client_secret_*.json`, `leo-engineering-*.json`, and `authenticator (1).txt`. Treat those as secure-review items, not normal documents.

## Projects Folder

The broken Mac has a remote-only `~/Projects` tree:

```text
/Volumes/Macintosh HD-1/Users/leonidbugaev/Projects
```

The current machine does not have `/Users/leonidbugaev/Projects`.

Top-level entries observed:

```text
CosyVoice
buger
fish-speech
newsletter
sherpa-onnx
silero-tts
speech-pattern-detector
```

This should be migrated as a whole tree, preserving folder structure but excluding generated caches/build artifacts:

```sh
rsync -aEH --info=progress2 \
  --exclude-from "/Users/leonidbugaev/go/src/gromit/migration-rsync-excludes-2026-05-21.txt" \
  "/Volumes/Macintosh HD-1/Users/leonidbugaev/Projects/" \
  "/Users/leonidbugaev/Projects/"
```

The exact `AI SE Handbook Volume 1.pdf` was found at:

```text
/Volumes/Macintosh HD-1/Users/leonidbugaev/Downloads/AI SE Handbook Volume 1.pdf
```

It was not present at the top level of `Projects/newsletter`; the `newsletter` folder itself contains useful work material such as `article-*.md`, `framework*.md`, `probelabs_*`, `rd-report-research-feb2025-jan2026.md`, plus `ai/` and `nasa/` subfolders.

## Notes And Limits

- I did not modify or copy any source files while producing this report.
- The SMB filesystem was high latency for broad metadata scans. I avoided full recursive diffs over every file and focused on top-level deltas, recent activity, and git status for engineering work under `~/go/src`.
- A complete byte-for-byte diff of all files is still possible, but it should be run as a staged `rsync --dry-run --checksum` pass per selected repo rather than one huge scan over the whole SMB share.
