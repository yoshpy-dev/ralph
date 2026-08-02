#!/usr/bin/env sh
# shellcheck disable=SC1090  # the whole point is sourcing $CONFIG dynamically
set -eu

# test-ralph-config.sh — tests for ralph-config.sh shared configuration module.
#
# ralph-config.sh's [loop]/[pipeline] defaults (RALPH_LOOP_*, per-phase
# RALPH_*_MODEL, RALPH_MAX_* iteration caps) were removed along with the
# Ralph Loop execution system. The surviving surface is the standard-flow
# pipeline cycle cap (RALPH_STANDARD_MAX_PIPELINE_CYCLES), the
# claude-as-reviewer model fallback (RALPH_CLAUDE_REVIEWER_MODEL), and the
# [org] envelope lock-step vars.

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
  _standard_cycles="$(unset RALPH_STANDARD_MAX_PIPELINE_CYCLES; . "$CONFIG"; echo "$RALPH_STANDARD_MAX_PIPELINE_CYCLES")"
  assert_eq "default RALPH_STANDARD_MAX_PIPELINE_CYCLES" "2" "$_standard_cycles"

  _reviewer="$(unset RALPH_CLAUDE_REVIEWER_MODEL; . "$CONFIG"; echo "$RALPH_CLAUDE_REVIEWER_MODEL")"
  assert_eq "default RALPH_CLAUDE_REVIEWER_MODEL" "opus" "$_reviewer"
}

# ═══════════════════════════════════════════════════════════════════
# Environment variable override
# ═══════════════════════════════════════════════════════════════════

test_env_override() {
  echo ""
  echo "=== Environment variable override tests ==="

  # Use separate assignment statements (not inline VAR=value . cmd) for portability
  # In bash with set -u, inline assignments may not persist after special builtins
  _standard_cycles="$(RALPH_STANDARD_MAX_PIPELINE_CYCLES=5; . "$CONFIG"; echo "$RALPH_STANDARD_MAX_PIPELINE_CYCLES")"
  assert_eq "override RALPH_STANDARD_MAX_PIPELINE_CYCLES=5" "5" "$_standard_cycles"

  _reviewer="$(RALPH_CLAUDE_REVIEWER_MODEL=sonnet; . "$CONFIG"; echo "$RALPH_CLAUDE_REVIEWER_MODEL")"
  assert_eq "override RALPH_CLAUDE_REVIEWER_MODEL=sonnet" "sonnet" "$_reviewer"
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

  assert_exits_nonzero "validate_all_numeric rejects bad RALPH_STANDARD_MAX_PIPELINE_CYCLES" \
    sh -c "RALPH_STANDARD_MAX_PIPELINE_CYCLES=abc . '$CONFIG'; validate_all_numeric"

  assert_exits_nonzero "validate_all_numeric rejects zero RALPH_STANDARD_MAX_PIPELINE_CYCLES" \
    sh -c "RALPH_STANDARD_MAX_PIPELINE_CYCLES=0 . '$CONFIG'; validate_all_numeric"
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
