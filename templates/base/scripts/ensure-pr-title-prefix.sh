#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BRANCH_NAME="${SCRIPT_DIR}/branch-name.sh"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/ensure-pr-title-prefix.sh [<pr-url-or-number-or-branch>]

If no target is provided, the current git branch is used.
The PR title must start with the branch type prefix, e.g. feat/foo -> feat: ...
USAGE
}

target="${1:-}"
if [ "${target:-}" = "-h" ] || [ "${target:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ -z "$target" ]; then
  target="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
fi

if [ -z "$target" ]; then
  printf 'ensure-pr-title-prefix: no PR target provided and current branch could not be detected\n' >&2
  exit 1
fi

title="$(gh pr view "$target" --json title --jq '.title')"
head_branch="$(gh pr view "$target" --json headRefName --jq '.headRefName')"
title_type="$("$BRANCH_NAME" type "$head_branch")"
prefix="${title_type}:"

case "$title" in
  "$prefix"|"${prefix} "*)
    printf 'ensure-pr-title-prefix: PR title already has prefix %s %s\n' "$prefix" "$title" >&2
    exit 0
    ;;
esac

allowed_re="$("$BRANCH_NAME" allowed-types | tr ' ' '|')"
title_body="$(printf '%s' "$title" | sed -E "s/^(${allowed_re}):[[:space:]]*//")"
if [ -n "$title_body" ]; then
  new_title="${prefix} ${title_body}"
else
  new_title="$prefix"
fi

printf 'ensure-pr-title-prefix: updating PR title: %s\n' "$new_title" >&2
gh pr edit "$target" --title "$new_title" >/dev/null

title_after="$(gh pr view "$target" --json title --jq '.title')"
if [ "$title_after" != "$new_title" ]; then
  printf 'ensure-pr-title-prefix: PR title still missing expected prefix after update\n' >&2
  printf '  expected: %s\n' "$new_title" >&2
  printf '  actual:   %s\n' "$title_after" >&2
  exit 1
fi

printf 'ensure-pr-title-prefix: PR title verified: %s\n' "$new_title" >&2
