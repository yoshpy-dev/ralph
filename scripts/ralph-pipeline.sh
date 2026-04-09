#!/usr/bin/env sh
set -eu

# Ralph Pipeline orchestrator — full autonomous development pipeline
# Inner Loop: implement → self-review → verify → test (repeat on failure)
# Outer Loop: sync-docs → codex-review (repeat on ACTION_REQUIRED) → PR
#
# State lives in .harness/state/pipeline/
# Requires: claude CLI, jq, git

PIPELINE_DIR=".harness/state/pipeline"
EVIDENCE_DIR="docs/evidence"
REPORTS_DIR="docs/reports"
MAX_ITERATIONS=20
MAX_INNER_CYCLES=10
MAX_OUTER_CYCLES=3
MAX_REPAIR_ATTEMPTS=5
DRY_RUN=0
PREFLIGHT_ONLY=0
RESUME=0
JSON_OUTPUT_SUPPORTED=0

usage() {
  cat <<'USAGE'
Usage: ralph-pipeline.sh [OPTIONS]

Full autonomous development pipeline with Inner/Outer Loop architecture.

Options:
  --max-iterations N       Total iteration cap across all cycles (default: 20)
  --max-inner-cycles N     Max Inner Loop cycles before escalation (default: 10)
  --max-outer-cycles N     Max Outer Loop regressions before escalation (default: 3)
  --max-repair-attempts N  Max fix attempts per failing test (default: 5)
  --preflight              Run capability probe only, then exit
  --resume                 Resume from existing checkpoint.json
  --dry-run                Print what would run without executing claude
  -h, --help               Show this help
USAGE
  exit 0
}

while [ $# -gt 0 ]; do
  case "$1" in
    --max-iterations)     shift; MAX_ITERATIONS="${1:?requires a number}" ;;
    --max-inner-cycles)   shift; MAX_INNER_CYCLES="${1:?requires a number}" ;;
    --max-outer-cycles)   shift; MAX_OUTER_CYCLES="${1:?requires a number}" ;;
    --max-repair-attempts) shift; MAX_REPAIR_ATTEMPTS="${1:?requires a number}" ;;
    --preflight)          PREFLIGHT_ONLY=1 ;;
    --resume)             RESUME=1 ;;
    --dry-run)            DRY_RUN=1 ;;
    -h|--help)            usage ;;
    *)                    echo "Unknown option: $1"; usage ;;
  esac
  shift
done

# ═══════════════════════════════════════════════════════════════════
# Utility functions
# ═══════════════════════════════════════════════════════════════════

ts() { date -u '+%Y-%m-%dT%H:%M:%SZ'; }
ts_file() { date -u '+%Y-%m-%d-%H%M%S'; }

log() { printf '[%s] %s\n' "$(ts)" "$*"; }
log_error() { printf '[%s] ERROR: %s\n' "$(ts)" "$*" >&2; }

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

# Run claude -p with a prompt file
# Outputs result text to $_log_file and full JSON (if available) to ${_log_file}.json
run_claude() {
  _prompt_file="$1"
  _log_file="$2"
  _extra_args="${3:-}"
  if [ "$DRY_RUN" -eq 1 ]; then
    log "[dry-run] Would run: claude -p < ${_prompt_file} ${_extra_args}"
    echo "[dry-run] iteration output" > "$_log_file"
    printf '{"result":"[dry-run] iteration output","session_id":null}' > "${_log_file}.json"
    return 0
  fi
  if [ "$JSON_OUTPUT_SUPPORTED" -eq 1 ]; then
    # JSON mode: separate stdout (JSON) from stderr
    # shellcheck disable=SC2086
    claude -p --output-format json ${_extra_args} < "$_prompt_file" > "${_log_file}.json" 2>"${_log_file}.stderr" || true
    # Extract .result from JSON; fall back to raw output on parse failure
    if jq -e '.result' "${_log_file}.json" >/dev/null 2>&1; then
      jq -r '.result // empty' "${_log_file}.json" > "$_log_file"
    else
      log "Warning: JSON parse failed for ${_log_file}.json, using raw output"
      cp "${_log_file}.json" "$_log_file"
    fi
    # Show result on stdout for visibility
    cat "$_log_file"
  else
    # Text fallback mode (older claude CLI or JSON not supported)
    # shellcheck disable=SC2086
    claude -p --output-format text ${_extra_args} < "$_prompt_file" 2>&1 | tee "$_log_file"
    # No JSON sidecar in text mode
    : > "${_log_file}.json"
  fi
}

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
      if ! printf '%s' "$_last_msg" | ./scripts/commit-msg-guard.sh 2>/dev/null; then
        _secret_check="fail"
        _all_pass=false
      fi
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
  mkdir -p "$EVIDENCE_DIR"
  _probe_result="${EVIDENCE_DIR}/preflight-probe.json"
  _all_pass=true
  _probes="[]"

  # Probe 1: claude CLI available
  _cli_check="fail"
  if command -v claude >/dev/null 2>&1; then
    _cli_check="pass"
  elif [ "$DRY_RUN" -eq 1 ]; then
    _cli_check="skip_dry_run"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_cli_check" '. += [{"probe":"claude_cli_available","result":$s}]')"
  log "  claude CLI: ${_cli_check}"

  # Probe 2: jq available
  _jq_check="fail"
  if command -v jq >/dev/null 2>&1; then
    _jq_check="pass"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_jq_check" '. += [{"probe":"jq_available","result":$s}]')"
  log "  jq: ${_jq_check}"

  # Probe 3: CLAUDE.md readable from -p mode
  _claudemd_check="fail"
  if [ "$DRY_RUN" -eq 1 ]; then
    _claudemd_check="skip_dry_run"
  elif command -v claude >/dev/null 2>&1; then
    _probe_prompt="${PIPELINE_DIR}/.preflight-probe.txt"
    mkdir -p "$PIPELINE_DIR"
    printf 'Reply with exactly the text PROBE_OK if you can read CLAUDE.md in this repository. Nothing else.' > "$_probe_prompt"
    _probe_output="$(claude -p --output-format text < "$_probe_prompt" 2>/dev/null || true)"
    if printf '%s' "$_probe_output" | grep -q 'PROBE_OK'; then
      _claudemd_check="pass"
    else
      _all_pass=false
    fi
    rm -f "$_probe_prompt"
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_claudemd_check" '. += [{"probe":"claude_md_readable","result":$s}]')"
  log "  CLAUDE.md readable: ${_claudemd_check}"

  # Probe 4: git available
  _git_check="fail"
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    _git_check="pass"
  else
    _all_pass=false
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_git_check" '. += [{"probe":"git_available","result":$s}]')"
  log "  git: ${_git_check}"

  # Probe 5: claude -p --output-format json support
  _json_check="fail"
  if [ "$DRY_RUN" -eq 1 ]; then
    _json_check="skip_dry_run"
    JSON_OUTPUT_SUPPORTED=1
  elif [ "$_cli_check" = "pass" ]; then
    _json_probe_prompt="${PIPELINE_DIR}/.json-probe.txt"
    mkdir -p "$PIPELINE_DIR"
    printf 'Reply with exactly the text JSON_PROBE_OK. Nothing else.' > "$_json_probe_prompt"
    _json_probe_raw="$(claude -p --output-format json < "$_json_probe_prompt" 2>/dev/null || true)"
    rm -f "$_json_probe_prompt"
    if printf '%s' "$_json_probe_raw" | jq -e '.result' >/dev/null 2>&1; then
      _json_check="pass"
      JSON_OUTPUT_SUPPORTED=1
    else
      _json_check="not_supported"
      log "Warning: --output-format json not supported, falling back to text mode"
    fi
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_json_check" '. += [{"probe":"json_output_format","result":$s}]')"
  log "  JSON output format: ${_json_check}"

  # Probe 6: codex CLI (optional)
  _codex_check="not_available"
  if [ -x ./scripts/codex-check.sh ]; then
    if ./scripts/codex-check.sh >/dev/null 2>&1; then
      _codex_check="available"
    fi
  fi
  _probes="$(printf '%s' "$_probes" | jq --arg s "$_codex_check" '. += [{"probe":"codex_cli","result":$s}]')"
  log "  codex CLI: ${_codex_check}"

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
  log "=== Inner Loop cycle ${_cycle}/${MAX_INNER_CYCLES} ==="
  _prev_phase="$(ckpt_read 'phase' || echo 'start')"
  ckpt_update ".phase = \"inner\" | .inner_cycle = ${_cycle}"
  ckpt_transition "$_prev_phase" "inner" "$_context"

  # --- Clear stale sidecar files at cycle start ---
  rm -f "${PIPELINE_DIR}/.agent-signal" "${PIPELINE_DIR}/.pr-url"

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

  run_claude "$_impl_prompt" "$_impl_log" "$_impl_extra"

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

  # --- Self-review phase ---
  log "--- Phase: self-review ---"
  _review_log="${PIPELINE_DIR}/inner-${_cycle}-self-review.log"
  _review_prompt="${PIPELINE_DIR}/.review-prompt.md"

  if [ -f "${PIPELINE_DIR}/pipeline-review.md" ]; then
    cp "${PIPELINE_DIR}/pipeline-review.md" "$_review_prompt"
  elif [ -f ".claude/skills/loop/prompts/pipeline-review.md" ]; then
    cp ".claude/skills/loop/prompts/pipeline-review.md" "$_review_prompt"
  else
    cat > "$_review_prompt" <<'REVIEW'
Review the current git diff for code quality issues. Focus on:
1. Unnecessary changes
2. Naming clarity
3. Readability
4. Security concerns
5. Debug code left behind

Write findings to .harness/state/pipeline/ following the self-review template.
If there are CRITICAL findings, clearly state them.
REVIEW
  fi

  run_claude "$_review_prompt" "$_review_log" ""
  report_event "self-review" "{\"cycle\":${_cycle},\"log\":\"${_review_log}\"}"

  # Check for CRITICAL findings (simple heuristic)
  if grep -qi 'CRITICAL' "$_review_log" 2>/dev/null; then
    _critical_count="$(grep -ci 'CRITICAL' "$_review_log" 2>/dev/null || echo 0)"
    log "Warning: ${_critical_count} CRITICAL finding(s) detected in self-review"
    # Don't stop — let verify and test catch real issues
  fi

  # --- Verify phase ---
  log "--- Phase: verify ---"
  _verify_log="${PIPELINE_DIR}/inner-${_cycle}-verify.log"
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] Would run: ./scripts/run-static-verify.sh" > "$_verify_log"
  elif [ -x ./scripts/run-static-verify.sh ]; then
    ./scripts/run-static-verify.sh 2>&1 | tee "$_verify_log" || true
  elif [ -x ./scripts/run-verify.sh ]; then
    HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh 2>&1 | tee "$_verify_log" || true
  fi
  report_event "verify" "{\"cycle\":${_cycle},\"log\":\"${_verify_log}\"}"

  # --- Test phase ---
  log "--- Phase: test ---"
  _test_log="${PIPELINE_DIR}/inner-${_cycle}-test.log"
  _test_exit=0
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] Would run: ./scripts/run-test.sh" > "$_test_log"
  elif [ -x ./scripts/run-test.sh ]; then
    if ! ./scripts/run-test.sh 2>&1 | tee "$_test_log"; then
      _test_exit=1
    fi
  elif [ -x ./scripts/run-verify.sh ]; then
    if ! HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh 2>&1 | tee "$_test_log"; then
      _test_exit=1
    fi
  else
    log "No test runner found — skipping test phase"
    echo "no_test_runner" > "$_test_log"
  fi

  report_event "test" "{\"cycle\":${_cycle},\"exit_code\":${_test_exit},\"log\":\"${_test_log}\"}"

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
# Outer Loop: sync-docs → codex-review → PR
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
Do NOT create a PR or run codex review — those are handled by the pipeline.
DOCS
  fi

  run_claude "$_docs_prompt" "$_docs_log" ""
  report_event "sync-docs" "{\"cycle\":${_cycle},\"log\":\"${_docs_log}\"}"

  # --- Codex review phase ---
  log "--- Phase: codex-review ---"
  _codex_log="${PIPELINE_DIR}/outer-${_cycle}-codex-review.log"
  _has_codex=false

  if [ -x ./scripts/codex-check.sh ] && ./scripts/codex-check.sh >/dev/null 2>&1; then
    _has_codex=true
  fi

  _action_required=0
  _worth_considering=0
  _dismissed=0

  if [ "$_has_codex" = "true" ] && [ "$DRY_RUN" -eq 0 ]; then
    log "Running codex review..."
    _base="$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||' || echo main)"
    if ! git diff "${_base}...HEAD" --quiet 2>/dev/null; then
      codex exec review --base "$_base" 2>&1 | tee "$_codex_log" || true

      # Parse triage results from the codex output or triage report
      _triage_report="$(find "$REPORTS_DIR" -name 'codex-triage-*' -newer "${PIPELINE_DIR}/checkpoint.json" 2>/dev/null | tail -1 || true)"
      if [ -n "$_triage_report" ]; then
        _action_required="$(grep -c 'ACTION_REQUIRED' "$_triage_report" 2>/dev/null || echo 0)"
        _worth_considering="$(grep -c 'WORTH_CONSIDERING' "$_triage_report" 2>/dev/null || echo 0)"
        _dismissed="$(grep -c 'DISMISSED' "$_triage_report" 2>/dev/null || echo 0)"
      fi
    else
      log "No diff against ${_base} — skipping codex review"
      echo "no_diff" > "$_codex_log"
    fi
  else
    log "Codex CLI not available — skipping codex review"
    echo "codex_not_available" > "$_codex_log"
  fi

  ckpt_update ".codex_triage = {\"action_required\":${_action_required},\"worth_considering\":${_worth_considering},\"dismissed\":${_dismissed}}"
  report_event "codex-review" "{\"cycle\":${_cycle},\"action_required\":${_action_required},\"worth_considering\":${_worth_considering},\"dismissed\":${_dismissed}}"

  # Decision: regress to Inner Loop or proceed to PR
  if [ "$_action_required" -gt 0 ]; then
    log "ACTION_REQUIRED findings (${_action_required}) detected — regressing to Inner Loop"
    return 1  # Signal to re-enter Inner Loop
  fi

  if [ "$_worth_considering" -gt 0 ]; then
    log "WORTH_CONSIDERING findings (${_worth_considering}) detected, but no ACTION_REQUIRED — proceeding to PR"
  fi

  # --- PR creation phase ---
  log "--- Phase: PR creation ---"
  _pr_log="${PIPELINE_DIR}/outer-${_cycle}-pr.log"
  _pr_prompt="${PIPELINE_DIR}/.pr-prompt.md"

  cat > "$_pr_prompt" <<'PR_PROMPT'
Create a pull request for the current branch.
Follow the repository's PR workflow:
1. Check for uncommitted changes and commit them
2. Push the branch
3. Create the PR with Japanese title and body
4. Archive the plan

Use gh pr create with the standard template.

After creating the PR, write the PR URL to .harness/state/pipeline/.pr-url:
  echo "https://github.com/..." > .harness/state/pipeline/.pr-url
PR_PROMPT

  run_claude "$_pr_prompt" "$_pr_log" ""

  # Detect PR URL (3-layer defense)
  _pr_url=""

  # Layer 1: external verification via gh CLI
  _head_branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
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
    log "PR created: ${_pr_url}"
    ckpt_update --arg url "$_pr_url" '.pr_created = true | .pr_url = $url | .status = "complete"'
    _pr_event="$(jq -n --argjson c "$_cycle" --arg u "$_pr_url" '{"cycle":$c,"url":$u}')"
    report_event "pr-created" "$_pr_event"
  else
    log "PR creation step completed but URL not detected (check log for details)"
    ckpt_update ".status = \"complete\""
    report_event "pr-step" "{\"cycle\":${_cycle},\"log\":\"${_pr_log}\"}"
  fi

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
  log "Dry run: ${DRY_RUN}"
  log ""

  mkdir -p "$PIPELINE_DIR" "$EVIDENCE_DIR" "$REPORTS_DIR"

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
  "review_findings": [],
  "codex_triage": {"action_required": 0, "worth_considering": 0, "dismissed": 0},
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
      run_inner_loop "$_inner_cycle" "$_context" || _inner_result=$?

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
        _context="codex ACTION_REQUIRED — regressed from Outer Loop"
        ckpt_transition "outer" "inner" "codex ACTION_REQUIRED"
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
  log "  Checkpoint: ${PIPELINE_DIR}/checkpoint.json"
  log "  Report: ${_report_file}"
  if [ "$(ckpt_read 'pr_created')" = "true" ]; then
    log "  PR: $(ckpt_read 'pr_url')"
  fi
}

main
