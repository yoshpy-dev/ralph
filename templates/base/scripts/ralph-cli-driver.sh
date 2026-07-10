#!/usr/bin/env sh
# ralph-cli-driver.sh — driver-agnostic agent invocation for Ralph Loop.
#
# Exposes these functions:
#   run_agent <prompt_file> <log_file> [extra_args] [model]
#   resolve_phase_model <phase> [cycle]
#   write_model_receipt <phase> <cycle> <requested_model> <reason>
#   count_triage_findings <triage_report_path> <category>
#   pick_reviewer
#
# run_agent branches on $RALPH_LOOP_DRIVER (claude|codex) to invoke the right
# CLI and emits two artefacts the caller can read uniformly:
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
#   RALPH_MODEL                Claude model id fallback (driver=claude only)
#   RALPH_EFFORT               Claude effort tier (driver=claude only)
#   RALPH_PERMISSION_MODE      Claude permission mode (driver=claude only)
#   RALPH_CODEX_SANDBOX        Codex sandbox mode (driver=codex only)
#   RALPH_CODEX_APPROVAL_POLICY Codex approval policy (driver=codex only)
#   JSON_OUTPUT_SUPPORTED      1 if `claude -p --output-format json` works
#   DRY_RUN                    1 to skip CLI invocation entirely
#   RALPH_FORCE_MODEL          when non-empty, overrides all phase models
#   RALPH_IMPLEMENT_MODEL      per-phase model (see ralph-config.sh)
#   RALPH_SELF_REVIEW_MODEL    per-phase model
#   RALPH_VERIFY_MODEL         per-phase model
#   RALPH_TEST_MODEL           per-phase model
#   RALPH_SYNC_DOCS_MODEL      per-phase model
#   RALPH_PR_MODEL             per-phase model
#   RALPH_PROBE_MODEL          per-phase model
#   RALPH_ESCALATION_MODEL     used for implement when cycle >= 2

# resolve_phase_model <phase> [cycle] — print the routed model for a phase.
#
# Precedence: RALPH_FORCE_MODEL > cycle-based escalation > per-phase var > fallback.
# phase values: implement, self_review, verify, test, sync_docs, pr, probe.
# cycle: numeric string; empty or missing is treated as 1.
# Unknown phase falls back to $RALPH_MODEL.
# Pure function: no side effects, no CLI invocation — safe to call in DRY_RUN.
resolve_phase_model() {
  _rpm_phase="${1:-}"
  _rpm_cycle="${2:-1}"

  # RALPH_FORCE_MODEL wins over everything when set.
  if [ -n "${RALPH_FORCE_MODEL:-}" ]; then
    printf '%s\n' "$RALPH_FORCE_MODEL"
    return 0
  fi

  # Cycle-based escalation: implement seat upgrades on cycle >= 2.
  if [ "$_rpm_phase" = "implement" ]; then
    # Treat empty/non-numeric cycle as 1 (no escalation).
    case "$_rpm_cycle" in
      ''|*[!0-9]*) _rpm_cycle=1 ;;
    esac
    if [ "$_rpm_cycle" -ge 2 ] 2>/dev/null; then
      printf '%s\n' "${RALPH_ESCALATION_MODEL:-opus}"
      return 0
    fi
  fi

  # Per-phase routing.
  case "$_rpm_phase" in
    implement)   printf '%s\n' "${RALPH_IMPLEMENT_MODEL:-sonnet}"   ;;
    self_review) printf '%s\n' "${RALPH_SELF_REVIEW_MODEL:-opus}"   ;;
    verify)      printf '%s\n' "${RALPH_VERIFY_MODEL:-sonnet}"      ;;
    test)        printf '%s\n' "${RALPH_TEST_MODEL:-sonnet}"        ;;
    sync_docs)   printf '%s\n' "${RALPH_SYNC_DOCS_MODEL:-sonnet}"   ;;
    pr)          printf '%s\n' "${RALPH_PR_MODEL:-sonnet}"          ;;
    probe)       printf '%s\n' "${RALPH_PROBE_MODEL:-haiku}"        ;;
    *)           printf '%s\n' "${RALPH_MODEL:-opus}"               ;;
  esac
}

# write_model_receipt <phase> <cycle> <requested_model> <reason>
#
# Appends one JSONL line to .harness/state/pipeline/model-receipts.jsonl.
# Fields: ts, phase, cycle, driver, requested_model, effective_model, honored,
#         effort, reason.
# driver=claude → effective_model=requested_model, honored=true.
# driver=codex  → effective_model="codex-default",  honored=false
#                 (codex exec does not receive a model argument).
# Safe to call in DRY_RUN mode (no CLI involved; receipt is still written).
write_model_receipt() {
  _wmr_phase="$1"
  _wmr_cycle="$2"
  _wmr_requested="$3"
  _wmr_reason="$4"
  _wmr_driver="${RALPH_LOOP_DRIVER:-claude}"
  _wmr_effort="${RALPH_EFFORT:-high}"
  _wmr_ts="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || printf 'unknown')"
  _wmr_dir=".harness/state/pipeline"
  _wmr_file="${_wmr_dir}/model-receipts.jsonl"

  mkdir -p "$_wmr_dir"

  if [ "$_wmr_driver" = "codex" ]; then
    _wmr_effective="codex-default"
    _wmr_honored="false"
  else
    _wmr_effective="$_wmr_requested"
    _wmr_honored="true"
  fi

  if command -v jq >/dev/null 2>&1; then
    jq -n \
      --arg ts              "$_wmr_ts" \
      --arg phase           "$_wmr_phase" \
      --arg cycle           "$_wmr_cycle" \
      --arg driver          "$_wmr_driver" \
      --arg requested_model "$_wmr_requested" \
      --arg effective_model "$_wmr_effective" \
      --argjson honored     "$_wmr_honored" \
      --arg effort          "$_wmr_effort" \
      --arg reason          "$_wmr_reason" \
      '{ts:$ts,phase:$phase,cycle:$cycle,driver:$driver,requested_model:$requested_model,effective_model:$effective_model,honored:$honored,effort:$effort,reason:$reason}' \
      >> "$_wmr_file"
  else
    # Minimal printf fallback when jq is missing (same pattern as _run_agent_codex).
    printf '{"ts":"%s","phase":"%s","cycle":"%s","driver":"%s","requested_model":"%s","effective_model":"%s","honored":%s,"effort":"%s","reason":"%s"}\n' \
      "$_wmr_ts" "$_wmr_phase" "$_wmr_cycle" "$_wmr_driver" \
      "$_wmr_requested" "$_wmr_effective" "$_wmr_honored" \
      "$_wmr_effort" "$_wmr_reason" \
      >> "$_wmr_file"
  fi
}

# count_triage_findings — print the count for one triage category from a
# cross-review-triage report. Prefers the canonical summary header line
# (`After triage: ACTION_REQUIRED=N, ...`) and falls back to counting the
# `|` table rows under each `## <CATEGORY>` heading. Lives here so the
# pipeline parser is testable in isolation: the previous in-line
# `grep -c '<CATEGORY>' "$file"` overcounted the literal headings (#44
# cross-review P1).
#
# Args: $1 = triage report path
#       $2 = category (ACTION_REQUIRED | WORTH_CONSIDERING | DISMISSED)
count_triage_findings() {
  _file="$1"
  _category="$2"
  if [ ! -s "$_file" ]; then
    printf '0\n'
    return 0
  fi
  # Anchor the summary match at the start of a line so a reviewer's prose
  # ("the diff includes ACTION_REQUIRED=2 as an example") cannot trigger the
  # summary path. The triage template (docs/reports/templates/cross-review-triage-report.md)
  # writes the canonical line as `- After triage: ACTION_REQUIRED=N, ...`.
  _summary="$(grep -m1 -E '^[- ]*After triage: ACTION_REQUIRED=[0-9]+' "$_file" 2>/dev/null || true)"
  if [ -n "$_summary" ]; then
    _n="$(printf '%s' "$_summary" | grep -oE "${_category}=[0-9]+" | head -1 | cut -d= -f2)"
    printf '%s\n' "${_n:-0}"
    return 0
  fi
  awk -v cat="## ${_category}" '
    $0 == cat { f = 1; next }
    /^## / { f = 0 }
    f && /^\|/ && !/^\| *# / && !/^\| *-+/ { n++ }
    END { print n+0 }
  ' "$_file" 2>/dev/null || printf '0\n'
}

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
#
# Usage: run_agent <prompt_file> <log_file> [extra_args] [model]
#   model (4th arg): model alias to pass to claude --model.
#                    Empty or omitted → falls back to $RALPH_MODEL.
#                    Ignored silently by the codex driver.
# Backward compatible: existing 2- and 3-arg call sites are unchanged.
run_agent() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"
  _model_arg="${4:-}"

  if [ "${DRY_RUN:-0}" -eq 1 ]; then
    printf '[dry-run] %s: would run with %s\n' "${RALPH_LOOP_DRIVER:-claude}" "$_prompt_file" > "$_log_file"
    printf '{"result":"[dry-run] iteration output","session_id":null}' > "${_log_file}.json"
    return 0
  fi

  case "${RALPH_LOOP_DRIVER:-claude}" in
    claude) _run_agent_claude "$_prompt_file" "$_log_file" "$_extra_args" "$_model_arg" ;;
    codex)  _run_agent_codex  "$_prompt_file" "$_log_file" "$_extra_args" ;;
    *)
      printf 'ralph-cli-driver: unknown RALPH_LOOP_DRIVER %s\n' "$RALPH_LOOP_DRIVER" >&2
      return 1
      ;;
  esac
}

# _run_agent_claude — preserves the prior `run_claude` behaviour exactly so
# existing pipelines keep their JSON-mode + text-fallback semantics.
# 4th arg _model: model alias for --model; empty → falls back to $RALPH_MODEL.
_run_agent_claude() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"
  _model="${4:-}"
  _effective_model="${_model:-$RALPH_MODEL}"

  if [ "${JSON_OUTPUT_SUPPORTED:-0}" -eq 1 ]; then
    # shellcheck disable=SC2086
    claude -p --model "$_effective_model" --effort "$RALPH_EFFORT" \
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
    claude -p --model "$_effective_model" --effort "$RALPH_EFFORT" \
      --permission-mode "$RALPH_PERMISSION_MODE" \
      --output-format text $_extra_args \
      < "$_prompt_file" 2>&1 | tee "$_log_file"
    : > "${_log_file}.json"
  fi
}

# _run_agent_codex — invokes `codex exec` with sandbox + approval policy and
# captures the agent's final message via `--output-last-message`. Synthesises
# the thin JSON sidecar so callers do not need to special-case the driver.
# Note: the optional 4th model arg is intentionally ignored here — codex model
# selection lives in .codex/config.toml and cannot be overridden via CLI flag.
# This is a documented known gap; write_model_receipt records honored=false.
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
    # Codex did not produce a final-message file. Surface stdout AND stderr
    # in <log_file> so the operator (and the surrounding pipeline parser)
    # actually sees the failure mode — without this, an empty log + a single
    # warning line was the only evidence and the real error stayed buried in
    # ${_log_file}.stderr.
    printf 'Warning: codex did not write %s; falling back to stdout/stderr\n' "$_last_file" >&2
    {
      printf '=== codex stdout (no --output-last-message file produced) ===\n'
      cat "${_log_file}.stdout" 2>/dev/null || true
      printf '\n=== codex stderr ===\n'
      cat "${_log_file}.stderr" 2>/dev/null || true
    } > "$_log_file"
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
