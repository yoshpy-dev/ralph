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
  printf '%s\n\n' '# Pre-compact checkpoint'
  printf '%s\n' "- Timestamp: $ts"
  printf '%s\n' "- Branch: $branch"
  printf '%s\n\n' "- Active plan: $plan"
  printf '%s\n\n' '## Git status'
  if command -v git >/dev/null 2>&1; then
    git status --short 2>/dev/null || true
  else
    printf '%s\n' 'git not available'
  fi
} > .harness/state/precompact-checkpoint.md

# WIP commit on feature branches before context compaction.
# RALPH_WIP_CHECKPOINT=off skips only the commit: ralph sets it in the herdr
# panes it creates for org seats, whose compaction must not commit (see
# .claude/rules/ralph/git-commit-strategy.md).
if command -v git >/dev/null 2>&1; then
  current_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf '%s' 'unknown')"
  case "$current_branch" in
    main|master|unknown) ;;
    *)
      if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
        if [ "${RALPH_WIP_CHECKPOINT:-}" = "off" ]; then
          printf '\n%s\n' '- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)' >> .harness/state/precompact-checkpoint.md
        else
          git add -A 2>/dev/null || true
          if ! git commit -m 'wip: checkpoint before context compaction' 2>/dev/null; then
            mkdir -p .harness/logs
            printf '%s WIP commit failed (precompact on branch %s)\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$current_branch" >> .harness/logs/hook-failures.log
          fi
        fi
      fi
      ;;
  esac
fi

exit 0
