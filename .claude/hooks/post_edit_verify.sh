#!/usr/bin/env sh
set -eu

payload="$(cat | tr '\n' ' ')"
file_path="$(printf '%s' "$payload" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

mkdir -p .harness/state
: > .harness/state/needs-verify
if [ -n "$file_path" ]; then
  printf '%s\n' "$file_path" >> .harness/state/edited-files.log
fi

case "$file_path" in
  *"/AGENTS.md"|*"AGENTS.md"|*"/CLAUDE.md"|*"CLAUDE.md"|*"/docs/"*|*"/.claude/rules/"*)
    msg="Instruction or documentation files changed. Keep plans, docs, and implementation aligned, and record evidence for behavior changes."
    escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
    printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}\n' "$escaped"
    ;;
esac

exit 0
