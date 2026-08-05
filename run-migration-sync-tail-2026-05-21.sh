#!/usr/bin/env bash
set -euo pipefail

SRC_HOME="/Volumes/Macintosh HD-1/Users/leonidbugaev"
DST_HOME="/Users/leonidbugaev"
ROOT="/Users/leonidbugaev/go/src/gromit"
EXCLUDES="$ROOT/migration-rsync-excludes-2026-05-21.txt"
STAMP="$(date +%Y%m%d-%H%M%S)"
LOG_DIR="$ROOT/migration-sync-logs/$STAMP"
STAGED_CONFIG="$DST_HOME/migration-from-broken-mac-2026-05-21/config-review"

mkdir -p "$LOG_DIR" "$STAGED_CONFIG"
exec > >(tee -a "$LOG_DIR/sync-tail.log") 2>&1

rsync_dir() {
  local src="$1"
  local dst="$2"
  local label="$3"
  if [ ! -e "$src" ]; then echo "SKIP missing source: $src"; return 0; fi
  echo
  echo "==> $label"
  mkdir -p "$dst"
  rsync -a --partial --exclude-from "$EXCLUDES" "$src/" "$dst/"
}

rsync_file() {
  local src="$1"
  local dst="$2"
  local label="$3"
  if [ ! -f "$src" ]; then echo "SKIP missing file: $src"; return 0; fi
  echo
  echo "==> $label"
  mkdir -p "$(dirname "$dst")"
  rsync -a --partial "$src" "$dst"
}

echo "Tail migration sync started at $(date)"

for name in reqproof-proof testkube-proof tyk-proof; do
  rsync_dir "$SRC_HOME/go/src/$name" "$DST_HOME/go/src/$name" "go/src/$name"
done

for rel in github.com/asaskevich/govalidator github.com/spf13/cast github.com/tidwall/gjson; do
  rsync_dir "$SRC_HOME/go/src/$rel" "$DST_HOME/go/src/$rel" "go/src/$rel"
done

rsync_dir "$SRC_HOME/go/src/tyk-analytics" "$DST_HOME/go/src/tyk-analytics-from-broken-mac" "divergent tyk-analytics side copy"
rsync_dir "$SRC_HOME/go/src/tyk-sink" "$DST_HOME/go/src/tyk-sink-from-broken-mac" "divergent tyk-sink side copy"

if [ -d "$DST_HOME/go/src/tyk" ] && [ ! -e "$DST_HOME/go/src/tyk-formal-requirements-policy" ]; then
  echo
  echo "==> Creating tyk worktree for broken-Mac commit f5b63df8e"
  git -C "$DST_HOME/go/src/tyk" worktree add "$DST_HOME/go/src/tyk-formal-requirements-policy" f5b63df8e || true
fi

rsync_dir "$SRC_HOME/Projects" "$DST_HOME/Projects" "remote-only ~/Projects"

DOWNLOAD_FILES=(
  "tyk-eu-ai-act-data-act-api-ai-governance-article.md"
  "tyk-ai-studio-eu-ai-act-data-act-readiness.md"
  "Audit_Report_PreDex.pdf"
  "system_truth_layer_proposal.docx"
  "agentic_delivery_provocation.pdf"
  "CVBGF-AI augmented delivery - working approach and hyptothesis-090526-095633.pdf"
  "agentic_exec_summary.pdf"
  "AI SE Handbook Volume 1.pdf"
  "69f24d3e09b921b92403774e_Claude-Deploying-Claude-Across-Your-Organization-04292026.pdf"
  "tyk-token-exchange-flow-examples.md"
  "tyk-token-exchange-rfc-review.md"
  "tyk-token-exchange-market-research.md"
  "EN-Token Exchange in externalOAuth-290426-120924.pdf"
  "Tyk Technologies - Questionnaire - API Management Software, Q3 2026(Questionnaire).xlsx"
  "Tyk Technologies - Questionnaire - API Management Software, Q3 2026(Questionnaire) - Tyk Technologies - Questionnaire - API Management Software, Q3 2026(Questionnaire).csv"
  "Tyk_RD_Report_Feb2025_Jan2026.docx"
  "tyk-rd-report-2025-research.md"
  "Docker_DHI_SMBPackage_Tyk_2026.pdf"
  "Technical_Report_Copilot_FRET (4).pdf"
  "Technical_Report_Copilot_FRET (4) (1).pdf"
  "Technical_Report_Copilot_FRET (4) (2).pdf"
  "FretRequirements.pdf"
  "FretRequirements (1).pdf"
  "FRET-SWS-May2020.pdf"
  "FRET2021Sfymmhv2.pdf"
  "TechnicalReport_NASA_FRET_PLCverif.pdf"
  "FINAL NFM23_NASA_FRET_PLCverif.pdf"
  "TechnicalReport__FRET_Realizability_Checking.pdf"
  "TechnicalReport__Unrealizability_in_FRET_FSESubmission.pdf"
  "nasa-engineering-for-software.pdf"
  "nasa.pptx"
  "claude-code-main.zip"
  "clusterfuzz-testcase-minimized-fuzzdelete-4649128545288192"
  "clusterfuzz-testcase-minimized-fuzzdelete-4649128545288192 (1)"
)

for file in "${DOWNLOAD_FILES[@]}"; do
  rsync_file "$SRC_HOME/Downloads/$file" "$DST_HOME/Downloads/$file" "Downloads/$file"
done

for file in .gitconfig .zshrc .zprofile .zshenv .zshlocal .bashrc .bashlocal .profile .npmrc .actrc; do
  rsync_file "$SRC_HOME/$file" "$STAGED_CONFIG/$file" "stage config $file"
done

echo
echo "Tail migration sync finished at $(date)"
