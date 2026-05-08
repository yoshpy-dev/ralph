#!/usr/bin/env bash
# test-ralph-cli-driver.sh — end-to-end behavioural tests for the Ralph
# Loop CLI driver wrapper (`scripts/ralph-cli-driver.sh`).
#
# These run the real wrapper against fake-claude / fake-codex stubs on PATH
# so we exercise actual command assembly, stdin piping, and the
# <log>/<log>.json synthesis — not a mock of the wrapper. AC-1, AC-3 of
# issue #44.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

STUBS="$REPO_ROOT/tests/fixtures/cli-stubs"
WORK_DIR="$(mktemp -d -t ralph-cli-driver-XXXXXX)"
trap 'rm -rf "$WORK_DIR"' EXIT

# Standalone PATH for the fixture stubs. The stubs are named `claude` and
# `codex` so PATH lookup resolves the wrapper's invocations to them. We
# keep the system PATH for jq/git/etc.
PATH_ORIG="$PATH"
PATH_FAKES="$STUBS:$PATH_ORIG"

PASS=0
FAIL=0

check() {
  local label="$1"
  shift
  if "$@"; then
    printf '  PASS  %s\n' "$label"
    PASS=$((PASS + 1))
  else
    printf '  FAIL  %s\n' "$label"
    FAIL=$((FAIL + 1))
  fi
}

assert_jq_equal() {
  # assert_jq_equal <jq-filter> <expected> <file> <label>
  local filter="$1" expected="$2" file="$3" label="$4"
  local actual
  actual="$(jq -r "$filter" "$file" 2>/dev/null || printf '<jq error>')"
  if [ "$actual" = "$expected" ]; then
    printf '  PASS  %s\n' "$label"
    PASS=$((PASS + 1))
  else
    printf '  FAIL  %s — expected %q got %q\n' "$label" "$expected" "$actual"
    FAIL=$((FAIL + 1))
  fi
}

assert_jq_contains() {
  local filter="$1" needle="$2" file="$3" label="$4"
  local actual
  actual="$(jq -r "$filter" "$file" 2>/dev/null || printf '')"
  case "$actual" in
    *"$needle"*)
      printf '  PASS  %s\n' "$label"
      PASS=$((PASS + 1))
      ;;
    *)
      printf '  FAIL  %s — expected %q in %q\n' "$label" "$needle" "$actual"
      FAIL=$((FAIL + 1))
      ;;
  esac
}

# ── Common setup ─────────────────────────────────────────────────────────
PROMPT_FILE="$WORK_DIR/prompt.md"
printf 'Hello fake CLI. Phase 2 driver test.\n' > "$PROMPT_FILE"

# Source the wrapper and set defaults the wrapper expects from the caller.
# shellcheck source=/dev/null
. "$REPO_ROOT/scripts/ralph-config.sh"
# shellcheck source=/dev/null
. "$REPO_ROOT/scripts/ralph-cli-driver.sh"

# ── Test 1: claude driver, JSON mode ────────────────────────────────────
echo
echo "── Test 1: driver=claude / JSON mode (preserves prior run_claude semantics)"
LOG="$WORK_DIR/test1.log"
CALL="$WORK_DIR/test1.call.json"
RALPH_LOOP_DRIVER=claude \
  JSON_OUTPUT_SUPPORTED=1 \
  DRY_RUN=0 \
  RALPH_MODEL=fake-model \
  RALPH_EFFORT=fake-effort \
  RALPH_PERMISSION_MODE=auto \
  RALPH_FAKE_CALL_LOG="$CALL" \
  RALPH_FAKE_RESULT="claude completed step" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG" "" >/dev/null 2>&1

check "1a. <log> file written"            test -s "$LOG"
check "1b. <log>.json file written"        test -s "$LOG.json"
check "1c. fake-claude was invoked"        test -s "$CALL"
assert_jq_equal '.bin'                     "fake-claude"            "$CALL" "1d. correct binary"
assert_jq_equal '.output_format'           "json"                   "$CALL" "1e. JSON output mode"
assert_jq_contains '.argv | join(" ")'     "--model fake-model"     "$CALL" "1f. model flag passed"
assert_jq_contains '.argv | join(" ")'     "--permission-mode auto" "$CALL" "1g. permission mode passed"
assert_jq_contains '.stdin'                "Phase 2 driver test"    "$CALL" "1h. prompt arrived via stdin"
assert_jq_equal '.result'                  "claude completed step"  "$LOG.json" "1i. <log>.json carries .result"
assert_jq_equal '.session_id'              "fake-session-id"        "$LOG.json" "1j. <log>.json carries .session_id"

# ── Test 2: codex driver, sandbox + approval + last-message synthesis ───
echo
echo "── Test 2: driver=codex / synthesises <log>.json from --output-last-message"
LOG="$WORK_DIR/test2.log"
CALL="$WORK_DIR/test2.call.json"
RALPH_LOOP_DRIVER=codex \
  JSON_OUTPUT_SUPPORTED=0 \
  DRY_RUN=0 \
  RALPH_CODEX_SANDBOX=workspace-write \
  RALPH_CODEX_APPROVAL_POLICY=on-failure \
  RALPH_FAKE_CALL_LOG="$CALL" \
  RALPH_FAKE_LAST_MESSAGE="codex final message" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG" "" >/dev/null 2>&1

check "2a. <log> file written"             test -s "$LOG"
check "2b. <log>.json file written"        test -s "$LOG.json"
check "2c. fake-codex was invoked"         test -s "$CALL"
assert_jq_equal '.bin'                     "fake-codex"             "$CALL" "2d. correct binary"
assert_jq_contains '.argv | join(" ")'     "exec"                   "$CALL" "2e. codex exec subcommand"
assert_jq_contains '.argv | join(" ")'     "-s workspace-write"     "$CALL" "2f. sandbox flag passed"
assert_jq_contains '.argv | join(" ")'     "approval_policy="       "$CALL" "2g. approval_policy override"
assert_jq_contains '.argv | join(" ")'     "--output-last-message"  "$CALL" "2h. --output-last-message present"
assert_jq_contains '.argv | tostring'      ".last"                  "$CALL" "2i. last-message path ends in .last"
assert_jq_contains '.stdin'                "Phase 2 driver test"    "$CALL" "2j. prompt arrived via stdin"
assert_jq_equal '.session_id'              "null"                   "$LOG.json" "2k. session_id null on codex"
assert_jq_contains '.result'               "codex final message"    "$LOG.json" "2l. .result captured from .last"

# ── Test 3: codex driver, missing .last triggers fallback ───────────────
echo
echo "── Test 3: driver=codex / missing --output-last-message file → empty fallback"
LOG="$WORK_DIR/test3.log"
CALL="$WORK_DIR/test3.call.json"
RALPH_LOOP_DRIVER=codex \
  JSON_OUTPUT_SUPPORTED=0 \
  DRY_RUN=0 \
  RALPH_CODEX_SANDBOX=workspace-write \
  RALPH_CODEX_APPROVAL_POLICY=on-failure \
  RALPH_FAKE_CALL_LOG="$CALL" \
  RALPH_FAKE_LAST_MESSAGE="" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG" "" >/dev/null 2>&1

check "3a. <log>.json still written"       test -s "$LOG.json"
assert_jq_equal '.session_id'              "null"                   "$LOG.json" "3b. session_id null"

# ── Test 4: dry-run short-circuit for both drivers ──────────────────────
echo
echo "── Test 4: DRY_RUN=1 short-circuits both drivers without invoking the CLI"
LOG="$WORK_DIR/test4-claude.log"
CALL="$WORK_DIR/test4.call.json"
rm -f "$CALL"
RALPH_LOOP_DRIVER=claude DRY_RUN=1 RALPH_FAKE_CALL_LOG="$CALL" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG" "" >/dev/null
check "4a. claude dry-run wrote <log>"     test -s "$LOG"
check "4b. claude dry-run wrote <log>.json" test -s "$LOG.json"
check "4c. claude dry-run did NOT invoke fake-claude" test ! -s "$CALL"

LOG="$WORK_DIR/test4-codex.log"
RALPH_LOOP_DRIVER=codex DRY_RUN=1 RALPH_FAKE_CALL_LOG="$CALL" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG" "" >/dev/null
check "4d. codex dry-run wrote <log>"      test -s "$LOG"
check "4e. codex dry-run wrote <log>.json" test -s "$LOG.json"
check "4f. codex dry-run did NOT invoke fake-codex" test ! -s "$CALL"

echo
echo "── Summary ──"
printf '  PASS: %d\n  FAIL: %d\n' "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
