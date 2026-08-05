#!/usr/bin/env bash
set -euo pipefail

HOME_DIR="/Users/leonidbugaev"
STAGE_BASE="$HOME_DIR/migration-from-broken-mac-2026-05-21/ai-metadata-staged"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="$HOME_DIR/migration-backups/ai-metadata-before-replace-$STAMP"
LOG_DIR="$HOME_DIR/go/src/gromit/migration-sync-logs/ai-metadata-replace"
LOG="$LOG_DIR/$STAMP-replace.log"
SUCCESS_MARKER="$HOME_DIR/migration-from-broken-mac-2026-05-21/ai-metadata-replaced-successfully.txt"

# Apple rsync 2.6.9 can fail with COPYFILE_UNPACK on staged AppleDouble/xattr
# sidecars. These AI metadata files do not need resource forks, so copy plain
# file contents and normal metadata only.
export COPYFILE_DISABLE=1
RSYNC=(rsync -a)

mkdir -p "$LOG_DIR"
exec > >(tee -a "$LOG") 2>&1

echo "AI metadata replacement started at $(date)"
echo "Stage: $STAGE_BASE"
echo "Backup: $BACKUP_DIR"
echo "Log: $LOG"

if [ -f "$SUCCESS_MARKER" ]; then
  echo
  echo "AI metadata was already replaced successfully according to:"
  echo "  $SUCCESS_MARKER"
  echo "Refusing to run again automatically. Remove that marker only if you intentionally want another replacement."
  exit 1
fi

FIRST_BACKUP="$(find "$HOME_DIR/migration-backups" -maxdepth 1 -type d -name 'ai-metadata-before-replace-*' 2>/dev/null | LC_ALL=C sort | head -n 1 || true)"
if [ -n "$FIRST_BACKUP" ]; then
  echo
  echo "Existing AI metadata backup found and preserved:"
  echo "  $FIRST_BACKUP"
  echo "A fresh backup of the current live state will also be created before this run continues."
fi

for path in "$STAGE_BASE/.claude" "$STAGE_BASE/.codex" "$STAGE_BASE/.claude.json"; do
  if [ ! -e "$path" ]; then
    echo "Missing staged source: $path"
    exit 1
  fi
done

PROCESS_FILE="$(mktemp)"
ps ax -o pid=,command= | awk '
  /replace-ai-metadata-from-staged-2026-05-21/ { next }
  /awk / { next }
  /ps ax -o/ { next }
  /@openai\/codex/ ||
  /\/codex( |$)/ ||
  /Claude Code/ ||
  /\/claude( |$)/ ||
  /\.claude\/shell-snapshots/ {
    print
  }
' > "$PROCESS_FILE"

if [ -s "$PROCESS_FILE" ]; then
  echo
  echo "Refusing to replace live AI metadata while Codex/Claude-related processes are running:"
  cat "$PROCESS_FILE"
  rm -f "$PROCESS_FILE"
  echo
  echo "Close Codex and Claude Code, then rerun this script."
  exit 1
fi
rm -f "$PROCESS_FILE"

mkdir -p "$BACKUP_DIR"

backup_dir() {
  local rel="$1"
  if [ -d "$HOME_DIR/$rel" ]; then
    mkdir -p "$BACKUP_DIR/$rel"
    "${RSYNC[@]}" "$HOME_DIR/$rel/" "$BACKUP_DIR/$rel/"
    echo "Backed up $HOME_DIR/$rel"
  fi
}

backup_file() {
  local rel="$1"
  if [ -f "$HOME_DIR/$rel" ]; then
    mkdir -p "$BACKUP_DIR/$(dirname "$rel")"
    "${RSYNC[@]}" "$HOME_DIR/$rel" "$BACKUP_DIR/$rel"
    echo "Backed up $HOME_DIR/$rel"
  fi
}

backup_dir ".claude"
backup_file ".claude.json"
backup_dir ".codex"

echo
echo "Replacing ~/.claude from staged broken-Mac copy"
mkdir -p "$HOME_DIR/.claude"
"${RSYNC[@]}" --delete "$STAGE_BASE/.claude/" "$HOME_DIR/.claude/"

echo
echo "Replacing ~/.codex from staged broken-Mac copy"
mkdir -p "$HOME_DIR/.codex"
"${RSYNC[@]}" --delete "$STAGE_BASE/.codex/" "$HOME_DIR/.codex/"

echo
echo "Replacing ~/.claude.json from staged broken-Mac copy"
"${RSYNC[@]}" "$STAGE_BASE/.claude.json" "$HOME_DIR/.claude.json"

echo
echo "Verifying key files"
for rel in \
  ".claude/history.jsonl" \
  ".claude.json" \
  ".codex/history.jsonl" \
  ".codex/logs_2.sqlite" \
  ".codex/state_5.sqlite"
do
  if [ ! -e "$HOME_DIR/$rel" ]; then
    echo "Missing destination key file: $HOME_DIR/$rel"
    exit 1
  fi
  if [ -f "$STAGE_BASE/$rel" ] && ! cmp -s "$STAGE_BASE/$rel" "$HOME_DIR/$rel"; then
    echo "Destination key file differs from staged source: $rel"
    exit 1
  fi
  stat -f '%z bytes %Sm %N' "$HOME_DIR/$rel"
done

echo
echo "Verifying top-level metadata entries"
for rel in .claude .codex; do
  DIFF_FILE="$(mktemp)"
  comm -3 \
    <(find "$STAGE_BASE/$rel" -maxdepth 1 -mindepth 1 -print | sed "s#^$STAGE_BASE/$rel/##" | LC_ALL=C sort) \
    <(find "$HOME_DIR/$rel" -maxdepth 1 -mindepth 1 -print | sed "s#^$HOME_DIR/$rel/##" | LC_ALL=C sort) \
    > "$DIFF_FILE"
  if [ -s "$DIFF_FILE" ]; then
    echo "Top-level mismatch for $rel:"
    cat "$DIFF_FILE"
    rm -f "$DIFF_FILE"
    exit 1
  fi
  rm -f "$DIFF_FILE"
  echo "$rel top-level entries match"
done

echo
du -sh "$HOME_DIR/.claude" "$HOME_DIR/.codex" "$HOME_DIR/.claude.json"
{
  echo "AI metadata replacement completed at $(date)"
  echo "Stage: $STAGE_BASE"
  echo "Backup from this run: $BACKUP_DIR"
  if [ -n "${FIRST_BACKUP:-}" ]; then
    echo "Oldest preserved backup: $FIRST_BACKUP"
  fi
  echo "Log: $LOG"
} > "$SUCCESS_MARKER"
echo "AI metadata replacement finished at $(date)"
