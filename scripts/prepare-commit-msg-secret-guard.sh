#!/usr/bin/env sh
# prepare-commit-msg-secret-guard.sh - scan non-ff merge history for secrets.
set -eu

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
SCANNER="$REPO_ROOT/scripts/secret-scan.sh"

merge_head_path="$(git rev-parse --git-path MERGE_HEAD 2>/dev/null || true)"
if [ -z "$merge_head_path" ] || [ ! -s "$merge_head_path" ]; then
  exit 0
fi

while IFS= read -r merge_head || [ -n "$merge_head" ]; do
  [ -n "$merge_head" ] || continue
  "$SCANNER" --range "HEAD..$merge_head"
done < "$merge_head_path"
