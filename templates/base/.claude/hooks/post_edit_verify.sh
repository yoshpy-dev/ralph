#!/usr/bin/env sh
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
# Claude Code PostToolUse payloads nest the target path under tool_input.
file_path="$(extract_json_field "$payload" "tool_input.file_path")"

mkdir -p .harness/state
: > .harness/state/needs-verify

# Reset consecutive failure counter on successful tool use
printf '0\n' > .harness/state/tool_failures.count

# Collect every path this tool call touched. Claude Code's Edit/Write/
# MultiEdit payloads carry a single tool_input.file_path. Codex's
# apply_patch payloads carry the patch body in tool_input.command instead
# and have no file_path -- derive the touched paths from the
# "*** Add/Update/Delete File:" and "*** Move to:" envelope lines in that
# patch body.
edited_paths=""
if [ -n "$file_path" ]; then
  edited_paths="$file_path"
elif command -v jq >/dev/null 2>&1; then
  # jq is required for this branch: the sed fallback in lib_json.sh only
  # matches a single-line quoted value and cannot reliably decode a
  # JSON-escaped multi-line patch body (an embedded quote in the diff
  # truncates the match early). Without jq, an apply_patch payload falls
  # through with no derived paths -- the same no-op this hook already had
  # for Codex edits before apply_patch support was added.
  patch_text="$(printf '%s' "$payload" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
  if [ -n "$patch_text" ]; then
    edited_paths="$(printf '%s\n' "$patch_text" | sed -n \
      -e 's/^\*\*\* Add File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Update File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Delete File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Move to: \(.*\)$/\1/p')"
  fi
fi

# Log every derived path, then classify the whole set into one message.
# A code-class path anywhere in the set wins over a doc-class path, which
# wins over the skip list -- this mirrors the single-path precedence the
# case statement below encoded back when only one path could ever be
# derived per call.
class="skip"
if [ -n "$edited_paths" ]; then
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    printf '%s\n' "$p" >> .harness/state/edited-files.log
    case "$p" in
      # Instruction and documentation files
      *"/AGENTS.md"|*"AGENTS.md"|*"/CLAUDE.md"|*"CLAUDE.md"|*"/docs/"*|*"/.claude/rules/"*)
        if [ "$class" != "code" ]; then class="doc"; fi
        ;;
      # Known non-code files: skip verify reminder
      *.md|*.txt|*.json|*.yaml|*.yml|*.toml|*.ini|*.cfg|*.conf|*.lock|*.csv)
        ;;
      # Everything else is treated as code
      *)
        class="code"
        ;;
    esac
  done <<EOF
$edited_paths
EOF
fi

msg=""
case "$class" in
  doc)
    msg="Instruction or documentation files changed. Keep plans, docs, and implementation aligned, and record evidence for behavior changes."
    ;;
  code)
    msg="Code file edited. Run ./scripts/run-verify.sh before claiming done. Save evidence to docs/evidence/."
    ;;
esac

if [ -n "$msg" ]; then
  escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}\n' "$escaped"
fi

exit 0
