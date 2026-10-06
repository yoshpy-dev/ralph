#!/usr/bin/env sh
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
# Real Claude Code PreToolUse payloads nest the Bash tool's argument under
# .tool_input.command, not a top-level .command. On the jq path,
# extract_json_field's dotted-path support (lib_json.sh) resolves this
# directly. The sed fallback (jq absent) matches the leaf key name
# ("command") anywhere in the payload regardless of nesting, and reads
# JSON-escaped quotes inside the value (\" and \\ come back as " and \;
# a newline stays as the two characters \n).
command="$(extract_json_field "$payload" "tool_input.command")"
# Claude Code sends the session's permission mode at the top level
# (default, plan, acceptEdits, auto, dontAsk, bypassPermissions). Only
# bypassPermissions changes anything below: it skips the ask rules. An
# absent key leaves every rule on. Codex's PreToolUse input (0.160.0) also
# carries permission_mode, so the same skip applies under Codex in that
# mode; Codex's handling of an ask decision itself is unverified.
mode="$(extract_json_field "$payload" "permission_mode")"

emit_decision() {
  decision="$1"
  reason="$2"
  escaped="$(printf '%s' "$reason" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"%s","permissionDecisionReason":"%s"}}\n' "$decision" "$escaped"
  exit 0
}

# ── Write targets for the .git and .env ask rules ─────────────────────────
# A case glob cannot tell `ls .git/ 2>&1` from `echo x > .git/x`, so these
# two rules look only at write targets: the word right after a redirection
# operator (>, >>, >|; an fd prefix such as 2> or &> still ends in >) and
# each argument of tee. After `2>&1` the word is `&1`, which never matches.
# POSIX ERE classes only (no \b, \s, \w), so BSD grep (macOS) and GNU grep
# (CI) agree. grep reads the command line by line on the jq path, where
# newlines are real; on the sed fallback a newline is still the two
# characters \n, so a backslash ends a target word and a tee argument list.
# A tab is likewise the two characters \t on the fallback, so a tab between
# a redirection or tee and its target is not seen there (the jq path sees it).
q="'"
# One character of a target word.
word_char="[^[:space:]\"${q};&|()<>\\\\]"
# What may follow a target: whitespace, a quote, a shell operator, a
# backslash, or the end of the line.
word_end="([[:space:]\"${q};&|)<>\\\\]|\$)"
# >, >> or >|, then optional spaces and an optional opening quote.
redirect_lead=">[>|]?[[:space:]]*[\"${q}]?"
# The tee word, then any earlier arguments ending in a space or a quote.
tee_lead="(^|[[:space:];&|(/])tee[[:space:]]([^;&|)\\\\]*[[:space:]\"${q}])?"
# .git itself (a worktree's .git is a file) or a path under it, either as
# the whole word or after a /. .github/ and .gitignore do not match.
git_target="(${word_char}*/)?[.]git(/|${word_end})"
# A word whose last path component starts with .env (.env, .env.local,
# .envrc).
env_target="(${word_char}*/)?[.]env[^[:space:]\"${q};&|()<>\\\\/]*${word_end}"

# command_writes_to <target-regex>: true when a redirection target or a tee
# argument in the command matches <target-regex>.
command_writes_to() {
  printf '%s\n' "$command" | grep -Eq -e "${redirect_lead}$1" -e "${tee_lead}$1"
}

# ── Deny rules: checked first, in every permission mode ───────────────────
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

# ── Ask rules: skipped in bypassPermissions mode ──────────────────────────
# A hook's ask forces a prompt even in bypass mode, so the guard itself
# stays quiet there. The deny rules above have already run.
if [ "$mode" = "bypassPermissions" ]; then
  exit 0
fi

if command_writes_to "$git_target"; then
  emit_decision "ask" "Direct writes into .git require explicit confirmation."
fi
if command_writes_to "$env_target"; then
  emit_decision "ask" "Secret or environment file writes require explicit confirmation."
fi

case "$command" in
  *"rm -rf "*)
    emit_decision "ask" "Recursive delete requires explicit confirmation."
    ;;
  *"gh pr create"*)
    emit_decision "ask" "gh pr create を検出。/pr スキル（Skill tool）経由で実行していますか？ /pr スキルは日本語テンプレート、事前チェック、プランアーカイブを強制します。直接実行は非推奨です。"
    ;;
esac

exit 0
