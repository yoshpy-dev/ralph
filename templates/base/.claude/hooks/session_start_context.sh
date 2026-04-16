#!/usr/bin/env sh
set -eu

mkdir -p .harness/state .harness/logs docs/plans/active docs/plans/archive docs/reports
printf '%s\n' "0" > .harness/state/tool_failures.count

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

msg="Harness reminder: use docs/plans/active for risky or multi-file work, keep AGENTS.md as a map not a manual, and run ./scripts/run-verify.sh before claiming done. Branch: $branch. Active plan: $plan."
escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$escaped"
