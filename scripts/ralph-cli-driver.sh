#!/usr/bin/env sh
# ralph-cli-driver.sh — driver-agnostic agent invocation for Ralph Loop.
#
# Exposes one function: run_agent <prompt_file> <log_file> [extra_args]
# Branches on $RALPH_LOOP_DRIVER (claude|codex) to invoke the right CLI and
# emits two artefacts the caller can read uniformly:
#   <log_file>       — the agent's last/result message as plain text
#   <log_file>.json  — { "result": "...", "session_id": "<id>|null" }
#
# Phase 2 / issue #44: replaces the inline `run_claude` previously in
# ralph-pipeline.sh so the same caller works for either CLI. Sidecar files
# (.agent-signal, .self-review-result, etc.) remain the authoritative
# structured signal channel — that contract is unchanged across drivers.
#
# Globals consumed (set by the caller, ralph-pipeline.sh):
#   RALPH_LOOP_DRIVER          claude|codex
#   RALPH_MODEL                Claude model id (driver=claude only)
#   RALPH_EFFORT               Claude effort tier (driver=claude only)
#   RALPH_PERMISSION_MODE      Claude permission mode (driver=claude only)
#   RALPH_CODEX_SANDBOX        Codex sandbox mode (driver=codex only)
#   RALPH_CODEX_APPROVAL_POLICY Codex approval policy (driver=codex only)
#   JSON_OUTPUT_SUPPORTED      1 if `claude -p --output-format json` works
#   DRY_RUN                    1 to skip CLI invocation entirely

# pick_reviewer — return the *opposite* CLI of the active driver, used by
# cross-review to keep the cross-model quality gate even when codex drives
# the Inner Loop. Prints "codex" or "claude" on stdout. Defined here so the
# inversion logic lives next to the dispatcher and is testable in isolation.
pick_reviewer() {
  case "${RALPH_LOOP_DRIVER:-claude}" in
    claude) printf 'codex\n'  ;;
    codex)  printf 'claude\n' ;;
    *)      printf 'codex\n'  ;;  # safe fallback; validate_loop_driver should have caught this
  esac
}

# run_agent — dispatch one agent turn. Writes <log_file> (text) and
# <log_file>.json (thin metadata) regardless of which driver runs.
run_agent() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"

  if [ "${DRY_RUN:-0}" -eq 1 ]; then
    printf '[dry-run] %s: would run with %s\n' "${RALPH_LOOP_DRIVER:-claude}" "$_prompt_file" > "$_log_file"
    printf '{"result":"[dry-run] iteration output","session_id":null}' > "${_log_file}.json"
    return 0
  fi

  case "${RALPH_LOOP_DRIVER:-claude}" in
    claude) _run_agent_claude "$_prompt_file" "$_log_file" "$_extra_args" ;;
    codex)  _run_agent_codex  "$_prompt_file" "$_log_file" "$_extra_args" ;;
    *)
      printf 'ralph-cli-driver: unknown RALPH_LOOP_DRIVER %s\n' "$RALPH_LOOP_DRIVER" >&2
      return 1
      ;;
  esac
}

# _run_agent_claude — preserves the prior `run_claude` behaviour exactly so
# existing pipelines keep their JSON-mode + text-fallback semantics.
_run_agent_claude() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"

  if [ "${JSON_OUTPUT_SUPPORTED:-0}" -eq 1 ]; then
    # shellcheck disable=SC2086
    claude -p --model "$RALPH_MODEL" --effort "$RALPH_EFFORT" \
      --permission-mode "$RALPH_PERMISSION_MODE" \
      --output-format json $_extra_args \
      < "$_prompt_file" > "${_log_file}.json" 2>"${_log_file}.stderr" || true
    if jq -e '.result' "${_log_file}.json" >/dev/null 2>&1; then
      jq -r '.result // empty' "${_log_file}.json" > "$_log_file"
    else
      printf 'Warning: JSON parse failed for %s.json, using raw output\n' "$_log_file" >&2
      cp "${_log_file}.json" "$_log_file"
    fi
    cat "$_log_file"
  else
    # shellcheck disable=SC2086
    claude -p --model "$RALPH_MODEL" --effort "$RALPH_EFFORT" \
      --permission-mode "$RALPH_PERMISSION_MODE" \
      --output-format text $_extra_args \
      < "$_prompt_file" 2>&1 | tee "$_log_file"
    : > "${_log_file}.json"
  fi
}

# _run_agent_codex — invokes `codex exec` with sandbox + approval policy and
# captures the agent's final message via `--output-last-message`. Synthesises
# the thin JSON sidecar so callers do not need to special-case the driver.
_run_agent_codex() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"

  _last_file="${_log_file}.last"
  rm -f "$_last_file"

  # `-` as the prompt argument makes codex read stdin (per `codex exec --help`),
  # so the prompt file is fed through the same channel claude -p uses.
  # shellcheck disable=SC2086
  codex exec \
    -s "$RALPH_CODEX_SANDBOX" \
    -c "approval_policy=\"$RALPH_CODEX_APPROVAL_POLICY\"" \
    --output-last-message "$_last_file" \
    --skip-git-repo-check \
    $_extra_args \
    - < "$_prompt_file" \
    > "${_log_file}.stdout" 2>"${_log_file}.stderr" || true

  if [ -f "$_last_file" ]; then
    cp "$_last_file" "$_log_file"
  else
    printf 'Warning: codex did not write %s; falling back to stdout\n' "$_last_file" >&2
    cp "${_log_file}.stdout" "$_log_file" 2>/dev/null || : > "$_log_file"
  fi

  # Synthesise driver-neutral metadata. Codex doesn't expose a Claude-shaped
  # session_id, so leave it null — the caller only resumes when non-null.
  if command -v jq >/dev/null 2>&1; then
    jq -Rs '{result: ., session_id: null}' < "$_log_file" > "${_log_file}.json"
  else
    # Minimal fallback when jq is missing — keeps callers parsing without crashing.
    printf '{"result":"see %s","session_id":null}' "$_log_file" > "${_log_file}.json"
  fi

  cat "$_log_file"
}
