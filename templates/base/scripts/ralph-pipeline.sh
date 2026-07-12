#!/usr/bin/env bash
set -euo pipefail

# Ralph Pipeline orchestrator — full autonomous development pipeline
# Inner Loop: implement → self-review → verify → test (repeat on failure)
# Outer Loop: sync-docs → cross-review (repeat on ACTION_REQUIRED) → PR
#
# State lives in .harness/state/pipeline/
# Requires: jq, git, and one of {claude, codex} binaries (selected by
#           RALPH_LOOP_DRIVER, default claude). cross-review prefers the
#           other reviewer binary when available.

# Use BASH_SOURCE[0] so SCRIPT_DIR resolves correctly when the script is sourced by tests
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/ralph-config.sh"
. "${SCRIPT_DIR}/ralph-common.sh"
. "${SCRIPT_DIR}/ralph-cli-driver.sh"

PIPELINE_DIR=".harness/state/pipeline"
EVIDENCE_DIR="docs/evidence"
REPORTS_DIR="docs/reports"
MAX_ITERATIONS="$RALPH_MAX_ITERATIONS"
MAX_INNER_CYCLES="$RALPH_MAX_INNER_CYCLES"
MAX_OUTER_CYCLES="$RALPH_MAX_OUTER_CYCLES"
MAX_REPAIR_ATTEMPTS="$RALPH_MAX_REPAIR_ATTEMPTS"
DRY_RUN=0
PREFLIGHT_ONLY=0
RESUME=0
SKIP_PR=0
FIX_ALL=0
JSON_OUTPUT_SUPPORTED=0

usage() {
  cat <<'USAGE'
Usage: ralph-pipeline.sh [OPTIONS]

Full autonomous development pipeline with Inner/Outer Loop architecture.

Options:
  --max-iterations N       Total iteration cap across all cycles (default: 20)
  --max-inner-cycles N     Max Inner Loop cycles before escalation (default: 10)
  --max-outer-cycles N     Max Outer Loop regressions before escalation (default: 2)
  --max-repair-attempts N  Max fix attempts per failing test (default: 5)
  --preflight              Run capability probe only, then exit
  --resume                 Resume from existing checkpoint.json
  --skip-pr                Skip PR creation phase in Outer Loop
  --fix-all                Fix ALL findings (any self-review findings override COMPLETE,
                           WORTH_CONSIDERING treated as ACTION_REQUIRED)
  --dry-run                Print what would run without executing claude
  -h, --help               Show this help
USAGE
  exit 0
}

while [ $# -gt 0 ]; do
  case "$1" in
    --max-iterations)     shift; MAX_ITERATIONS="${1:?requires a number}"; validate_numeric "--max-iterations" "$MAX_ITERATIONS" ;;
    --max-inner-cycles)   shift; MAX_INNER_CYCLES="${1:?requires a number}"; validate_numeric "--max-inner-cycles" "$MAX_INNER_CYCLES" ;;
    --max-outer-cycles)   shift; MAX_OUTER_CYCLES="${1:?requires a number}"; validate_numeric "--max-outer-cycles" "$MAX_OUTER_CYCLES" ;;
    --max-repair-attempts) shift; MAX_REPAIR_ATTEMPTS="${1:?requires a number}"; validate_numeric "--max-repair-attempts" "$MAX_REPAIR_ATTEMPTS" ;;
    --skip-pr)            SKIP_PR=1 ;;
    --fix-all)            FIX_ALL=1 ;;
    --preflight)          PREFLIGHT_ONLY=1 ;;
    --resume)             RESUME=1 ;;
    --dry-run)            DRY_RUN=1 ;;
    -h|--help)            usage ;;
    *)                    echo "Unknown option: $1"; usage ;;
  esac
  shift
done

# Validate all numeric config (catches bad env vars even without CLI args)
validate_all_numeric

# Per-slice Ralph Loop pipelines use the fast changed-language verifier/tester.
# The integration pipeline (`--skip-pr --fix-all`) is the broad merged-branch
# gate and keeps full repository language coverage unless explicitly overridden.
if [ -z "${RALPH_VERIFY_SCOPE:-}" ]; then
  if [ "$SKIP_PR" -eq 1 ] && [ "$FIX_ALL" -eq 1 ]; then
    RALPH_VERIFY_SCOPE="full"
  else
    RALPH_VERIFY_SCOPE="changed"
  fi
  export RALPH_VERIFY_SCOPE
fi

# ═══════════════════════════════════════════════════════════════════
# Cleanup trap
# ═══════════════════════════════════════════════════════════════════

_pipeline_cleanup() {
  rm -f "${PIPELINE_DIR}/.impl-prompt.md" "${PIPELINE_DIR}/.review-prompt.md" \
        "${PIPELINE_DIR}/.verify-prompt.md" "${PIPELINE_DIR}/.test-prompt.md" \
        "${PIPELINE_DIR}/.docs-prompt.md" "${PIPELINE_DIR}/.pr-prompt.md" \
        "${PIPELINE_DIR}/.preflight-probe.txt" "${PIPELINE_DIR}/.json-probe.txt" 2>/dev/null || true
}
trap _pipeline_cleanup EXIT

# ═══════════════════════════════════════════════════════════════════
# Utility functions
# ═══════════════════════════════════════════════════════════════════

# ts, ts_file, log, log_error are provided by ralph-common.sh (sourced above).

# Read a field from checkpoint.json using jq
ckpt_read() {
  _field="$1"
  if [ -f "${PIPELINE_DIR}/checkpoint.json" ]; then
    jq -r ".${_field} // empty" "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null || true
  fi
}

# Update checkpoint.json fields using jq
ckpt_update() {
  _tmp="${PIPELINE_DIR}/checkpoint.tmp.json"
  if [ ! -f "${PIPELINE_DIR}/checkpoint.json" ]; then
    echo '{}' > "${PIPELINE_DIR}/checkpoint.json"
  fi
  # All arguments are forwarded to jq (filter + optional --arg flags)
  jq "$@" "${PIPELINE_DIR}/checkpoint.json" > "$_tmp" && mv "$_tmp" "${PIPELINE_DIR}/checkpoint.json"
}

# Append a phase transition event
ckpt_transition() {
  _from="$1"
  _to="$2"
  _reason="${3:-}"
  _entry="{\"from\": \"${_from}\", \"to\": \"${_to}\", \"timestamp\": \"$(ts)\""
  if [ -n "$_reason" ]; then
    _entry="${_entry}, \"reason\": \"${_reason}\""
  fi
  _entry="${_entry}}"
  ckpt_update ".phase_transitions += [${_entry}]"
}

# Append a pipeline execution event to the report
report_event() {
  _event_type="$1"
  _details="$2"
  _report="${PIPELINE_DIR}/execution-events.jsonl"
  printf '{"timestamp":"%s","event":"%s","details":%s}\n' "$(ts)" "$_event_type" "$_details" >> "$_report"
}

# emit_insight_event — fire-and-forget wrapper around insights-append.sh.
#
# Usage: emit_insight_event <phase> <cycle> <verdict> \
#          [--critical N] [--high N] [--medium N] [--low N] \
#          [--action-required N] [--worth-considering N] [--dismissed N] \
#          [--requested-model M] [--effective-model M] [--honored bool]
#
# All routing and identity fields (run_id, slug, flow, driver) are
# injected from the pipeline globals set during main() init.
# The call is guarded with || log_warn so it is provably non-fatal.
emit_insight_event() {
  _eie_phase="$1"
  _eie_cycle="$2"
  _eie_verdict="$3"
  shift 3

  # Locate insights-append.sh relative to this script.
  _eie_appender="${SCRIPT_DIR}/insights-append.sh"
  if [ ! -x "$_eie_appender" ]; then
    log "Warning: insights-append.sh not found or not executable — skipping insight event"
    return 0
  fi

  # Build base args. PIPELINE_RUN_ID and PIPELINE_SLUG are set in main().
  _eie_args="--slug ${PIPELINE_SLUG:-unknown} --flow loop --phase ${_eie_phase} --verdict ${_eie_verdict} --source pipeline --cycle ${_eie_cycle}"
  if [ -n "${PIPELINE_RUN_ID:-}" ]; then
    _eie_args="${_eie_args} --run-id ${PIPELINE_RUN_ID}"
  fi
  _eie_args="${_eie_args} --driver ${RALPH_LOOP_DRIVER:-claude}"

  # Pass remaining args (count flags and routing overrides) through.
  # shellcheck disable=SC2086  # word splitting intentional for arg list
  bash "$_eie_appender" ${_eie_args} "$@" || log "Warning: insight event append failed (non-fatal)"
}

# log_warn — alias used in emit_insight_event guard messages
log_warn() { log "Warning: $*"; }

# run_agent is provided by ralph-cli-driver.sh — it dispatches to claude or
# codex based on RALPH_LOOP_DRIVER and writes the same <log>/<log>.json
# artefacts the rest of this script consumes.

# Check for uncommitted changes and warn
check_uncommitted() {
  if command -v git >/dev/null 2>&1; then
    _uncommitted="$(git status --porcelain 2>/dev/null || true)"
    if [ -n "$_uncommitted" ]; then
      log "Warning: uncommitted changes detected"
      return 1
    fi
  fi
  return 0
}

# Hook parity check: run safety checks that hooks would normally enforce
run_hook_parity() {
  _parity_result="${EVIDENCE_DIR}/hook-parity-checklist.json"
  _all_pass=true
  _checks="[]"

  # Check 1: Secret leak detection in recent commits
  _secret_check="pass"
  if [ -x ./scripts/commit-msg-guard.sh ]; then
    _last_msg="$(git log -1 --format='%B' 2>/dev/null || true)"
    if [ -n "$_last_msg" ]; then
      _tmp_msg="$(mktemp)"
      printf '%s' "$_last_msg" > "$_tmp_msg"
      if ! ./scripts/commit-msg-guard.sh "$_tmp_msg" 2>/dev/null; then
        _secret_check="fail"
        _all_pass=false
      fi
      rm -f "$_tmp_msg"
    fi
  fi
  _checks="$(printf '%s' "$_checks" | jq --arg s "$_secret_check" '. += [{"check":"secret_leak_detection","result":$s}]')"

  # Check 2: Uncommitted changes
  _uncommitted_check="pass"
  if ! check_uncommitted; then
    _uncommitted_check="warn"
  fi
  _checks="$(printf '%s' "$_checks" | jq --arg s "$_uncommitted_check" '. += [{"check":"uncommitted_changes","result":$s}]')"

  # Check 3: Forbidden patterns in staged files (simplified pre_bash_guard equivalent)
  _forbidden_check="pass"
  _staged="$(git diff --cached --name-only 2>/dev/null || true)"
  if [ -n "$_staged" ]; then
    if printf '%s\n' "$_staged" | grep -qE '\.env$|credentials\.json$|\.pem$'; then
      _forbidden_check="warn"
    fi
  fi
  _checks="$(printf '%s' "$_checks" | jq --arg s "$_forbidden_check" '. += [{"check":"forbidden_file_patterns","result":$s}]')"

  # Write result
  mkdir -p "$EVIDENCE_DIR"
  jq -n --argjson checks "$_checks" --arg ts "$(ts)" --arg pass "$_all_pass" \
    '{"timestamp":$ts,"all_pass":($pass == "true"),"checks":$checks}' > "$_parity_result"

  if [ "$_all_pass" = "false" ]; then
    log_error "Hook parity check failed. See ${_parity_result}"
    return 1
  fi
  log "Hook parity check passed"
  return 0
}

# Stuck detection: returns 0 if stuck, 1 if not
# Compares HEAD commit hash before/after iteration to detect real progress,
# not working tree diff (which is empty after commits, causing false positives).
check_stuck() {
  _stuck_count="$(ckpt_read 'stuck_count' || echo 0)"
  _stuck_count="${_stuck_count:-0}"
  if command -v git >/dev/null 2>&1; then
    _head_after="$(git rev-parse HEAD 2>/dev/null || true)"
    _head_before="$(cat "${PIPELINE_DIR}/.head_before" 2>/dev/null || true)"
    if [ "$_head_before" = "$_head_after" ]; then
      _stuck_count=$((_stuck_count + 1))
      log "Warning: no new commits detected (stuck count: ${_stuck_count}/3)"
    else
      _stuck_count=0
    fi
    ckpt_update ".stuck_count = ${_stuck_count}"
    if [ "$_stuck_count" -ge 3 ]; then
      return 0
    fi
  fi
  return 1
}

# Save HEAD commit hash before an iteration
save_diff_before() {
  if command -v git >/dev/null 2>&1; then
    git rev-parse HEAD 2>/dev/null > "${PIPELINE_DIR}/.head_before" || true
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Preflight capability probe
# ═══════════════════════════════════════════════════════════════════

run_preflight() {
  log "=== Preflight capability probe ==="
  log "  driver: ${RALPH_LOOP_DRIVER}"
  validate_loop_driver
  mkdir -p "$EVIDENCE_DIR"
  _probe_result="${EVIDENCE_DIR}/preflight-probe.json"
  _all_pass=true
  _probes="[]"

  # Probe 1: driver binary available — claude or codex depending on RALPH_LOOP_DRIVER.
  # When driver=codex we still tolerate `claude` being absent here because the
  # cross-review reviewer-inversion path checks for it separately at use time.
  _cli_check="fail"
  case "$RALPH_LOOP_DRIVER" in
    claude) _cli_bin="claude" ;;
    codex)  _cli_bin="codex"  ;;
  esac
  if command -v "$_cli_bin" >/dev/null 2>&1; then
    _cli_check="pass"
  elif [ "$DRY_RUN" -eq 1 ]; then
    _cli_check="skip_dry_run"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_cli_check" --arg n "${_cli_bin}_cli_available" '. += [{"probe":$n,"result":$s}]')"
  log "  ${_cli_bin} binary: ${_cli_check}"

  # Probe 2: jq available
  _jq_check="fail"
  if command -v jq >/dev/null 2>&1; then
    _jq_check="pass"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_jq_check" '. += [{"probe":"jq_available","result":$s}]')"
  log "  jq: ${_jq_check}"

  # Probe 3: instruction file readable from non-interactive mode.
  # driver=claude → CLAUDE.md via `claude -p`
  # driver=codex  → AGENTS.md is the source of truth (see AGENTS.md and
  #                 .codex/AGENTS.override.md). We do a static existence
  #                 check rather than booting codex to avoid the latency
  #                 cost of a full agent turn just for a smoke test.
  _ctxfile_check="fail"
  case "$RALPH_LOOP_DRIVER" in
    claude)
      _probe_name="claude_md_readable"
      if [ "$DRY_RUN" -eq 1 ]; then
        _ctxfile_check="skip_dry_run"
      elif command -v claude >/dev/null 2>&1; then
        _probe_prompt="${PIPELINE_DIR}/.preflight-probe.txt"
        mkdir -p "$PIPELINE_DIR"
        printf 'Reply with exactly the text PROBE_OK if you can read CLAUDE.md in this repository. Nothing else.' > "$_probe_prompt"
        _probe_output="$(claude -p --model "$(resolve_phase_model probe 1)" --effort "$RALPH_EFFORT" --permission-mode "$RALPH_PERMISSION_MODE" --output-format text < "$_probe_prompt" 2>/dev/null || true)"
        if printf '%s' "$_probe_output" | grep -q 'PROBE_OK'; then
          _ctxfile_check="pass"
        else
          _ctxfile_check="warn"
          log "Warning: CLAUDE.md not readable from claude -p mode. Pipeline context may be limited."
          log "Warning: This is expected in nested worktree scenarios."
        fi
        rm -f "$_probe_prompt"
      fi
      ;;
    codex)
      _probe_name="agents_md_readable"
      if [ -f AGENTS.md ]; then
        _ctxfile_check="pass"
      else
        _ctxfile_check="warn"
        log "Warning: AGENTS.md not found — Codex will fall back to default instructions."
      fi
      ;;
  esac
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_ctxfile_check" --arg n "$_probe_name" '. += [{"probe":$n,"result":$s}]')"
  log "  ${_probe_name}: ${_ctxfile_check}"

  # Probe 4: git available
  _git_check="fail"
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    _git_check="pass"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_git_check" '. += [{"probe":"git_available","result":$s}]')"
  log "  git: ${_git_check}"

  # Probe 5: structured-output capability for the active driver.
  # claude → `claude -p --output-format json` round-trip
  # codex  → `codex exec --help` exposes --output-last-message + -s + -c
  case "$RALPH_LOOP_DRIVER" in
    claude)
      _so_check="fail"
      _so_name="json_output_format"
      if [ "$DRY_RUN" -eq 1 ]; then
        _so_check="skip_dry_run"
        JSON_OUTPUT_SUPPORTED=1
      elif [ "$_cli_check" = "pass" ]; then
        _json_probe_prompt="${PIPELINE_DIR}/.json-probe.txt"
        mkdir -p "$PIPELINE_DIR"
        printf 'Reply with exactly the text JSON_PROBE_OK. Nothing else.' > "$_json_probe_prompt"
        _json_probe_raw="$(claude -p --model "$(resolve_phase_model probe 1)" --effort "$RALPH_EFFORT" --permission-mode "$RALPH_PERMISSION_MODE" --output-format json < "$_json_probe_prompt" 2>/dev/null || true)"
        rm -f "$_json_probe_prompt"
        if printf '%s' "$_json_probe_raw" | jq -e '.result' >/dev/null 2>&1; then
          _so_check="pass"
          JSON_OUTPUT_SUPPORTED=1
        else
          _so_check="not_supported"
          log "Warning: --output-format json not supported, falling back to text mode"
        fi
      fi
      ;;
    codex)
      _so_check="fail"
      _so_name="codex_exec_flags"
      # JSON_OUTPUT_SUPPORTED only gates the claude branch in
      # ralph-cli-driver.sh — the codex branch synthesises <log>.json itself
      # via jq. Pin it to 0 so the var is defined for any caller that
      # inspects it; the codex path does not read it.
      # shellcheck disable=SC2034  # exported for any future inspector
      JSON_OUTPUT_SUPPORTED=0
      if [ "$DRY_RUN" -eq 1 ]; then
        _so_check="skip_dry_run"
      elif command -v codex >/dev/null 2>&1; then
        _help="$(codex exec --help 2>&1 || true)"
        _missing=""
        for _flag in "--output-last-message" "--sandbox" "--config"; do
          if ! printf '%s' "$_help" | grep -q -- "$_flag"; then
            _missing="${_missing:+${_missing}, }${_flag}"
          fi
        done
        if [ -z "$_missing" ]; then
          _so_check="pass"
        else
          _so_check="missing_flags"
          log_error "codex exec is missing required flags: ${_missing}"
          _all_pass=false
        fi
      else
        _so_check="codex_missing"
        _all_pass=false
      fi
      ;;
  esac
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_so_check" --arg n "$_so_name" '. += [{"probe":$n,"result":$s}]')"
  log "  ${_so_name}: ${_so_check}"

  # Probe 6: gh CLI (optional but needed for PR creation)
  _gh_check="not_available"
  if command -v gh >/dev/null 2>&1; then
    _gh_check="available"
  else
    log "Warning: gh CLI not found — PR creation will be unavailable"
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_gh_check" '. += [{"probe":"gh_cli","result":$s}]')"
  log "  gh CLI: ${_gh_check}"

  # Probe 7: opposite reviewer availability — needed for cross-review reviewer
  # inversion. When driver=claude the reviewer is codex, and vice versa.
  case "$RALPH_LOOP_DRIVER" in
    claude) _other_bin="codex"  ;;
    codex)  _other_bin="claude" ;;
  esac
  _other_check="not_available"
  if command -v "$_other_bin" >/dev/null 2>&1; then
    _other_check="available"
  else
    log "Warning: ${_other_bin} binary not found — cross-review will be skipped"
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_other_check" --arg n "${_other_bin}_cli" '. += [{"probe":$n,"result":$s}]')"
  log "  ${_other_bin} binary: ${_other_check}"

  # Write probe results
  jq -n --argjson probes "$_probes" --arg ts "$(ts)" --arg pass "$_all_pass" \
    '{"timestamp":$ts,"all_pass":($pass == "true"),"probes":$probes}' > "$_probe_result"

  log "Preflight results saved to ${_probe_result}"

  if [ "$_all_pass" = "false" ]; then
    log_error "Preflight probe FAILED. Pipeline execution blocked."
    log_error "See ${_probe_result} for details."
    return 1
  fi

  log "=== Preflight probe PASSED ==="
  return 0
}

# ═══════════════════════════════════════════════════════════════════
# Inner Loop: implement → self-review → verify → test
# ═══════════════════════════════════════════════════════════════════

run_inner_loop() {
  _cycle="$1"
  _context="${2:-}"
  # 3rd arg: 1-based pass number (1 = initial pass, >= 2 = post-cross-review fix pass).
  # Default to 1 so callers that omit the arg get the non-escalation path.
  _outer_cycle_num="${3:-1}"
  log "=== Inner Loop cycle ${_cycle}/${MAX_INNER_CYCLES} (outer=${_outer_cycle_num}) ==="
  _prev_phase="$(ckpt_read 'phase' || echo 'start')"
  ckpt_update ".phase = \"inner\" | .inner_cycle = ${_cycle}"
  ckpt_transition "$_prev_phase" "inner" "$_context"

  # --- Clear stale sidecar files at cycle start ---
  rm -f "${PIPELINE_DIR}/.agent-signal" "${PIPELINE_DIR}/.pr-url"
  rm -f "${PIPELINE_DIR}/.self-review-result" "${PIPELINE_DIR}/.verify-result" "${PIPELINE_DIR}/.test-result"

  # --- Implementation phase ---
  log "--- Phase: implement ---"
  save_diff_before
  _impl_log="${PIPELINE_DIR}/inner-${_cycle}-implement.log"

  # Build the prompt with context injection
  _impl_prompt="${PIPELINE_DIR}/.impl-prompt.md"
  # Prefer substituted copy from ralph-loop-init.sh --pipeline, fall back to raw template
  if [ -f "${PIPELINE_DIR}/pipeline-inner.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-inner.md" "$_impl_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-inner.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-inner.md" "$_impl_prompt"
  elif [ -f ".harness/state/loop/PROMPT.md" ]; then
    cp ".harness/state/loop/PROMPT.md" "$_impl_prompt"
  else
    log_error "No implementation prompt found. Run ralph-loop-init.sh --pipeline first."
    ckpt_update '.status = "config_error"'
    return 5
  fi

  # Append checkpoint context if resuming or in later cycles
  if [ "$_cycle" -gt 1 ] || [ -n "$_context" ]; then
    {
      echo ""
      echo "## Pipeline context"
      echo ""
      echo "Inner cycle: ${_cycle}"
      if [ -n "$_context" ]; then
        echo "Reason for re-entry: ${_context}"
      fi
      # Include failure info from checkpoint
      _failures="$(ckpt_read 'test_failures' || true)"
      if [ -n "$_failures" ] && [ "$_failures" != "null" ] && [ "$_failures" != "[]" ]; then
        echo ""
        echo "### Previous test failures"
        echo '```json'
        printf '%s\n' "$_failures"
        echo '```'
      fi
      # Include failure triage
      _triage="$(jq '.failure_triage // []' "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null || echo '[]')"
      if [ "$_triage" != "[]" ] && [ "$_triage" != "null" ]; then
        echo ""
        echo "### Failure triage history"
        echo '```json'
        printf '%s\n' "$_triage"
        echo '```'
      fi
    } >> "$_impl_prompt"
  fi

  _impl_extra=""
  _session_id="$(ckpt_read 'session_id' || true)"
  if [ -n "$_session_id" ] && [ "$_cycle" -gt 1 ]; then
    _impl_extra="--resume ${_session_id}"
  fi

  # Per-phase model routing: escalate to RALPH_ESCALATION_MODEL on outer cycle >= 2.
  _impl_model="$(resolve_phase_model implement "$_outer_cycle_num")"
  if [ "$_outer_cycle_num" -ge 2 ] 2>/dev/null; then
    _impl_reason="escalation"
  else
    _impl_reason="phase-default"
  fi
  write_model_receipt implement "$_outer_cycle_num" "$_impl_model" "$_impl_reason"
  run_agent "$_impl_prompt" "$_impl_log" "$_impl_extra" "$_impl_model"

  # In dry-run mode, simulate COMPLETE signal (cleared each cycle start)
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "COMPLETE" > "${PIPELINE_DIR}/.agent-signal"
  fi

  # Capture session ID from JSON output (no grep fallback — warns if absent)
  _new_session=""
  if [ -f "${_impl_log}.json" ] && [ -s "${_impl_log}.json" ]; then
    _new_session="$(jq -r '.session_id // empty' "${_impl_log}.json" 2>/dev/null || true)"
  fi
  if [ -z "$_new_session" ]; then
    log "Warning: session_id not found in JSON output"
  else
    ckpt_update --arg sid "$_new_session" '.session_id = $sid'
  fi

  report_event "implement" "{\"cycle\":${_cycle},\"log\":\"${_impl_log}\"}"

  # Insight event: implement — emitted on iteration success (no parsed verdict;
  # verdict=complete is set on successful agent invocation, before signal check).
  _impl_effective_model="$_impl_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _impl_effective_model="codex-default"
    _impl_honored="false"
  else
    _impl_honored="true"
  fi
  emit_insight_event implement "$_cycle" complete \
    --requested-model "$_impl_model" \
    --effective-model "$_impl_effective_model" \
    --honored "$_impl_honored"

  # Check for COMPLETE/ABORT signals (2-layer detection)
  # Layer 1: sidecar file .agent-signal (written by agent via Bash)
  # Layer 2: marker tag in result text (grep fallback)
  _agent_signal=""
  if [ -f "${PIPELINE_DIR}/.agent-signal" ]; then
    _agent_signal="$(cat "${PIPELINE_DIR}/.agent-signal" 2>/dev/null || true)"
  fi

  # ABORT detection
  _agent_abort=0
  if printf '%s' "$_agent_signal" | grep -qi 'ABORT' 2>/dev/null; then
    _agent_abort=1
  elif grep -q '<promise>ABORT</promise>' "$_impl_log" 2>/dev/null; then
    _agent_abort=1
  fi
  if [ "$_agent_abort" -eq 1 ]; then
    log "Agent signalled ABORT during implementation"
    ckpt_update '.status = "aborted"'
    return 2
  fi

  # COMPLETE detection: agent believes acceptance criteria are met.
  # Still run verify/test to honour the test contract before proceeding to Outer Loop.
  _agent_complete=0
  if printf '%s' "$_agent_signal" | grep -qi 'COMPLETE' 2>/dev/null; then
    log "Agent signalled COMPLETE (via sidecar) — will still run verify/test before proceeding"
    _agent_complete=1
  elif grep -q '<promise>COMPLETE</promise>' "$_impl_log" 2>/dev/null; then
    log "Agent signalled COMPLETE (via marker) — will still run verify/test before proceeding"
    _agent_complete=1
  fi

  # Stuck detection
  if check_stuck; then
    log_error "Stuck detected (3 consecutive iterations with no changes)"
    ckpt_update '.status = "stuck"'
    return 3
  fi

  # --- Self-review phase (agent-driven) ---
  log "--- Phase: self-review ---"
  _review_log="${PIPELINE_DIR}/inner-${_cycle}-self-review.log"
  _review_prompt="${PIPELINE_DIR}/.review-prompt.md"

  if [ -f "${PIPELINE_DIR}/pipeline-self-review.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-self-review.md" "$_review_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-self-review.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-self-review.md" "$_review_prompt"
  else
    cat > "$_review_prompt" <<'REVIEW'
Review the current git diff for code quality issues. Focus on:
1. Unnecessary changes
2. Naming clarity
3. Readability
4. Security concerns
5. Debug code left behind

Write findings to .harness/state/pipeline/self-review.md following the self-review template.
If there are CRITICAL findings, clearly state them.
Write a sidecar: echo '{"critical":0,"high":0,"medium":0,"low":0}' > .harness/state/pipeline/.self-review-result
REVIEW
  fi

  _review_model="$(resolve_phase_model self_review "$_outer_cycle_num")"
  write_model_receipt self_review "$_outer_cycle_num" "$_review_model" "phase-default"
  run_agent "$_review_prompt" "$_review_log" "" "$_review_model"
  report_event "self-review" "{\"cycle\":${_cycle},\"log\":\"${_review_log}\"}"

  # Check for findings (3-layer detection)
  # Layer 1: sidecar file
  _sr_critical=0
  _sr_high=0
  _sr_medium=0
  _sr_low=0
  if [ -f "${PIPELINE_DIR}/.self-review-result" ]; then
    _sr_critical="$(jq -r '.critical // 0' "${PIPELINE_DIR}/.self-review-result" 2>/dev/null || echo 0)"
    _sr_high="$(jq -r '.high // 0' "${PIPELINE_DIR}/.self-review-result" 2>/dev/null || echo 0)"
    _sr_medium="$(jq -r '.medium // 0' "${PIPELINE_DIR}/.self-review-result" 2>/dev/null || echo 0)"
    _sr_low="$(jq -r '.low // 0' "${PIPELINE_DIR}/.self-review-result" 2>/dev/null || echo 0)"
  fi
  # Layer 2: JSON output parse
  if [ "$_sr_critical" -eq 0 ] && [ -f "${_review_log}.json" ] && [ -s "${_review_log}.json" ]; then
    _sr_critical="$(jq -r '.result // empty' "${_review_log}.json" 2>/dev/null | jq -r '.self_review.critical // 0' 2>/dev/null || echo 0)"
  fi
  # Layer 3: grep fallback
  if [ "$_sr_critical" -eq 0 ] && grep -qi 'CRITICAL' "$_review_log" 2>/dev/null; then
    _sr_critical="$(grep -ci 'CRITICAL' "$_review_log" 2>/dev/null || echo 0)"
  fi
  if [ "$_sr_critical" -gt 0 ]; then
    log "Warning: ${_sr_critical} CRITICAL finding(s) detected in self-review"
  fi
  ckpt_update ".self_review_result = {\"critical\":${_sr_critical},\"high\":${_sr_high},\"medium\":${_sr_medium},\"low\":${_sr_low}}"

  # Insight event: self_review — emitted after severity counts are parsed.
  _sr_verdict="pass"
  if [ "$_sr_critical" -gt 0 ]; then _sr_verdict="fail"; fi
  _sr_effective_model="$_review_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _sr_effective_model="codex-default"
    _sr_honored="false"
  else
    _sr_honored="true"
  fi
  emit_insight_event self_review "$_cycle" "$_sr_verdict" \
    --critical "$_sr_critical" --high "$_sr_high" \
    --medium "$_sr_medium" --low "$_sr_low" \
    --requested-model "$_review_model" \
    --effective-model "$_sr_effective_model" \
    --honored "$_sr_honored"

  # --fix-all: ANY self-review findings → override COMPLETE, force retry
  if [ "$FIX_ALL" -eq 1 ]; then
    _sr_total=$((_sr_critical + _sr_high + _sr_medium + _sr_low))
    if [ "$_sr_total" -gt 0 ]; then
      log "fix-all: ${_sr_total} self-review finding(s) — overriding COMPLETE"
      _agent_complete=0
    fi
  fi

  # --- Verify phase (agent-driven) ---
  log "--- Phase: verify ---"
  _verify_log="${PIPELINE_DIR}/inner-${_cycle}-verify.log"
  _verify_prompt="${PIPELINE_DIR}/.verify-prompt.md"

  if [ -f "${PIPELINE_DIR}/pipeline-verify.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-verify.md" "$_verify_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-verify.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-verify.md" "$_verify_prompt"
  else
    cat > "$_verify_prompt" <<'VERIFY'
Verify the current work against the plan's acceptance criteria and run static analysis.
Run: ./scripts/run-static-verify.sh
Write results to .harness/state/pipeline/verify.md
Write a sidecar: echo '{"verdict":"pass","ac_met":0,"ac_total":0}' > .harness/state/pipeline/.verify-result
VERIFY
  fi

  _verify_model="$(resolve_phase_model verify "$_outer_cycle_num")"
  write_model_receipt verify "$_outer_cycle_num" "$_verify_model" "phase-default"
  run_agent "$_verify_prompt" "$_verify_log" "" "$_verify_model"
  report_event "verify" "{\"cycle\":${_cycle},\"log\":\"${_verify_log}\"}"

  # Parse verify verdict (3-layer detection)
  _verify_verdict="pass"
  if [ -f "${PIPELINE_DIR}/.verify-result" ]; then
    _verify_verdict="$(jq -r '.verdict // "pass"' "${PIPELINE_DIR}/.verify-result" 2>/dev/null || echo 'pass')"
  elif [ -f "${_verify_log}.json" ] && [ -s "${_verify_log}.json" ]; then
    _verify_verdict="$(jq -r '.result // empty' "${_verify_log}.json" 2>/dev/null | jq -r '.verify // "pass"' 2>/dev/null || echo 'pass')"
  fi
  if [ "$_verify_verdict" = "fail" ]; then
    log "Warning: verify verdict is FAIL"
  fi
  ckpt_update --arg v "$_verify_verdict" '.verify_result = $v'

  # Insight event: verify — emitted after verdict is parsed (same variables as ckpt_update above).
  _verify_effective_model="$_verify_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _verify_effective_model="codex-default"
    _verify_honored="false"
  else
    _verify_honored="true"
  fi
  emit_insight_event verify "$_cycle" "$_verify_verdict" \
    --requested-model "$_verify_model" \
    --effective-model "$_verify_effective_model" \
    --honored "$_verify_honored"

  # --- Test phase (agent-driven) ---
  log "--- Phase: test ---"
  _test_log="${PIPELINE_DIR}/inner-${_cycle}-test.log"
  _test_prompt="${PIPELINE_DIR}/.test-prompt.md"
  _test_exit=0

  if [ -f "${PIPELINE_DIR}/pipeline-test.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-test.md" "$_test_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-test.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-test.md" "$_test_prompt"
  else
    cat > "$_test_prompt" <<'TESTPROMPT'
Run behavioral tests and produce a test report.
Run: ./scripts/run-test.sh
Write results to .harness/state/pipeline/test.md
Write a sidecar: echo '{"verdict":"pass","total":0,"passed":0,"failed":0}' > .harness/state/pipeline/.test-result
TESTPROMPT
  fi

  _test_model="$(resolve_phase_model test "$_outer_cycle_num")"
  write_model_receipt test "$_outer_cycle_num" "$_test_model" "phase-default"
  run_agent "$_test_prompt" "$_test_log" "" "$_test_model"

  # Parse test verdict (3-layer detection)
  _test_verdict="pass"
  if [ -f "${PIPELINE_DIR}/.test-result" ]; then
    _test_verdict="$(jq -r '.verdict // "pass"' "${PIPELINE_DIR}/.test-result" 2>/dev/null || echo 'pass')"
  elif [ -f "${_test_log}.json" ] && [ -s "${_test_log}.json" ]; then
    _test_verdict="$(jq -r '.result // empty' "${_test_log}.json" 2>/dev/null | jq -r '.test // "pass"' 2>/dev/null || echo 'pass')"
  elif grep -qi 'fail' "$_test_log" 2>/dev/null && ! grep -qi 'no.test.runner\|0 failed' "$_test_log" 2>/dev/null; then
    _test_verdict="fail"
  fi

  if [ "$_test_verdict" = "fail" ]; then
    _test_exit=1
  fi

  report_event "test" "{\"cycle\":${_cycle},\"exit_code\":${_test_exit},\"log\":\"${_test_log}\"}"

  # Insight event: test — emitted after verdict is parsed (same variables as _test_verdict above).
  _test_effective_model="$_test_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _test_effective_model="codex-default"
    _test_honored="false"
  else
    _test_honored="true"
  fi
  emit_insight_event test "$_cycle" "$_test_verdict" \
    --requested-model "$_test_model" \
    --effective-model "$_test_effective_model" \
    --honored "$_test_honored"

  if [ "$_test_exit" -ne 0 ]; then
    log "Tests FAILED in Inner Loop cycle ${_cycle}"
    # Record failure triage entry
    _failure_id="F$(printf '%03d' "$_cycle")"
    ckpt_update ".last_test_result = \"fail\" | .test_failures += [\"cycle_${_cycle}\"]"
    ckpt_update ".failure_triage += [{\"failure_id\":\"${_failure_id}\",\"cycle\":${_cycle},\"test_name\":\"cycle_${_cycle}_tests\",\"hypothesis\":\"pending_agent_analysis\",\"planned_fix\":\"pending_agent_analysis\",\"expected_evidence\":\"test pass after fix\",\"attempt\":1,\"max_attempts\":${MAX_REPAIR_ATTEMPTS},\"resolved\":false,\"timestamp\":\"$(ts)\"}]"

    # Check repair attempt limit
    _total_repairs="$(jq '[.failure_triage[] | select(.resolved == false)] | length' "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null || echo 0)"
    if [ "$_total_repairs" -ge "$MAX_REPAIR_ATTEMPTS" ]; then
      log_error "Repair attempt limit (${MAX_REPAIR_ATTEMPTS}) reached. Escalating to human."
      ckpt_update '.status = "repair_limit"'
      return 4
    fi

    return 1  # Signal to retry Inner Loop
  fi

  # Tests passed
  log "Tests PASSED in Inner Loop cycle ${_cycle}"
  ckpt_update '.last_test_result = "pass"'

  # Run hook parity check
  run_hook_parity || log "Warning: hook parity check had issues"

  # If agent signalled COMPLETE and tests passed, proceed to Outer Loop
  if [ "$_agent_complete" -eq 1 ]; then
    log "Agent COMPLETE confirmed — verify/test passed"
    ckpt_update '.status = "complete"'
    return 0
  fi

  # Tests passed but agent has not signalled COMPLETE — keep iterating
  log "Tests passed but COMPLETE not signalled — continuing Inner Loop"
  return 6
}

# ═══════════════════════════════════════════════════════════════════
# Outer Loop: sync-docs → cross-review → PR
# ═══════════════════════════════════════════════════════════════════

run_outer_loop() {
  _cycle="$1"
  log "=== Outer Loop cycle ${_cycle}/${MAX_OUTER_CYCLES} ==="
  ckpt_update ".phase = \"outer\" | .outer_cycle = ${_cycle}"
  ckpt_transition "inner" "outer" "tests passed"

  # --- Sync docs phase ---
  log "--- Phase: sync-docs ---"
  _docs_log="${PIPELINE_DIR}/outer-${_cycle}-sync-docs.log"
  _docs_prompt="${PIPELINE_DIR}/.docs-prompt.md"

  if [ -f "${PIPELINE_DIR}/pipeline-outer.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-outer.md" "$_docs_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-outer.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-outer.md" "$_docs_prompt"
  else
    cat > "$_docs_prompt" <<'DOCS'
Synchronize documentation with the current implementation changes.
Update any affected docs, rules, and reports.
Commit documentation changes with: docs: <description>
Do NOT create a PR or run cross-review — those are handled by the pipeline.
DOCS
  fi

  _docs_model="$(resolve_phase_model sync_docs "$_cycle")"
  write_model_receipt sync_docs "$_cycle" "$_docs_model" "phase-default"
  run_agent "$_docs_prompt" "$_docs_log" "" "$_docs_model"
  report_event "sync-docs" "{\"cycle\":${_cycle},\"log\":\"${_docs_log}\"}"

  # Insight event: sync_docs — emitted after agent run (no parsed result; verdict=complete).
  _docs_effective_model="$_docs_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _docs_effective_model="codex-default"
    _docs_honored="false"
  else
    _docs_honored="true"
  fi
  emit_insight_event sync_docs "$_cycle" complete \
    --requested-model "$_docs_model" \
    --effective-model "$_docs_effective_model" \
    --honored "$_docs_honored"

  # --- Cross-review phase (driver-aware reviewer inversion) ---
  #
  # The contract is that the reviewer is the other driver,
  # so the cross-model gate is preserved regardless of which driver ran the
  # Inner Loop. Phase 2 (issue #44) adds the codex-driven branch.
  log "--- Phase: cross-review ---"
  _xreview_log="${PIPELINE_DIR}/outer-${_cycle}-cross-review.log"
  _reviewer="$(pick_reviewer)"

  _has_reviewer=false
  if command -v "$_reviewer" >/dev/null 2>&1; then
    _has_reviewer=true
  fi

  _action_required=0
  _worth_considering=0
  _dismissed=0
  _render_failed=0

  if [ "$_has_reviewer" = "true" ] && [ "$DRY_RUN" -eq 0 ]; then
    log "Running cross-review (driver=${RALPH_LOOP_DRIVER}, reviewer=${_reviewer})..."
    _base="$(detect_base_branch)"
    if ! git diff "${_base}...HEAD" --quiet 2>/dev/null; then
      case "$_reviewer" in
        codex)
          # model routing: reviewer=codex (driver inverted from RALPH_LOOP_DRIVER=claude).
          # Pass "codex" as 5th arg so the receipt records the actual reviewer CLI,
          # not RALPH_LOOP_DRIVER (which is "claude" at this call site).
          write_model_receipt cross_review "$_cycle" "codex-default" "cross-review-codex" "codex"
          codex exec review --base "$_base" 2>&1 | tee "$_xreview_log" || true
          ;;
        claude)
          # When codex drove the Inner Loop, ask claude -p to play adversarial
          # reviewer. The prompt lives under .claude/skills/cross-review/prompts/
          # and writes the triage report itself; we only stream its stdout to
          # the per-cycle log.
          _adv_prompt=".claude/skills/cross-review/prompts/adversarial-claude.md"
          if [ -f "$_adv_prompt" ]; then
            # Pre-render ${BASE_BRANCH} / ${REPORTS_DIR} into a per-cycle copy
            # before piping to `claude -p`. `claude -p` does NOT template-
            # substitute its stdin, so literal `${BASE_BRANCH}` previously
            # reached the reviewer and the triage report landed at a literal
            # path the parser could not find — silently bypassing the gate
            # (issue #50).
            #
            # The renderer uses awk index()/substr() instead of gsub() because
            # gsub's replacement string interprets `&` (matched text) and `\`
            # specially, and valid git refs can contain `&` (`feature&1`) and
            # backslashes. index()+substr() treats the replacement value as
            # a literal string, so git refs with `#` / `&` / `\` / `/` and
            # configurable REPORTS_DIR values pass through unchanged.
            # Render failure must fail the gate CLOSED — set _render_failed=1
            # so the gate decision below treats it like ACTION_REQUIRED.
            # Without that, a render that produces no triage report would
            # leave _action_required at its initial 0 and the gate would
            # silently pass — the exact failure shape of issue #50 the
            # renderer is meant to fix.
            _rendered_prompt="${PIPELINE_DIR}/outer-${_cycle}-adversarial-claude.md"
            if ! BASE_BRANCH="$_base" REPORTS_DIR="$REPORTS_DIR" \
                 awk '
                   function lreplace(s, needle, repl,    out, idx) {
                     out = ""
                     while ((idx = index(s, needle)) > 0) {
                       out = out substr(s, 1, idx - 1) repl
                       s = substr(s, idx + length(needle))
                     }
                     return out s
                   }
                   {
                     line = $0
                     line = lreplace(line, "${BASE_BRANCH}", ENVIRON["BASE_BRANCH"])
                     line = lreplace(line, "${REPORTS_DIR}", ENVIRON["REPORTS_DIR"])
                     print line
                   }
                 ' "$_adv_prompt" > "$_rendered_prompt"; then
              log_error "cross-review: failed to render adversarial prompt to ${_rendered_prompt}"
              echo "render_failed_awk" > "$_xreview_log"
              _render_failed=1
            else
              # Allowlist-based unresolved-placeholder guard. Any ${...}
              # token remaining in the rendered output means a placeholder
              # was added to the prompt without updating the renderer.
              # Fail the gate closed rather than passing a broken prompt
              # to the reviewer.
              _leftover_placeholders="$(grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' "$_rendered_prompt" 2>/dev/null | sort -u || true)"
              if [ -n "$_leftover_placeholders" ]; then
                log_error "cross-review: unresolved placeholders in rendered prompt: ${_leftover_placeholders}"
                echo "render_failed_unresolved_placeholders" > "$_xreview_log"
                _render_failed=1
              fi
            fi

            if [ "$_render_failed" -eq 0 ]; then
              # `--permission-mode auto` (not plan) is required because the
              # adversarial reviewer must write the triage report into
              # docs/reports/. Plan mode is read-only and silently drops the
              # write — the parser then sees zero findings and the cross-model
              # gate is bypassed (cycle-2 cross-review P1, #44).
              # reviewer=claude (driver inverted from RALPH_LOOP_DRIVER=codex).
              # Pass "claude" as 5th arg so the receipt records driver=claude,
              # effective_model=$RALPH_CLAUDE_REVIEWER_MODEL, honored=true.
              write_model_receipt cross_review "$_cycle" "$RALPH_CLAUDE_REVIEWER_MODEL" "reviewer-inversion" "claude"
              claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" \
                --permission-mode auto --output-format text \
                < "$_rendered_prompt" 2>&1 | tee "$_xreview_log" || true
            fi
          else
            log "Warning: adversarial-claude prompt missing at ${_adv_prompt}"
            echo "missing_adversarial_prompt" > "$_xreview_log"
          fi
          ;;
      esac

      # Parse triage results from the latest triage report (CLI-neutral path).
      # The template ships with literal `## ACTION_REQUIRED` / `## WORTH_CONSIDERING`
      # / `## DISMISSED` headings plus a summary header line, so a naive
      # `grep -c 'ACTION_REQUIRED'` over the whole file always reports >=2
      # matches even on a clean report and forces a spurious Inner Loop
      # regression. Prefer the canonical `After triage: ACTION_REQUIRED=N, ...`
      # summary line; fall back to counting `|` table rows under each heading
      # when the summary is missing.
      _triage_report="$(find "$REPORTS_DIR" -name 'cross-review-triage-*' -newer "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null | tail -1 || true)"
      if [ -n "$_triage_report" ]; then
        _action_required="$(count_triage_findings "$_triage_report" ACTION_REQUIRED)"
        _worth_considering="$(count_triage_findings "$_triage_report" WORTH_CONSIDERING)"
        _dismissed="$(count_triage_findings "$_triage_report" DISMISSED)"
      fi
    else
      log "No diff against ${_base} — skipping cross-review"
      echo "no_diff" > "$_xreview_log"
    fi
  else
    log "${_reviewer} binary not available — skipping cross-review"
    printf '%s_not_available\n' "$_reviewer" > "$_xreview_log"
  fi

  ckpt_update ".cross_review_triage = {\"driver\":\"${RALPH_LOOP_DRIVER}\",\"reviewer\":\"${_reviewer}\",\"action_required\":${_action_required},\"worth_considering\":${_worth_considering},\"dismissed\":${_dismissed},\"render_failed\":${_render_failed}}"
  report_event "cross-review" "{\"cycle\":${_cycle},\"driver\":\"${RALPH_LOOP_DRIVER}\",\"reviewer\":\"${_reviewer}\",\"action_required\":${_action_required},\"worth_considering\":${_worth_considering},\"dismissed\":${_dismissed},\"render_failed\":${_render_failed}}"

  # Insight event: cross_review — emitted after triage counts are parsed.
  # verdict: action_required when render failed or _action_required > 0; pass otherwise.
  # Render failure is checked first (same order as the gate below: render → action_required).
  # Routing fields: cross-review uses the *reviewer* CLI, not RALPH_LOOP_DRIVER.
  # The reviewer model and driver come from write_model_receipt call sites above.
  _xr_verdict="pass"
  if [ "$_render_failed" -ne 0 ]; then
    _xr_verdict="action_required"
  elif [ "$_action_required" -gt 0 ]; then
    _xr_verdict="action_required"
  fi
  # Reviewer is always the inverted CLI; model is the reviewer model used above.
  case "$_reviewer" in
    codex)
      _xr_req_model="codex-default"
      _xr_eff_model="codex-default"
      _xr_honored="false"
      ;;
    *)
      _xr_req_model="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"
      _xr_eff_model="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"
      _xr_honored="true"
      ;;
  esac
  emit_insight_event cross_review "$_cycle" "$_xr_verdict" \
    --action-required "$_action_required" \
    --worth-considering "$_worth_considering" \
    --dismissed "$_dismissed" \
    --requested-model "$_xr_req_model" \
    --effective-model "$_xr_eff_model" \
    --honored "$_xr_honored"

  # Decision: regress to Inner Loop or proceed to PR.
  #
  # Render failure is treated as gate failure (fail closed) — without
  # this, a renderer that produces no triage report would leave
  # _action_required at 0 and the gate would silently pass. That is the
  # exact silent-bypass shape of issue #50.
  if [ "$_render_failed" -ne 0 ]; then
    log "Adversarial prompt render failed — regressing to Inner Loop (gate fails closed)"
    return 1
  fi

  if [ "$_action_required" -gt 0 ]; then
    log "ACTION_REQUIRED findings (${_action_required}) detected — regressing to Inner Loop"
    return 1  # Signal to re-enter Inner Loop
  fi

  # --fix-all: WORTH_CONSIDERING → treat as ACTION_REQUIRED
  if [ "$FIX_ALL" -eq 1 ] && [ "$_worth_considering" -gt 0 ]; then
    log "fix-all: ${_worth_considering} WORTH_CONSIDERING finding(s) — regressing to Inner Loop"
    return 1
  fi

  if [ "$_worth_considering" -gt 0 ]; then
    log "WORTH_CONSIDERING findings (${_worth_considering}) detected, but no ACTION_REQUIRED — proceeding to PR"
  fi

  # --- PR creation phase ---
  if [ "$SKIP_PR" -eq 1 ]; then
    log "--- Phase: PR creation (skipped — --skip-pr) ---"
    ckpt_update '.status = "complete"'
    return 0
  fi

  log "--- Phase: PR creation ---"
  if ! command -v gh >/dev/null 2>&1; then
    log_error "gh CLI not found — cannot create PR. Install gh and retry."
    ckpt_update '.status = "gh_unavailable"'
    return 2  # distinct from 1 (ACTION_REQUIRED) — terminal config error
  fi
  _head_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
  if ! ./scripts/branch-name.sh validate "$_head_branch" >/dev/null 2>&1; then
    log_error "Invalid PR branch name: ${_head_branch}. Expected <type>/<slug> or <type>/<issue>/<slug>."
    ckpt_update '.status = "invalid_branch_name"'
    return 2
  fi
  _title_prefix="$(./scripts/branch-name.sh title-prefix "$_head_branch")"
  _pr_log="${PIPELINE_DIR}/outer-${_cycle}-pr.log"
  _pr_prompt="${PIPELINE_DIR}/.pr-prompt.md"

  cat > "$_pr_prompt" <<PR_PROMPT
Create a pull request for the current branch.
Follow the repository's PR workflow:
1. Check for uncommitted changes and commit them
2. Push the branch
3. Create a ready-for-review PR with Japanese title and body. The PR title must start with "${_title_prefix} ".
4. Archive the plan

Use gh pr create with the standard template. Do not pass --draft unless the
operator explicitly requested a draft. After creation, run:
  ./scripts/ensure-pr-title-prefix.sh <pr-url-or-current-branch>
  ./scripts/ensure-pr-ready.sh <pr-url-or-current-branch>

After creating the PR, write the PR URL to .harness/state/pipeline/.pr-url:
  echo "https://github.com/..." > .harness/state/pipeline/.pr-url
PR_PROMPT

  _pr_model="$(resolve_phase_model pr "$_cycle")"
  write_model_receipt pr "$_cycle" "$_pr_model" "phase-default"
  run_agent "$_pr_prompt" "$_pr_log" "" "$_pr_model"

  # Detect PR URL (3-layer defense)
  _pr_url=""

  # Layer 1: external verification via gh CLI
  if [ -n "$_head_branch" ] && command -v gh >/dev/null 2>&1; then
    _pr_url="$(gh pr list --head "$_head_branch" --state open --json url --jq '.[0].url' 2>/dev/null || true)"
    if [ -n "$_pr_url" ]; then
      log "PR detected via gh pr list: ${_pr_url}"
    fi
  fi

  # Layer 2: sidecar file written by agent
  if [ -z "$_pr_url" ] && [ -f "${PIPELINE_DIR}/.pr-url" ]; then
    _pr_url="$(cat "${PIPELINE_DIR}/.pr-url" 2>/dev/null | grep -oE 'https://github\.com/[^ ]+/pull/[0-9]+' | head -1 || true)"
    if [ -n "$_pr_url" ]; then
      log "PR detected via sidecar file: ${_pr_url}"
    fi
  fi

  # Layer 3: grep agent output log (legacy fallback)
  if [ -z "$_pr_url" ]; then
    _pr_url="$(grep -oE 'https://github\.com/[^ ]+/pull/[0-9]+' "$_pr_log" 2>/dev/null | head -1 || true)"
    if [ -n "$_pr_url" ]; then
      log "PR detected via log grep: ${_pr_url}"
    fi
  fi

  if [ -n "$_pr_url" ]; then
    if ! ./scripts/ensure-pr-title-prefix.sh "$_pr_url" >> "$_pr_log" 2>&1; then
      log_error "PR exists but could not be verified with branch type title prefix: ${_pr_url}"
      ckpt_update --arg url "$_pr_url" '.pr_created = true | .pr_url = $url | .status = "pr_title_prefix_check_failed"'
      return 2
    fi
    if ! ./scripts/ensure-pr-ready.sh "$_pr_url" >> "$_pr_log" 2>&1; then
      log_error "PR exists but could not be verified as ready-for-review: ${_pr_url}"
      ckpt_update --arg url "$_pr_url" '.pr_created = true | .pr_url = $url | .status = "pr_draft_or_ready_check_failed"'
      return 2
    fi
    log "PR created: ${_pr_url}"
    ckpt_update --arg url "$_pr_url" '.pr_created = true | .pr_url = $url | .status = "complete"'
    _pr_event="$(jq -n --argjson c "$_cycle" --arg u "$_pr_url" '{"cycle":$c,"url":$u}')"
    report_event "pr-created" "$_pr_event"
  else
    log "PR creation step completed but URL not detected (check log for details)"
    ckpt_update ".status = \"complete\""
    report_event "pr-step" "{\"cycle\":${_cycle},\"log\":\"${_pr_log}\"}"
  fi

  # Insight event: pr — emitted after PR creation step completes.
  _pr_effective_model="$_pr_model"
  if [ "${RALPH_LOOP_DRIVER:-claude}" = "codex" ]; then
    _pr_effective_model="codex-default"
    _pr_honored="false"
  else
    _pr_honored="true"
  fi
  emit_insight_event pr "$_cycle" complete \
    --requested-model "$_pr_model" \
    --effective-model "$_pr_effective_model" \
    --honored "$_pr_honored"

  return 0
}

# ═══════════════════════════════════════════════════════════════════
# Main pipeline orchestrator
# ═══════════════════════════════════════════════════════════════════

main() {
  log "=== Ralph Pipeline v2 ==="
  log "Max iterations: ${MAX_ITERATIONS}"
  log "Max inner cycles: ${MAX_INNER_CYCLES}"
  log "Max outer cycles: ${MAX_OUTER_CYCLES}"
  log "Max repair attempts: ${MAX_REPAIR_ATTEMPTS}"
  log "Skip PR: ${SKIP_PR}"
  log "Fix all: ${FIX_ALL}"
  log "Verify scope: ${RALPH_VERIFY_SCOPE}"
  log "Dry run: ${DRY_RUN}"
  log ""

  mkdir -p "$PIPELINE_DIR" "$EVIDENCE_DIR" "$REPORTS_DIR"

  # --- Insight event identity: run_id and slug ---
  # PIPELINE_RUN_ID: unique per invocation; ts_file format avoids colons/spaces.
  # PIPELINE_SLUG: derived from branch name (<type>/<slug> → <slug>).
  # Both are exported so emit_insight_event can read them as globals.
  _pii_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
  # Strip type prefix (everything up to and including the first '/').
  # If branch has issue component (type/NNN/slug), strip two prefixes.
  _pii_slug="${_pii_branch##*/}"
  if [ -z "$_pii_slug" ] || [ "$_pii_slug" = "$_pii_branch" ]; then
    _pii_slug="unknown"
  fi
  PIPELINE_SLUG="$_pii_slug"
  PIPELINE_RUN_ID="$(ts_file)-$$"

  # --- Preflight ---
  if ! run_preflight; then
    exit 1
  fi

  if [ "$PREFLIGHT_ONLY" -eq 1 ]; then
    log "Preflight-only mode. Exiting."
    exit 0
  fi

  # --- Initialize or resume checkpoint ---
  if [ "$RESUME" -eq 1 ] && [ -f "${PIPELINE_DIR}/checkpoint.json" ]; then
    log "Resuming from existing checkpoint"
    _inner_cycle="$(ckpt_read 'inner_cycle' || echo 1)"
    _outer_cycle="$(ckpt_read 'outer_cycle' || echo 0)"
  else
    _inner_cycle=1
    _outer_cycle=0
    cat > "${PIPELINE_DIR}/checkpoint.json" <<INIT_JSON
{
  "schema_version": 1,
  "iteration": 0,
  "phase": "preflight",
  "status": "running",
  "inner_cycle": 0,
  "outer_cycle": 0,
  "stuck_count": 0,
  "last_test_result": null,
  "test_failures": [],
  "failure_triage": [],
  "self_review_result": null,
  "verify_result": null,
  "review_findings": [],
  "cross_review_triage": {"driver": null, "reviewer": null, "action_required": 0, "worth_considering": 0, "dismissed": 0},
  "acceptance_criteria_met": [],
  "acceptance_criteria_remaining": [],
  "session_id": null,
  "pr_created": false,
  "pr_url": null,
  "phase_transitions": []
}
INIT_JSON
    : > "${PIPELINE_DIR}/execution-events.jsonl"
    log "Initialized fresh checkpoint"
  fi

  ckpt_update '.status = "running"'
  ckpt_transition "preflight" "inner" "pipeline start"

  _total_iteration=0
  _context=""

  # --- Main loop ---
  while [ "$_total_iteration" -lt "$MAX_ITERATIONS" ]; do
    _total_iteration=$((_total_iteration + 1))
    ckpt_update ".iteration = ${_total_iteration}"

    # Inner Loop
    while [ "$_inner_cycle" -le "$MAX_INNER_CYCLES" ] && [ "$_total_iteration" -le "$MAX_ITERATIONS" ]; do
      _inner_result=0
      run_inner_loop "$_inner_cycle" "$_context" "$((_outer_cycle + 1))" || _inner_result=$?

      case "$_inner_result" in
        0) # COMPLETE + tests passed → move to Outer Loop
          break
          ;;
        1) # Tests failed → retry Inner Loop
          _inner_cycle=$((_inner_cycle + 1))
          _total_iteration=$((_total_iteration + 1))
          _context="test failure — retry"
          ;;
        6) # Tests passed but COMPLETE not signalled → continue Inner Loop
          _inner_cycle=$((_inner_cycle + 1))
          _total_iteration=$((_total_iteration + 1))
          _context="tests pass, awaiting COMPLETE signal"
          ;;
        2) # ABORT
          log "=== Pipeline aborted by agent ==="
          _finalize "aborted"
          return 0
          ;;
        3) # Stuck
          log "=== Pipeline stopped: stuck ==="
          _finalize "stuck"
          return 0
          ;;
        4) # Repair limit
          log "=== Pipeline stopped: repair limit reached ==="
          _finalize "repair_limit"
          return 0
          ;;
        5) # Config error (missing prompt, etc.)
          log "=== Pipeline stopped: configuration error ==="
          _finalize "config_error"
          return 1
          ;;
      esac
    done

    # Check if inner cycle limit exceeded
    if [ "$_inner_cycle" -gt "$MAX_INNER_CYCLES" ]; then
      log_error "Max Inner Loop cycles (${MAX_INNER_CYCLES}) reached. Escalating."
      _finalize "max_inner_cycles"
      return 0
    fi

    # Outer Loop
    _outer_cycle=$((_outer_cycle + 1))
    if [ "$_outer_cycle" -gt "$MAX_OUTER_CYCLES" ]; then
      log_error "Max Outer Loop cycles (${MAX_OUTER_CYCLES}) reached. Escalating."
      _finalize "max_outer_cycles"
      return 0
    fi

    _outer_result=0
    run_outer_loop "$_outer_cycle" || _outer_result=$?

    case "$_outer_result" in
      0) # PR created → done
        log "=== Pipeline complete ==="
        _finalize "complete"
        return 0
        ;;
      1) # ACTION_REQUIRED → regress to Inner Loop
        _inner_cycle=$((_inner_cycle + 1))
        _context="cross-review ACTION_REQUIRED — regressed from Outer Loop"
        ckpt_transition "outer" "inner" "cross-review ACTION_REQUIRED"
        ;;
      2) # Terminal config error (e.g., gh_unavailable) → stop pipeline
        log "=== Pipeline stopped: missing dependency ==="
        _finalize "gh_unavailable"
        return 0
        ;;
    esac
  done

  # Max total iterations reached
  log_error "Max total iterations (${MAX_ITERATIONS}) reached."
  _finalize "max_iterations"
}

_finalize() {
  _final_status="$1"
  _end_ts="$(ts)"
  ckpt_update ".status = \"${_final_status}\""

  # Write execution report
  _report_file="${REPORTS_DIR}/pipeline-execution-$(ts_file).json"
  jq -n \
    --arg status "$_final_status" \
    --arg start "$(jq -r '.phase_transitions[0].timestamp // empty' "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null || echo '')" \
    --arg end "$_end_ts" \
    --argjson checkpoint "$(cat "${PIPELINE_DIR}/checkpoint.json")" \
    '{status:$status,started:$start,ended:$end,checkpoint:$checkpoint}' > "$_report_file"

  log ""
  log "=== Ralph Pipeline summary ==="
  log "  Status: ${_final_status}"
  log "  Iterations: $(ckpt_read 'iteration')"
  log "  Inner cycles: $(ckpt_read 'inner_cycle')"
  log "  Outer cycles: $(ckpt_read 'outer_cycle')"
  _sr="$(ckpt_read 'self_review_result' || true)"
  if [ -n "$_sr" ] && [ "$_sr" != "null" ]; then
    log "  Self-review: ${_sr}"
  fi
  _vr="$(ckpt_read 'verify_result' || true)"
  if [ -n "$_vr" ] && [ "$_vr" != "null" ]; then
    log "  Verify: ${_vr}"
  fi
  log "  Checkpoint: ${PIPELINE_DIR}/checkpoint.json"
  log "  Report: ${_report_file}"
  if [ "$(ckpt_read 'pr_created')" = "true" ]; then
    log "  PR: $(ckpt_read 'pr_url')"
  fi
}

# Source guard: allow test files to source this script to reach functions
# without triggering pipeline execution.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
