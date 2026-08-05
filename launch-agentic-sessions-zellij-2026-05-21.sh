#!/usr/bin/env bash
set -euo pipefail

ROOT="/Users/leonidbugaev/go/src/gromit"
LAYOUT="$ROOT/zellij-agentic-sessions-2026-05-21.kdl"
SESSION_NAME="${1:-old-mac-agentic-sessions}"

missing=0

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing command: $1"
    missing=1
  fi
}

need_dir() {
  if [ ! -d "$1" ]; then
    echo "Missing folder: $1"
    missing=1
  fi
}

need_codex_session() {
  local id="$1"
  if ! find "$HOME/.codex/sessions" -type f -name "*$id.jsonl" -print -quit 2>/dev/null | grep -q .; then
    echo "Missing live Codex session metadata: $id"
    missing=1
  fi
}

need_claude_session() {
  local id="$1"
  if ! find "$HOME/.claude/projects" -type f -name "$id.jsonl" -print -quit 2>/dev/null | grep -q .; then
    echo "Missing live Claude session metadata: $id"
    missing=1
  fi
}

need_cmd zellij
need_cmd codex
need_cmd claude

need_dir /Users/leonidbugaev/go/src/reqforge
need_dir /Users/leonidbugaev/go/src/fret
need_dir /Users/leonidbugaev/go/src/tyk
need_dir /Users/leonidbugaev/go/src/fret/fret-electron
need_dir /Users/leonidbugaev/go/src/list-docker-cves
need_dir /Users/leonidbugaev/go/src/tyk-performance-testing
need_dir /Users/leonidbugaev/go/src/graphql-go-tools-proof
need_dir /Users/leonidbugaev/go/src/jsonparser
need_dir /Users/leonidbugaev/go/src/helpwanted.dev

need_codex_session 019d4580-38c3-7970-81d3-841199cd35cb
need_codex_session 019df6ce-6def-7c63-a711-7981798d3626
need_codex_session 019d4ea8-8a93-7801-8f19-ca92fcd0ee30
need_codex_session 019c03f2-90f6-7740-8b56-b7425462c49e
need_codex_session 019df6fe-f694-7451-b21f-55cfe7cbc300
need_codex_session 019df376-2fc3-7490-aaeb-7ec6c557b69a
need_codex_session 019df2a3-b7ae-7393-a5be-732ec22f5aca
need_codex_session 019d876c-e93d-7ef2-b396-b010838bcce6
need_codex_session 019d646c-d39d-7343-b101-104ea390ec05
need_codex_session 019cf06f-de85-7383-a38d-d02ccb77e42e

need_claude_session ba9c8515-3817-419b-811d-0bfbe3309083
need_claude_session 978f8ef4-2e16-46a1-8a4d-ef2d381f47d2
need_claude_session 36ed7149-c5c4-4639-bd60-5379c76298ef
need_claude_session 3959a225-28c9-4744-9999-930e629c1931
need_claude_session 0ab98529-666f-4359-9fc6-6f359fe26221
need_claude_session 638361b0-f592-4307-b1c2-dc8d97c911aa
need_claude_session 455a3e3b-368b-45e9-afec-f8112d966d16
need_claude_session 6fb1405f-9eb4-4927-8190-b00ad56142df

if [ "$missing" -ne 0 ]; then
  cat <<'MSG'

Cannot launch cleanly yet.

If the missing items are only AI session metadata, close Codex and Claude Code,
then run:

  /Users/leonidbugaev/go/src/gromit/replace-ai-metadata-from-staged-2026-05-21.sh

Then rerun this launcher.
MSG
  exit 1
fi

session_line="$(zellij list-sessions 2>/dev/null | grep -F "$SESSION_NAME" || true)"
if printf '%s\n' "$session_line" | grep -q "EXITED"; then
  echo "Deleting dead Zellij session: $SESSION_NAME"
  zellij delete-session "$SESSION_NAME"
elif [ -n "$session_line" ]; then
  echo "Attaching to existing Zellij session: $SESSION_NAME"
  exec zellij attach "$SESSION_NAME"
fi

exec zellij --session "$SESSION_NAME" --layout "$LAYOUT"
