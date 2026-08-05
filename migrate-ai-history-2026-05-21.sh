#!/usr/bin/env bash
set -euo pipefail

SRC_HOME="/Volumes/Macintosh HD-1/Users/leonidbugaev"
DST_HOME="/Users/leonidbugaev"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="$DST_HOME/migration-backups/ai-history-$STAMP"

ps ax -o pid=,command= | awk '
  /[Cc]laude|[Cc]odex|codex/ &&
  $0 !~ /migrate-ai-history-2026-05-21/ &&
  $0 !~ /awk / &&
  $0 !~ /ps ax -o/ {
    print
  }
' >/tmp/ai-history-processes.$$

if [ -s /tmp/ai-history-processes.$$ ]; then
  echo "Claude/Codex-like processes are running. Close Claude Code and Codex, then rerun."
  cat /tmp/ai-history-processes.$$
  rm -f /tmp/ai-history-processes.$$
  exit 1
fi
rm -f /tmp/ai-history-processes.$$

mkdir -p "$BACKUP_DIR"

backup_path() {
  local rel="$1"
  if [ -e "$DST_HOME/$rel" ]; then
    mkdir -p "$BACKUP_DIR/$(dirname "$rel")"
    rsync -aEH "$DST_HOME/$rel" "$BACKUP_DIR/$rel"
  fi
}

merge_jsonl_unique() {
  local src="$1"
  local dst="$2"
  mkdir -p "$(dirname "$dst")"
  touch "$dst"
  local tmp
  tmp="$(mktemp)"
  awk 'NF && !seen[$0]++' "$dst" "$src" > "$tmp"
  mv "$tmp" "$dst"
}

echo "Backing up local AI history/config to $BACKUP_DIR"
backup_path ".claude"
backup_path ".claude.json"
backup_path ".codex/history.jsonl"
backup_path ".codex/sessions"
backup_path ".codex/shell_snapshots"
backup_path ".codex/state_5.sqlite"
backup_path ".codex/state_5.sqlite-shm"
backup_path ".codex/state_5.sqlite-wal"
backup_path ".codex/logs_2.sqlite"
backup_path ".codex/logs_2.sqlite-shm"
backup_path ".codex/logs_2.sqlite-wal"

echo "Merging Claude Code history directories"
mkdir -p "$DST_HOME/.claude"
for rel in projects file-history todos tasks sessions shell-snapshots paste-cache plans agents commands plugins session-env; do
  if [ -e "$SRC_HOME/.claude/$rel" ]; then
    mkdir -p "$DST_HOME/.claude/$rel"
    rsync -aEH --backup --suffix=".$STAMP.local" "$SRC_HOME/.claude/$rel/" "$DST_HOME/.claude/$rel/"
  fi
done

if [ -f "$SRC_HOME/.claude/history.jsonl" ]; then
  echo "Merging .claude/history.jsonl by unique JSONL line"
  merge_jsonl_unique "$SRC_HOME/.claude/history.jsonl" "$DST_HOME/.claude/history.jsonl"
fi

if [ -f "$SRC_HOME/.claude.json" ]; then
  echo "Replacing .claude.json after backup because reliable structural merge is not guaranteed"
  rsync -aEH "$SRC_HOME/.claude.json" "$DST_HOME/.claude.json"
fi

echo "Merging Codex history directories"
mkdir -p "$DST_HOME/.codex"
for rel in sessions shell_snapshots memories rules; do
  if [ -e "$SRC_HOME/.codex/$rel" ]; then
    mkdir -p "$DST_HOME/.codex/$rel"
    rsync -aEH --backup --suffix=".$STAMP.local" "$SRC_HOME/.codex/$rel/" "$DST_HOME/.codex/$rel/"
  fi
done

if [ -f "$SRC_HOME/.codex/history.jsonl" ]; then
  echo "Merging .codex/history.jsonl by unique JSONL line"
  merge_jsonl_unique "$SRC_HOME/.codex/history.jsonl" "$DST_HOME/.codex/history.jsonl"
fi

echo "Replacing Codex SQLite history/state databases after backup"
for rel in logs_2.sqlite logs_2.sqlite-shm logs_2.sqlite-wal state_5.sqlite state_5.sqlite-shm state_5.sqlite-wal; do
  if [ -f "$SRC_HOME/.codex/$rel" ]; then
    rsync -aEH "$SRC_HOME/.codex/$rel" "$DST_HOME/.codex/$rel"
  fi
done

if [ "${MIGRATE_CODEX_RAW_LOG:-0}" = "1" ] && [ -e "$SRC_HOME/.codex/log" ]; then
  echo "Migrating raw Codex log directory because MIGRATE_CODEX_RAW_LOG=1"
  mkdir -p "$DST_HOME/.codex/log"
  rsync -aEH --backup --suffix=".$STAMP.local" "$SRC_HOME/.codex/log/" "$DST_HOME/.codex/log/"
else
  echo "Skipping .codex/log by default; set MIGRATE_CODEX_RAW_LOG=1 to copy the extra ~12.8 GiB raw log directory."
fi

echo "Done. Backup is at $BACKUP_DIR"
