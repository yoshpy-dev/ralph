#!/usr/bin/env sh
set -eu

mkdir -p .harness/state

ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
branch="unknown"
if command -v git >/dev/null 2>&1; then
  branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' 'unknown')"
fi

{
  printf '%s\n\n' '# Session end summary'
  printf '%s\n' "- Timestamp: $ts"
  printf '%s\n\n' "- Branch: $branch"
  printf '%s\n\n' '## Git status'
  if command -v git >/dev/null 2>&1; then
    git status --short 2>/dev/null || true
  else
    printf '%s\n' 'git not available'
  fi
  printf '\n%s\n\n' '## Recent edited files'
  if [ -f .harness/state/edited-files.log ]; then
    tail -n 20 .harness/state/edited-files.log
  else
    printf '%s\n' 'No tracked edits in this session.'
  fi
} > .harness/state/session-end.md

# WIP commit on feature branches before session end
if command -v git >/dev/null 2>&1; then
  current_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' 'unknown')"
  case "$current_branch" in
    main|master|unknown) ;;
    *)
      if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        git add -A 2>/dev/null || true
        git commit -m "wip: checkpoint before session end" 2>/dev/null || true
      fi
      ;;
  esac
fi

exit 0
