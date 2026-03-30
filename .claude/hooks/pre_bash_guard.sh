#!/usr/bin/env sh
set -eu

payload="$(cat | tr '\n' ' ')"
command="$(printf '%s' "$payload" | sed -n 's/.*"command"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

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
esac

exit 0
