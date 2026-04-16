#!/usr/bin/env sh
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
file_path="$(extract_json_field "$payload" "file_path")"

mkdir -p .harness/state
: > .harness/state/needs-verify

# Reset consecutive failure counter on successful tool use
printf '0\n' > .harness/state/tool_failures.count
if [ -n "$file_path" ]; then
  printf '%s\n' "$file_path" >> .harness/state/edited-files.log
fi

msg=""
case "$file_path" in
  # Instruction and documentation files
  *"/AGENTS.md"|*"AGENTS.md"|*"/CLAUDE.md"|*"CLAUDE.md"|*"/docs/"*|*"/.claude/rules/"*)
    msg="Instruction or documentation files changed. Keep plans, docs, and implementation aligned, and record evidence for behavior changes."
    ;;
  # Known non-code files: skip verify reminder
  *.md|*.txt|*.json|*.yaml|*.yml|*.toml|*.ini|*.cfg|*.conf|*.lock|*.csv|"")
    ;;
  # Everything else is treated as code
  *)
    msg="Code file edited. Run ./scripts/run-verify.sh before claiming done. Save evidence to docs/evidence/."
    ;;
esac

if [ -n "$msg" ]; then
  escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}\n' "$escaped"
fi

exit 0
