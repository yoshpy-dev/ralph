#!/usr/bin/env sh
set -eu

# test-ralph-config.sh — tests for ralph-config.sh shared configuration module

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG="${PROJECT_ROOT}/scripts/ralph-config.sh"

_pass=0
_fail=0
_total=0

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

assert_exits_nonzero() {
  _desc="$1"
  shift
  _total=$((_total + 1))
  if "$@" >/dev/null 2>&1; then
    _fail=$((_fail + 1))
    printf '  FAIL: %s (should have exited non-zero)\n' "$_desc"
  else
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Default values
# ═══════════════════════════════════════════════════════════════════

test_defaults() {
  echo ""
  echo "=== Default value tests ==="

  # Source in a subshell to avoid polluting this shell
  _model="$(unset RALPH_MODEL; . "$CONFIG"; echo "$RALPH_MODEL")"
  assert_eq "default RALPH_MODEL" "opus" "$_model"

  _effort="$(unset RALPH_EFFORT; . "$CONFIG"; echo "$RALPH_EFFORT")"
  assert_eq "default RALPH_EFFORT" "high" "$_effort"

  _perm="$(unset RALPH_PERMISSION_MODE; . "$CONFIG"; echo "$RALPH_PERMISSION_MODE")"
  assert_eq "default RALPH_PERMISSION_MODE" "bypassPermissions" "$_perm"

  _max_iter="$(unset RALPH_MAX_ITERATIONS; . "$CONFIG"; echo "$RALPH_MAX_ITERATIONS")"
  assert_eq "default RALPH_MAX_ITERATIONS" "20" "$_max_iter"

  _max_inner="$(unset RALPH_MAX_INNER_CYCLES; . "$CONFIG"; echo "$RALPH_MAX_INNER_CYCLES")"
  assert_eq "default RALPH_MAX_INNER_CYCLES" "10" "$_max_inner"

  _max_outer="$(unset RALPH_MAX_OUTER_CYCLES; . "$CONFIG"; echo "$RALPH_MAX_OUTER_CYCLES")"
  assert_eq "default RALPH_MAX_OUTER_CYCLES" "3" "$_max_outer"

  _max_repair="$(unset RALPH_MAX_REPAIR_ATTEMPTS; . "$CONFIG"; echo "$RALPH_MAX_REPAIR_ATTEMPTS")"
  assert_eq "default RALPH_MAX_REPAIR_ATTEMPTS" "5" "$_max_repair"

  _max_par="$(unset RALPH_MAX_PARALLEL; . "$CONFIG"; echo "$RALPH_MAX_PARALLEL")"
  assert_eq "default RALPH_MAX_PARALLEL" "4" "$_max_par"

  _timeout="$(unset RALPH_SLICE_TIMEOUT; . "$CONFIG"; echo "$RALPH_SLICE_TIMEOUT")"
  assert_eq "default RALPH_SLICE_TIMEOUT" "1800" "$_timeout"
}

# ═══════════════════════════════════════════════════════════════════
# Environment variable override
# ═══════════════════════════════════════════════════════════════════

test_env_override() {
  echo ""
  echo "=== Environment variable override tests ==="

  # Use separate assignment statements (not inline VAR=value . cmd) for portability
  # In bash with set -u, inline assignments may not persist after special builtins
  _model="$(RALPH_MODEL=sonnet; . "$CONFIG"; echo "$RALPH_MODEL")"
  assert_eq "override RALPH_MODEL=sonnet" "sonnet" "$_model"

  _effort="$(RALPH_EFFORT=low; . "$CONFIG"; echo "$RALPH_EFFORT")"
  assert_eq "override RALPH_EFFORT=low" "low" "$_effort"

  _max_iter="$(RALPH_MAX_ITERATIONS=50; . "$CONFIG"; echo "$RALPH_MAX_ITERATIONS")"
  assert_eq "override RALPH_MAX_ITERATIONS=50" "50" "$_max_iter"

  _timeout="$(RALPH_SLICE_TIMEOUT=3600; . "$CONFIG"; echo "$RALPH_SLICE_TIMEOUT")"
  assert_eq "override RALPH_SLICE_TIMEOUT=3600" "3600" "$_timeout"
}

# ═══════════════════════════════════════════════════════════════════
# Numeric validation
# ═══════════════════════════════════════════════════════════════════

test_validate_numeric() {
  echo ""
  echo "=== Numeric validation tests ==="

  # Valid numbers should pass
  _total=$((_total + 1))
  if (. "$CONFIG"; validate_numeric "test" "42") 2>/dev/null; then
    _pass=$((_pass + 1))
    printf '  PASS: validate_numeric accepts 42\n'
  else
    _fail=$((_fail + 1))
    printf '  FAIL: validate_numeric rejected 42\n'
  fi

  _total=$((_total + 1))
  if (. "$CONFIG"; validate_numeric "test" "1") 2>/dev/null; then
    _pass=$((_pass + 1))
    printf '  PASS: validate_numeric accepts 1\n'
  else
    _fail=$((_fail + 1))
    printf '  FAIL: validate_numeric rejected 1\n'
  fi

  # Invalid values should fail
  assert_exits_nonzero "validate_numeric rejects 'abc'" sh -c ". '$CONFIG'; validate_numeric test abc"
  assert_exits_nonzero "validate_numeric rejects empty" sh -c ". '$CONFIG'; validate_numeric test ''"
  assert_exits_nonzero "validate_numeric rejects negative" sh -c ". '$CONFIG'; validate_numeric test -5"
  assert_exits_nonzero "validate_numeric rejects 0" sh -c ". '$CONFIG'; validate_numeric test 0"
  assert_exits_nonzero "validate_numeric rejects float" sh -c ". '$CONFIG'; validate_numeric test 3.14"
  assert_exits_nonzero "validate_numeric rejects mixed" sh -c ". '$CONFIG'; validate_numeric test 10abc"
}

# ═══════════════════════════════════════════════════════════════════
# validate_all_numeric
# ═══════════════════════════════════════════════════════════════════

test_validate_all() {
  echo ""
  echo "=== validate_all_numeric tests ==="

  _total=$((_total + 1))
  if (. "$CONFIG"; validate_all_numeric) 2>/dev/null; then
    _pass=$((_pass + 1))
    printf '  PASS: validate_all_numeric passes with defaults\n'
  else
    _fail=$((_fail + 1))
    printf '  FAIL: validate_all_numeric failed with defaults\n'
  fi

  assert_exits_nonzero "validate_all_numeric rejects bad RALPH_MAX_ITERATIONS" \
    sh -c "RALPH_MAX_ITERATIONS=abc . '$CONFIG'; validate_all_numeric"
}

# ═══════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════

main() {
  echo "=== ralph-config.sh tests ==="

  test_defaults
  test_env_override
  test_validate_numeric
  test_validate_all

  echo ""
  echo "========================================="
  printf 'Results: %d/%d passed' "$_pass" "$_total"
  if [ "$_fail" -gt 0 ]; then
    printf ', %d FAILED' "$_fail"
  fi
  echo ""
  echo "========================================="

  [ "$_fail" -eq 0 ]
}

main
