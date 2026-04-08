#!/usr/bin/env sh
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
command="$(extract_json_field "$payload" "command")"

emit_decision() {
  decision="$1"
  reason="$2"
  escaped="$(printf '%s' "$reason" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"%s","permissionDecisionReason":"%s"}}\n' "$decision" "$escaped"
  exit 0
}

case "$command" in
  *"sudo "*)
    emit_decision "deny" "Avoid sudo inside the harness. Use project-local commands or escalate to a human only if truly necessary."
    ;;
  *"git push --force"*|*"git push -f"*)
    emit_decision "deny" "Force push is blocked by the scaffold."
    ;;
  *"git reset --hard"*)
    emit_decision "deny" "Hard reset is blocked by the scaffold."
    ;;
  *".git/"*">"*|*"> .git"*|*"tee .git"*)
    emit_decision "ask" "Direct writes into .git require explicit confirmation."
    ;;
  *".env"*">"*|*"> .env"*|*"tee .env"*)
    emit_decision "ask" "Secret or environment file writes require explicit confirmation."
    ;;
  *"rm -rf "*)
    emit_decision "ask" "Recursive delete requires explicit confirmation."
    ;;
  *"gh pr create"*)
    emit_decision "deny" "Do not call 'gh pr create' directly. Use the /pr skill (Skill tool) instead — it enforces the Japanese PR template, pre-checks, and plan archiving."
    ;;
esac

exit 0
