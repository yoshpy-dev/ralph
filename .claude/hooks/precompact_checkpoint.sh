#!/usr/bin/env sh
set -eu

mkdir -p .harness/state

ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
branch="unknown"
if command -v git >/dev/null 2>&1; then
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' 'unknown')"
fi

plan="none"
if [ -d docs/plans/active ]; then
  plan="$(find docs/plans/active -maxdepth 1 -type f -name '*.md' | sort | tail -n 1)"
  if [ -z "$plan" ]; then
    plan="none"
  fi
fi

{
  printf '# Pre-compact checkpoint\n\n'
  printf '- Timestamp: %s\n' "$ts"
  printf '- Branch: %s\n' "$branch"
  printf '- Active plan: %s\n\n' "$plan"
  printf '## Git status\n\n'
  if command -v git >/dev/null 2>&1; then
    git status --short 2>/dev/null || true
  else
    printf 'git not available\n'
  fi
} > .harness/state/precompact-checkpoint.md

exit 0
