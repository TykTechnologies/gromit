# AI Metadata Stage Report - 2026-05-21

## Staged Source

Broken Mac source:

- `/Volumes/Macintosh HD-1/Users/leonidbugaev/.claude`
- `/Volumes/Macintosh HD-1/Users/leonidbugaev/.codex`
- `/Volumes/Macintosh HD-1/Users/leonidbugaev/.claude.json`

Local staged copy:

- `/Users/leonidbugaev/migration-from-broken-mac-2026-05-21/ai-metadata-staged/.claude`
- `/Users/leonidbugaev/migration-from-broken-mac-2026-05-21/ai-metadata-staged/.codex`
- `/Users/leonidbugaev/migration-from-broken-mac-2026-05-21/ai-metadata-staged/.claude.json`

## Result

The metadata was copied to local staging first. Live `~/.claude`, `~/.codex`, and `~/.claude.json` were not replaced during staging.

Final staged sizes:

- `.claude`: 2.7G
- `.codex`: 27G
- `.claude.json`: 388K

Key staged files verified present:

- `.claude/history.jsonl`: 1,609,499 bytes, modified May 20 10:19:51 2026
- `.claude.json`: 395,983 bytes, modified May 20 11:34:54 2026
- `.codex/history.jsonl`: 1,331,457 bytes, modified May 20 09:55:43 2026
- `.codex/logs_2.sqlite`: 684,347,392 bytes, modified May 20 11:41:03 2026
- `.codex/state_5.sqlite`: 1,830,912 bytes, modified May 20 11:34:42 2026

## Read Error Retry

The first full `.codex` staging pass reported SMB input/output errors on two session JSONL files under `.codex/sessions/2026/04/02`. Both files succeeded on individual retry, and the final reconciliation pass completed successfully.

## Replacement Command

Close Codex and Claude Code first, then run:

```sh
/Users/leonidbugaev/go/src/gromit/replace-ai-metadata-from-staged-2026-05-21.sh
```

The replacement script backs up current local AI metadata under `/Users/leonidbugaev/migration-backups/ai-metadata-before-replace-YYYYMMDD-HHMMSS/`, then replaces live metadata from the staged local copy.
