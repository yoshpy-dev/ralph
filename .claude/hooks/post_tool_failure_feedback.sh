#!/usr/bin/env sh
set -eu

mkdir -p .harness/state
count_file=".harness/state/tool_failures.count"

count="0"
if [ -f "$count_file" ]; then
  count="$(cat "$count_file" 2>/dev/null || printf '%s' '0')"
fi

case "$count" in
  ''|*[!0-9]*) count="0" ;;
esac

count=$((count + 1))
printf '%s\n' "$count" > "$count_file"

if [ "$count" -ge 3 ]; then
  msg="There have been multiple tool failures. Stop repeating the same move. Shrink scope, inspect evidence, update the plan, or switch to reviewer or verifier agents."
  escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUseFailure","additionalContext":"%s"}}\n' "$escaped"
fi

exit 0
