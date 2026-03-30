#!/usr/bin/env sh
set -eu

mkdir -p .harness/state

ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
branch="unknown"
if command -v git >/dev/null 2>&1; then
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' 'unknown')"
fi

{
  printf '# Session end summary\n\n'
  printf '- Timestamp: %s\n' "$ts"
  printf '- Branch: %s\n\n' "$branch"
  printf '## Git status\n\n'
  if command -v git >/dev/null 2>&1; then
    git status --short 2>/dev/null || true
  else
    printf 'git not available\n'
  fi
  printf '\n## Recent edited files\n\n'
  if [ -f .harness/state/edited-files.log ]; then
    tail -n 20 .harness/state/edited-files.log
  else
    printf 'No tracked edits in this session.\n'
  fi
} > .harness/state/session-end.md

exit 0
