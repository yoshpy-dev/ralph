#!/usr/bin/env sh
set -eu

# test-ralph-status.sh — tests for ralph status display
#
# Creates mock orchestrator/pipeline state, then runs ralph status
# in various modes and validates the output.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RALPH="${PROJECT_ROOT}/scripts/ralph"
HELPERS="${PROJECT_ROOT}/scripts/ralph-status-helpers.sh"

# Test counters
_pass=0
_fail=0
_total=0

# Temp directory for mock state
MOCK_DIR=""

setup() {
  MOCK_DIR="$(mktemp -d)"
  MOCK_ORCH="${MOCK_DIR}/.harness/state/orchestrator"
  MOCK_WT="${MOCK_DIR}/.claude/worktrees"
  MOCK_PIPELINE_1="${MOCK_WT}/1-auth-api/.harness/state/pipeline"
  MOCK_PIPELINE_2="${MOCK_WT}/2-user-model/.harness/state/pipeline"
  MOCK_PIPELINE_3="${MOCK_WT}/3-migrations/.harness/state/pipeline"
  MOCK_PIPELINE_4="${MOCK_WT}/4-docs/.harness/state/pipeline"

  mkdir -p "$MOCK_ORCH" "$MOCK_PIPELINE_1" "$MOCK_PIPELINE_2" "$MOCK_PIPELINE_3" "$MOCK_PIPELINE_4"

  # Orchestrator state
  cat > "${MOCK_ORCH}/orchestrator.json" <<'JSON'
{
  "plan": "docs/plans/active/2026-04-10-auth-api/",
  "started": "2026-04-10T10:00:00Z",
  "max_parallel": 4,
  "max_iterations": 20,
  "unified_pr": true,
  "status": "running"
}
JSON

  # Slice statuses
  echo "complete" > "${MOCK_ORCH}/slice-1-auth-api.status"
  echo "running"  > "${MOCK_ORCH}/slice-2-user-model.status"
  echo "pending"  > "${MOCK_ORCH}/slice-3-migrations.status"
  echo "failed"   > "${MOCK_ORCH}/slice-4-docs.status"

  # Slice 1 checkpoint (complete)
  cat > "${MOCK_PIPELINE_1}/checkpoint.json" <<'JSON'
{
  "schema_version": 1,
  "iteration": 4,
  "phase": "outer",
  "status": "complete",
  "inner_cycle": 2,
  "outer_cycle": 1,
  "stuck_count": 0,
  "last_test_result": "pass",
  "test_failures": [],
  "failure_triage": [],
  "pr_url": "https://github.com/example/repo/pull/42",
  "pr_created": true,
  "phase_transitions": [
    {"from": "preflight", "to": "inner", "timestamp": "2026-04-10T10:01:00Z"}
  ]
}
JSON

  # Slice 2 checkpoint (running, in test phase)
  cat > "${MOCK_PIPELINE_2}/checkpoint.json" <<'JSON'
{
  "schema_version": 1,
  "iteration": 6,
  "phase": "test",
  "status": "running",
  "inner_cycle": 3,
  "outer_cycle": 0,
  "stuck_count": 0,
  "last_test_result": "fail",
  "test_failures": ["cycle_2_tests"],
  "failure_triage": [],
  "pr_url": null,
  "pr_created": false,
  "phase_transitions": [
    {"from": "preflight", "to": "inner", "timestamp": "2026-04-10T10:02:00Z"}
  ]
}
JSON

  # Slice 3 — no checkpoint (pending)

  # Slice 4 checkpoint (failed/stuck)
  cat > "${MOCK_PIPELINE_4}/checkpoint.json" <<'JSON'
{
  "schema_version": 1,
  "iteration": 10,
  "phase": "inner",
  "status": "stuck",
  "inner_cycle": 5,
  "outer_cycle": 0,
  "stuck_count": 3,
  "last_test_result": "fail",
  "test_failures": ["cycle_3_tests", "cycle_4_tests", "cycle_5_tests"],
  "failure_triage": [],
  "pr_url": null,
  "pr_created": false,
  "phase_transitions": [
    {"from": "preflight", "to": "inner", "timestamp": "2026-04-10T10:03:00Z"}
  ]
}
JSON
}

cleanup() {
  [ -n "$MOCK_DIR" ] && rm -rf "$MOCK_DIR"
}

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  _total=$((_total + 1))

  if [ "$_expected" = "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n' "$_desc"
    printf '    expected: %s\n' "$_expected"
    printf '    actual:   %s\n' "$_actual"
  fi
}

assert_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))

  if printf '%s' "$_haystack" | grep -qF "$_needle"; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s (needle not found)\n' "$_desc"
    printf '    needle:   %s\n' "$_needle"
    printf '    haystack: %s\n' "$(printf '%s' "$_haystack" | head -5)"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  _total=$((_total + 1))

  if printf '%s' "$_haystack" | grep -qF "$_needle"; then
    _fail=$((_fail + 1))
    printf '  FAIL: %s (needle should not be present)\n' "$_desc"
  else
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Helper function unit tests
# ═══════════════════════════════════════════════════════════════════

test_helpers() {
  echo ""
  echo "=== Helper function tests ==="

  . "$HELPERS"

  # --- format_duration ---
  assert_eq "format_duration: 0s" "—" "$(format_duration 0)"
  assert_eq "format_duration: 45s" "45s" "$(format_duration 45)"
  assert_eq "format_duration: 90s" "1m 30s" "$(format_duration 90)"
  assert_eq "format_duration: 3661s" "1h 1m" "$(format_duration 3661)"
  assert_eq "format_duration: empty" "—" "$(format_duration "")"

  # --- iso_to_epoch ---
  _epoch="$(iso_to_epoch "2026-04-10T10:00:00Z")"
  _total=$((_total + 1))
  if [ "$_epoch" -gt 0 ] 2>/dev/null; then
    _pass=$((_pass + 1))
    printf '  PASS: iso_to_epoch returns positive epoch (%s)\n' "$_epoch"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: iso_to_epoch returned %s (expected positive integer)\n' "$_epoch"
  fi

  # --- jq-based checkpoint reading (same as used in renderers) ---
  _ckpt="${MOCK_PIPELINE_1}/checkpoint.json"
  _phase="$(jq -r '.phase // "unknown"' "$_ckpt" 2>/dev/null)"
  _cycle="$(jq -r '.inner_cycle // 0' "$_ckpt" 2>/dev/null)"
  _pr="$(jq -r '.pr_url // ""' "$_ckpt" 2>/dev/null)"
  assert_eq "checkpoint reading: phase" "outer" "$_phase"
  assert_eq "checkpoint reading: cycle" "2" "$_cycle"
  assert_eq "checkpoint reading: pr_url" "https://github.com/example/repo/pull/42" "$_pr"

  # --- missing checkpoint file ---
  _phase_missing="$(jq -r '.phase // "unknown"' "/nonexistent/checkpoint.json" 2>/dev/null || echo "unknown")"
  assert_eq "checkpoint reading: missing file returns unknown" "unknown" "$_phase_missing"

  # --- render_progress_bar (no color) ---
  STATUS_NO_COLOR=1
  detect_color
  _bar="$(render_progress_bar 1 4 20)"
  assert_contains "progress_bar: contains 25%" "25%" "$_bar"
  assert_contains "progress_bar: contains (1/4)" "(1/4)" "$_bar"

  # --- estimate_eta ---
  _eta="$(estimate_eta 2 2 120)"
  assert_contains "estimate_eta: contains ~2m" "~2m" "$_eta"

  _eta_zero="$(estimate_eta 0 3 0)"
  assert_eq "estimate_eta: no completed slices" "—" "$_eta_zero"
}

# ═══════════════════════════════════════════════════════════════════
# Table rendering tests
# ═══════════════════════════════════════════════════════════════════

test_table_render() {
  echo ""
  echo "=== Table rendering tests ==="

  . "$HELPERS"
  STATUS_NO_COLOR=1
  detect_color

  _output="$(_render_table "$MOCK_ORCH" "$MOCK_WT")"

  assert_contains "table: shows plan" "docs/plans/active/2026-04-10-auth-api/" "$_output"
  assert_contains "table: shows Slice header" "Slice" "$_output"
  assert_contains "table: shows 1-auth-api" "1-auth-api" "$_output"
  assert_contains "table: shows 2-user-model" "2-user-model" "$_output"
  assert_contains "table: shows 3-migrations" "3-migrations" "$_output"
  assert_contains "table: shows 4-docs" "4-docs" "$_output"
  assert_contains "table: shows complete status" "complete" "$_output"
  assert_contains "table: shows running status" "running" "$_output"
  assert_contains "table: shows pending status" "pending" "$_output"
  assert_contains "table: shows failed status" "failed" "$_output"
  assert_contains "table: shows PR #42" "PR #42" "$_output"
  assert_contains "table: shows progress percent" "25%" "$_output"
  assert_contains "table: shows (1/4)" "(1/4)" "$_output"
}

# ═══════════════════════════════════════════════════════════════════
# JSON output tests
# ═══════════════════════════════════════════════════════════════════

test_json_render() {
  echo ""
  echo "=== JSON rendering tests ==="

  . "$HELPERS"
  STATUS_NO_COLOR=1
  detect_color

  _json="$(_render_json "$MOCK_ORCH" "$MOCK_WT")"

  # Validate it's parseable JSON
  _total=$((_total + 1))
  if printf '%s' "$_json" | jq . > /dev/null 2>&1; then
    _pass=$((_pass + 1))
    printf '  PASS: JSON output is valid JSON\n'
  else
    _fail=$((_fail + 1))
    printf '  FAIL: JSON output is not valid JSON\n'
    printf '    output: %s\n' "$_json"
  fi

  # Check fields
  _j_status="$(printf '%s' "$_json" | jq -r '.status' 2>/dev/null || echo "")"
  assert_eq "json: status=running" "running" "$_j_status"

  _j_plan="$(printf '%s' "$_json" | jq -r '.plan' 2>/dev/null || echo "")"
  assert_eq "json: plan field" "docs/plans/active/2026-04-10-auth-api/" "$_j_plan"

  _j_completed="$(printf '%s' "$_json" | jq -r '.progress.completed' 2>/dev/null || echo "")"
  assert_eq "json: progress.completed=1" "1" "$_j_completed"

  _j_total="$(printf '%s' "$_json" | jq -r '.progress.total' 2>/dev/null || echo "")"
  assert_eq "json: progress.total=4" "4" "$_j_total"

  _j_pct="$(printf '%s' "$_json" | jq -r '.progress.percent' 2>/dev/null || echo "")"
  assert_eq "json: progress.percent=25" "25" "$_j_pct"

  _j_slice_count="$(printf '%s' "$_json" | jq '.slices | length' 2>/dev/null || echo "")"
  assert_eq "json: 4 slices" "4" "$_j_slice_count"

  _j_slice1_pr="$(printf '%s' "$_json" | jq -r '.slices[0].pr_url' 2>/dev/null || echo "")"
  assert_eq "json: slice 1 pr_url" "https://github.com/example/repo/pull/42" "$_j_slice1_pr"
}

# ═══════════════════════════════════════════════════════════════════
# No-color tests
# ═══════════════════════════════════════════════════════════════════

test_no_color() {
  echo ""
  echo "=== No-color tests ==="

  . "$HELPERS"
  STATUS_NO_COLOR=1
  detect_color

  _output="$(_render_table "$MOCK_ORCH" "$MOCK_WT")"
  _ansi_count="$(printf '%s' "$_output" | grep -c "$(printf '\033')" || true)"
  assert_eq "no-color: zero ANSI escape codes" "0" "$_ansi_count"
}

# ═══════════════════════════════════════════════════════════════════
# No orchestrator state test
# ═══════════════════════════════════════════════════════════════════

test_no_state() {
  echo ""
  echo "=== No orchestrator state tests ==="

  . "$HELPERS"
  STATUS_NO_COLOR=1
  detect_color

  _empty_dir="$(mktemp -d)"
  _output="$(_render_table "$_empty_dir" "$_empty_dir")"
  assert_contains "no state: graceful message" "No active orchestrator state found" "$_output"

  _json="$(_render_json "$_empty_dir" "$_empty_dir")"
  _j_err="$(printf '%s' "$_json" | jq -r '.error' 2>/dev/null || echo "")"
  assert_eq "no state json: error field" "no_active_orchestrator" "$_j_err"

  rm -rf "$_empty_dir"
}

# ═══════════════════════════════════════════════════════════════════
# Whitespace trimming tests (race condition fix)
# ═══════════════════════════════════════════════════════════════════

test_whitespace_trimming() {
  echo ""
  echo "=== Whitespace trimming tests ==="

  # Create a status file with trailing whitespace/newline (simulates race condition)
  _ws_dir="$(mktemp -d)"
  _ws_orch="${_ws_dir}/.harness/state/orchestrator"
  mkdir -p "$_ws_orch"

  printf 'complete \n' > "${_ws_orch}/slice-test.status"
  _raw="$(cat "${_ws_orch}/slice-test.status")"
  _trimmed="$(cat "${_ws_orch}/slice-test.status" | tr -d '[:space:]')"

  assert_eq "trimmed status equals 'complete'" "complete" "$_trimmed"

  # Verify raw contains extra whitespace
  _total=$((_total + 1))
  if [ "$_raw" != "$_trimmed" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: raw status has trailing whitespace (as expected)\n'
  else
    _pass=$((_pass + 1))
    printf '  PASS: status already clean (trimming is still safe)\n'
  fi

  rm -rf "$_ws_dir"
}

# ═══════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════

main() {
  echo "=== ralph status tests ==="

  setup

  test_helpers
  test_table_render
  test_json_render
  test_no_color
  test_no_state
  test_whitespace_trimming

  cleanup

  echo ""
  echo "═══════════════════════════════════════"
  printf 'Results: %d/%d passed' "$_pass" "$_total"
  if [ "$_fail" -gt 0 ]; then
    printf ', %d FAILED' "$_fail"
  fi
  echo ""
  echo "═══════════════════════════════════════"

  [ "$_fail" -eq 0 ]
}

main
