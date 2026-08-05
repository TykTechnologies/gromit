#!/usr/bin/env bash
set -euo pipefail

SRC_BASE="/Volumes/Macintosh HD-1/Users/leonidbugaev/go/src"
DST_BASE="/Users/leonidbugaev/go/src"
ROOT="/Users/leonidbugaev/go/src/gromit"
EXCLUDES="$ROOT/migration-rsync-excludes-2026-05-21.txt"
STAMP="$(date +%Y%m%d-%H%M%S)"
LOG_DIR="$ROOT/migration-sync-logs/$STAMP"
mkdir -p "$LOG_DIR"
exec > >(tee -a "$LOG_DIR/reqforge-chunked.log") 2>&1

skip_generated_top() {
  case "$1" in
    .DS_Store|.codex|.wrangler|node_modules|dist|build|coverage|.cache|tmp|.tmp|target|.venv|venv|env)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

copy_entry() {
  local src_dir="$1"
  local dst_dir="$2"
  local entry="$3"
  local src="$src_dir/$entry"

  if skip_generated_top "$entry"; then
    echo "SKIP generated/cache top-level entry: $src"
    return 0
  fi

  if [ "$entry" = ".claude" ] && [ -d "$src" ]; then
    copy_claude_dir "$src" "$dst_dir/.claude"
    return 0
  fi

  if [ "$entry" = ".git" ] && [ -d "$src" ]; then
    echo "  rsync metadata dir: $entry"
    rsync -a --partial "$src" "$dst_dir/"
  else
    echo "  rsync entry: $entry"
    rsync -a --partial --exclude-from "$EXCLUDES" "$src" "$dst_dir/"
  fi
}

copy_claude_dir() {
  local src="$1"
  local dst="$2"

  echo "  rsync .claude in chunks"
  mkdir -p "$dst"

  while IFS= read -r sub; do
    [ -n "$sub" ] || continue
    if [ "$sub" = "worktrees" ] && [ -d "$src/$sub" ]; then
      mkdir -p "$dst/worktrees"
      while IFS= read -r agent; do
        [ -n "$agent" ] || continue
        echo "    rsync .claude/worktrees/$agent"
        rsync -a --partial --exclude-from "$EXCLUDES" "$src/worktrees/$agent" "$dst/worktrees/"
      done < <(/bin/ls -A "$src/worktrees")
    else
      echo "    rsync .claude/$sub"
      rsync -a --partial --exclude-from "$EXCLUDES" "$src/$sub" "$dst/"
    fi
  done < <(/bin/ls -A "$src")
}

copy_tree_chunked() {
  local name="$1"
  local src_dir="$SRC_BASE/$name"
  local dst_dir="$DST_BASE/$name"

  echo
  echo "==> chunked rsync $name"
  echo "    $src_dir"
  echo " -> $dst_dir"

  if [ ! -d "$src_dir" ]; then
    echo "SKIP missing source dir: $src_dir"
    return 0
  fi

  mkdir -p "$dst_dir"

  while IFS= read -r entry; do
    [ -n "$entry" ] || continue
    copy_entry "$src_dir" "$dst_dir" "$entry"
  done < <(/bin/ls -A "$src_dir")
}

for name in \
  reqforge \
  reqforge-release-proof-fix \
  reqforge-worktrees \
  reqforge-wt-signals-cli \
  reqforge-wt-signals-core \
  reqforge-wt-signals-fixtures \
  reqforge-wt-signals-workflow
do
  copy_tree_chunked "$name"
done

echo
echo "Chunked reqforge/worktree rsync finished at $(date)"
