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
    emit_decision "ask" "gh pr create を検出。/pr スキル（Skill tool）経由で実行していますか？ /pr スキルは日本語テンプレート、事前チェック、プランアーカイブを強制します。直接実行は非推奨です。"
    ;;
esac

# Layer: detect command substitution inside double-quoted git commit -m messages
# Prevents shell expansion of backticks or $() that could leak env vars / secrets
case "$command" in
  *"git commit"*"-m "*)
    # Extract the part after -m
    msg_part="${command#*-m }"
    # Check if message uses double quotes containing backticks or $(...)
    case "$msg_part" in
      '"'*'`'*|'"'*'$('*)
        emit_decision "deny" "コミットメッセージのダブルクォート内にバッククォートまたは \$() を検出しました。シェルのコマンド置換として解釈され、環境変数やシークレットが漏洩する恐れがあります。代わりにシングルクォートまたは HEREDOC (<<'EOF') を使用してください。"
        ;;
    esac
    ;;
esac

exit 0
