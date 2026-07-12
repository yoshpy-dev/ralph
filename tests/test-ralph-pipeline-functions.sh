#!/usr/bin/env bash
# test-ralph-pipeline-functions.sh — unit tests for ralph-pipeline.sh functions
# Sources the script (via source guard) to test functions in isolation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PIPELINE_SCRIPT="${PROJECT_ROOT}/scripts/ralph-pipeline.sh"

PASS=0
FAIL=0

pass() {
  PASS=$((PASS + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '  FAIL: %s\n' "$1"
  if [ -n "${2:-}" ]; then
    printf '    detail: %s\n' "$2"
  fi
}

assert_eq() {
  _label="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    pass "$_label"
  else
    fail "$_label" "expected='${_expected}' actual='${_actual}'"
  fi
}

assert_file_contains() {
  _label="$1"
  _needle="$2"
  _file="$3"
  if grep -qF "$_needle" "$_file" 2>/dev/null; then
    pass "$_label"
  else
    fail "$_label" "needle='${_needle}' not found in ${_file}"
  fi
}

assert_exits_nonzero() {
  _label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    fail "$_label" "expected non-zero exit but got 0"
  else
    pass "$_label"
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Sandbox setup
# ═══════════════════════════════════════════════════════════════════

SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/ralph-pipeline-fn.XXXXXX")"
cleanup() { rm -rf "$SANDBOX"; }
trap cleanup EXIT

# Set up a minimal git repo in the sandbox
(
  cd "$SANDBOX"
  git init -q -b main
  git config user.email test@example.com
  git config user.name "Test User"
  printf 'seed\n' > README.md
  git add README.md
  git commit -q -m 'chore: seed'
  mkdir -p .harness/state/pipeline docs/evidence docs/reports
)

# Source the pipeline script from the PROJECT_ROOT directory so SCRIPT_DIR
# resolves to scripts/ and ralph-config.sh / ralph-common.sh are found.
# The source guard (BASH_SOURCE[0] != 0) prevents main() from running.
# Then switch to sandbox for function tests that need a real git repo/dirs.
# shellcheck disable=SC1090
source "$PIPELINE_SCRIPT"

# Now switch to sandbox for the actual function tests.
# Override PIPELINE_DIR so ckpt_update writes into the sandbox, not the repo.
cd "$SANDBOX"
PIPELINE_DIR="${SANDBOX}/.harness/state/pipeline"

printf '==> ralph-pipeline.sh function tests\n'

# ═══════════════════════════════════════════════════════════════════
# ckpt_update tests
# ═══════════════════════════════════════════════════════════════════

printf '\n--- ckpt_update ---\n'

# Test 1: creates checkpoint.json from scratch when absent
rm -f "${PIPELINE_DIR}/checkpoint.json"
ckpt_update '.phase = "inner"'
if [ -f "${PIPELINE_DIR}/checkpoint.json" ]; then
  pass "ckpt_update creates checkpoint.json when absent"
else
  fail "ckpt_update creates checkpoint.json when absent"
fi

# Test 2: written value is readable back
_phase="$(jq -r '.phase // empty' "${PIPELINE_DIR}/checkpoint.json")"
assert_eq "ckpt_update sets phase field" "inner" "$_phase"

# Test 3: subsequent update merges without overwriting unrelated fields
ckpt_update '.inner_cycle = 3'
_phase2="$(jq -r '.phase // empty' "${PIPELINE_DIR}/checkpoint.json")"
_cycle="$(jq -r '.inner_cycle // empty' "${PIPELINE_DIR}/checkpoint.json")"
assert_eq "ckpt_update preserves existing phase field" "inner" "$_phase2"
assert_eq "ckpt_update adds inner_cycle field" "3" "$_cycle"

# Test 4: temp-file swap atomicity — original not corrupted on jq failure
_orig="$(cat "${PIPELINE_DIR}/checkpoint.json")"
# Pass a deliberately malformed jq filter; ckpt_update should fail and leave original intact
if ckpt_update 'INVALID_FILTER !!!' 2>/dev/null; then
  fail "ckpt_update with malformed filter should exit non-zero"
else
  pass "ckpt_update with malformed filter exits non-zero"
fi
_after="$(cat "${PIPELINE_DIR}/checkpoint.json")"
assert_eq "ckpt_update with malformed filter leaves checkpoint intact" "$_orig" "$_after"

# Test 5: --arg flag forwarding
ckpt_update --arg url "https://example.com/pr/1" '.pr_url = $url'
_url="$(jq -r '.pr_url // empty' "${PIPELINE_DIR}/checkpoint.json")"
assert_eq "ckpt_update forwards --arg flags to jq" "https://example.com/pr/1" "$_url"

# Test 6: value with special characters (spaces, quotes) survives round-trip
ckpt_update --arg msg "feat: add 'quoted' value" '.last_msg = $msg'
_msg="$(jq -r '.last_msg // empty' "${PIPELINE_DIR}/checkpoint.json")"
assert_eq "ckpt_update handles special characters in values" "feat: add 'quoted' value" "$_msg"

printf '\n==> ralph-pipeline.sh function tests: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
