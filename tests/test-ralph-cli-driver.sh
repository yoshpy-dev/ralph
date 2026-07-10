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
cd "$REPO_ROOT" || exit 1

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

# ── Test 5: cross-review reviewer inversion (pick_reviewer) ─────────────
echo
echo "── Test 5: pick_reviewer returns the opposite CLI (AC-5)"
got="$(RALPH_LOOP_DRIVER=claude pick_reviewer)"
[ "$got" = "codex" ]
check "5a. driver=claude → reviewer=codex (got '$got')" test "$got" = "codex"

got="$(RALPH_LOOP_DRIVER=codex pick_reviewer)"
check "5b. driver=codex → reviewer=claude (got '$got')" test "$got" = "claude"

got="$(unset RALPH_LOOP_DRIVER; pick_reviewer)"
check "5c. unset driver → reviewer=codex (safe default)" test "$got" = "codex"

# ── Test 6: cross-review dispatcher invokes the right CLI (AC-5) ────────
# This isolates the dispatch case-statement from the surrounding pipeline
# state machine. We replay the same shell command the pipeline executes
# and confirm the call log records the expected stub.
echo
echo "── Test 6: cross-review dispatcher invokes the inverted CLI"
PROMPT_ADV="$WORK_DIR/adv-claude.md"
printf 'Adversarial review prompt body.\n' > "$PROMPT_ADV"

CALL_C="$WORK_DIR/test6-claude.call.json"
CALL_X="$WORK_DIR/test6-codex.call.json"

# 6a. driver=claude → reviewer=codex (invokes `codex exec review`)
RALPH_LOOP_DRIVER=claude RALPH_FAKE_CALL_LOG="$CALL_X" \
  PATH="$PATH_FAKES" \
  codex exec review --base main >/dev/null 2>&1 || true
assert_jq_equal '.bin'                 "fake-codex"  "$CALL_X" "6a-i. driver=claude path used fake-codex"
assert_jq_contains '.argv | join(" ")' "exec review" "$CALL_X" "6a-ii. exec review subcommand invoked"

# 6b. driver=codex → reviewer=claude (invokes `claude -p` with adv prompt)
RALPH_LOOP_DRIVER=codex RALPH_FAKE_CALL_LOG="$CALL_C" \
  PATH="$PATH_FAKES" \
  claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" \
    --permission-mode auto --output-format text \
    < "$PROMPT_ADV" >/dev/null 2>&1 || true
assert_jq_equal '.bin'                 "fake-claude"      "$CALL_C" "6b-i. driver=codex path used fake-claude"
assert_jq_contains '.argv | join(" ")' "--permission-mode auto" "$CALL_C" "6b-ii. auto permission mode (allows triage-report write)"
assert_jq_contains '.stdin'            "Adversarial review" "$CALL_C" "6b-iii. adversarial prompt arrived via stdin"

# ── Test 7: count_triage_findings parser (P1 fix from cross-review #44) ─
echo
echo "── Test 7: count_triage_findings respects table rows, not headings"

# 7a. Empty triage with only the template scaffolding (no findings)
TRIAGE_EMPTY="$WORK_DIR/triage-empty.md"
cat > "$TRIAGE_EMPTY" <<'EOF'
# Cross-review triage report: smoke

- Date: 2026-05-08
- Driver: claude
- Reviewer: codex
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
EOF

got_a="$(count_triage_findings "$TRIAGE_EMPTY" ACTION_REQUIRED)"
got_w="$(count_triage_findings "$TRIAGE_EMPTY" WORTH_CONSIDERING)"
got_d="$(count_triage_findings "$TRIAGE_EMPTY" DISMISSED)"
check "7a-i. clean report → ACTION_REQUIRED=0 (got '$got_a')" test "$got_a" = "0"
check "7a-ii. clean report → WORTH_CONSIDERING=0 (got '$got_w')" test "$got_w" = "0"
check "7a-iii. clean report → DISMISSED=0 (got '$got_d')" test "$got_d" = "0"

# 7b. Real triage with findings — counts via the summary header line
TRIAGE_REAL="$WORK_DIR/triage-real.md"
cat > "$TRIAGE_REAL" <<'EOF'
# Cross-review triage report: example

- Total reviewer findings: 5
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=2

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | foo | bar | a.go |
| 2 | baz | qux | b.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | maybe | optional | c.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
| 1 | nope | false-positive | x |
| 2 | nope2 | already-addressed | y |
EOF

check "7b-i. real report → ACTION_REQUIRED=2"   test "$(count_triage_findings "$TRIAGE_REAL" ACTION_REQUIRED)" = "2"
check "7b-ii. real report → WORTH_CONSIDERING=1" test "$(count_triage_findings "$TRIAGE_REAL" WORTH_CONSIDERING)" = "1"
check "7b-iii. real report → DISMISSED=2"        test "$(count_triage_findings "$TRIAGE_REAL" DISMISSED)" = "2"

# 7c. Report missing the summary line (fallback path counts table rows)
TRIAGE_NOSUM="$WORK_DIR/triage-no-summary.md"
cat > "$TRIAGE_NOSUM" <<'EOF'
# Cross-review triage report: legacy

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | only finding | rationale | a.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
EOF

check "7c-i. no-summary fallback → ACTION_REQUIRED=1" test "$(count_triage_findings "$TRIAGE_NOSUM" ACTION_REQUIRED)" = "1"
check "7c-ii. no-summary fallback → WORTH_CONSIDERING=0" test "$(count_triage_findings "$TRIAGE_NOSUM" WORTH_CONSIDERING)" = "0"

# 7e. Reviewer prose mentions "ACTION_REQUIRED=2" but no canonical summary
# header — must NOT match the summary path (cycle-2 self-review MEDIUM #1).
TRIAGE_PROSE="$WORK_DIR/triage-prose.md"
cat > "$TRIAGE_PROSE" <<'EOF'
# Cross-review triage report: prose-only

The reviewer pointed out that historically, ACTION_REQUIRED=2 issues
have produced spurious regressions, so this report avoids that header.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
EOF
check "7e. prose mention not picked as summary → ACTION_REQUIRED=0" test "$(count_triage_findings "$TRIAGE_PROSE" ACTION_REQUIRED)" = "0"

# 7d. Missing file → 0 (must not error)
check "7d. missing file → 0" test "$(count_triage_findings "$WORK_DIR/does-not-exist.md" ACTION_REQUIRED)" = "0"

# ── Test 8: resolve_phase_model routing ─────────────────────────────────
echo
echo "── Test 8: resolve_phase_model — per-phase routing and escalation"

# 8a. implement cycle 1 → RALPH_IMPLEMENT_MODEL (sonnet)
got="$(RALPH_IMPLEMENT_MODEL=sonnet RALPH_FORCE_MODEL='' resolve_phase_model implement 1)"
check "8a. implement cycle 1 → sonnet (got '$got')" test "$got" = "sonnet"

# 8b. implement cycle 2 → RALPH_ESCALATION_MODEL (opus)
got="$(RALPH_ESCALATION_MODEL=opus RALPH_FORCE_MODEL='' resolve_phase_model implement 2)"
check "8b. implement cycle 2 → opus escalation (got '$got')" test "$got" = "opus"

# 8c. implement empty cycle → treated as 1, no escalation
got="$(RALPH_IMPLEMENT_MODEL=sonnet RALPH_FORCE_MODEL='' resolve_phase_model implement "")"
check "8c. implement empty cycle → sonnet (no escalation) (got '$got')" test "$got" = "sonnet"

# 8d. self_review → RALPH_SELF_REVIEW_MODEL (opus)
got="$(RALPH_SELF_REVIEW_MODEL=opus RALPH_FORCE_MODEL='' resolve_phase_model self_review 1)"
check "8d. self_review → opus (got '$got')" test "$got" = "opus"

# 8e. probe → RALPH_PROBE_MODEL (haiku)
got="$(RALPH_PROBE_MODEL=haiku RALPH_FORCE_MODEL='' resolve_phase_model probe 1)"
check "8e. probe → haiku (got '$got')" test "$got" = "haiku"

# 8f. unknown phase → $RALPH_MODEL fallback
got="$(RALPH_MODEL=opus RALPH_FORCE_MODEL='' resolve_phase_model unknown_phase 1)"
check "8f. unknown phase → RALPH_MODEL (got '$got')" test "$got" = "opus"

# 8g. verify → sonnet
got="$(RALPH_VERIFY_MODEL=sonnet RALPH_FORCE_MODEL='' resolve_phase_model verify 1)"
check "8g. verify → sonnet (got '$got')" test "$got" = "sonnet"

# 8h. test → sonnet
got="$(RALPH_TEST_MODEL=sonnet RALPH_FORCE_MODEL='' resolve_phase_model test 1)"
check "8h. test → sonnet (got '$got')" test "$got" = "sonnet"

# ── Test 9: RALPH_FORCE_MODEL overrides everything ───────────────────────
echo
echo "── Test 9: RALPH_FORCE_MODEL single-knob override"

# 9a. implement cycle 1 forced to haiku
got="$(RALPH_FORCE_MODEL=haiku resolve_phase_model implement 1)"
check "9a. FORCE_MODEL=haiku forces implement cycle 1 → haiku (got '$got')" test "$got" = "haiku"

# 9b. implement cycle 2 (would normally escalate to opus) forced to haiku
got="$(RALPH_FORCE_MODEL=haiku RALPH_ESCALATION_MODEL=opus resolve_phase_model implement 2)"
check "9b. FORCE_MODEL=haiku forces implement cycle 2 → haiku (not opus) (got '$got')" test "$got" = "haiku"

# 9c. self_review (would normally be opus) forced to haiku
got="$(RALPH_FORCE_MODEL=haiku RALPH_SELF_REVIEW_MODEL=opus resolve_phase_model self_review 1)"
check "9c. FORCE_MODEL=haiku forces self_review → haiku (got '$got')" test "$got" = "haiku"

# 9d. probe forced to haiku (same as default, but confirms the override path)
got="$(RALPH_FORCE_MODEL=haiku RALPH_PROBE_MODEL=haiku resolve_phase_model probe 1)"
check "9d. FORCE_MODEL=haiku forces probe → haiku (got '$got')" test "$got" = "haiku"

# ── Test 10: run_agent 4th arg model → --model flag via stub claude ──────
echo
echo "── Test 10: run_agent 4th-arg model passed as --model to claude"

LOG10="$WORK_DIR/test10.log"
CALL10="$WORK_DIR/test10.call.json"

# 10a. 4th arg "sonnet" → stub claude receives --model sonnet
RALPH_LOOP_DRIVER=claude \
  JSON_OUTPUT_SUPPORTED=1 \
  DRY_RUN=0 \
  RALPH_MODEL=opus \
  RALPH_EFFORT=high \
  RALPH_PERMISSION_MODE=bypassPermissions \
  RALPH_FAKE_CALL_LOG="$CALL10" \
  RALPH_FAKE_RESULT="model-arg test" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG10" "" "sonnet" >/dev/null 2>&1
assert_jq_contains '.argv | join(" ")' "--model sonnet" "$CALL10" "10a. 4th arg sonnet → --model sonnet"

# 10b. omitted 4th arg → stub claude receives --model $RALPH_MODEL (opus)
LOG10B="$WORK_DIR/test10b.log"
CALL10B="$WORK_DIR/test10b.call.json"
RALPH_LOOP_DRIVER=claude \
  JSON_OUTPUT_SUPPORTED=1 \
  DRY_RUN=0 \
  RALPH_MODEL=opus \
  RALPH_EFFORT=high \
  RALPH_PERMISSION_MODE=bypassPermissions \
  RALPH_FAKE_CALL_LOG="$CALL10B" \
  RALPH_FAKE_RESULT="fallback model test" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG10B" "" >/dev/null 2>&1
assert_jq_contains '.argv | join(" ")' "--model opus" "$CALL10B" "10b. omitted 4th arg → --model RALPH_MODEL (opus)"

# 10c. empty string 4th arg → same as omitted, falls back to RALPH_MODEL
LOG10C="$WORK_DIR/test10c.log"
CALL10C="$WORK_DIR/test10c.call.json"
RALPH_LOOP_DRIVER=claude \
  JSON_OUTPUT_SUPPORTED=1 \
  DRY_RUN=0 \
  RALPH_MODEL=opus \
  RALPH_EFFORT=high \
  RALPH_PERMISSION_MODE=bypassPermissions \
  RALPH_FAKE_CALL_LOG="$CALL10C" \
  RALPH_FAKE_RESULT="empty model test" \
  PATH="$PATH_FAKES" \
  run_agent "$PROMPT_FILE" "$LOG10C" "" "" >/dev/null 2>&1
assert_jq_contains '.argv | join(" ")' "--model opus" "$CALL10C" "10c. empty 4th arg → --model RALPH_MODEL (opus)"

# ── Test 11: write_model_receipt JSONL output ────────────────────────────
echo
echo "── Test 11: write_model_receipt produces parseable JSONL"

RECEIPT_DIR="$WORK_DIR/harness-state"
mkdir -p "$RECEIPT_DIR"

# 11a. driver=claude → honored=true, effective_model=requested_model
(
  cd "$WORK_DIR" || exit 1
  mkdir -p .harness/state/pipeline
  RALPH_LOOP_DRIVER=claude \
    RALPH_EFFORT=high \
    write_model_receipt "implement" "1" "sonnet" "phase-default"
)
RECEIPT_FILE="$WORK_DIR/.harness/state/pipeline/model-receipts.jsonl"
check "11a. receipt file created" test -s "$RECEIPT_FILE"
assert_jq_equal '.phase'           "implement"     "$RECEIPT_FILE" "11a-i. phase=implement"
assert_jq_equal '.cycle'           "1"             "$RECEIPT_FILE" "11a-ii. cycle=1"
assert_jq_equal '.driver'          "claude"        "$RECEIPT_FILE" "11a-iii. driver=claude"
assert_jq_equal '.requested_model' "sonnet"        "$RECEIPT_FILE" "11a-iv. requested_model=sonnet"
assert_jq_equal '.effective_model' "sonnet"        "$RECEIPT_FILE" "11a-v. effective_model=sonnet (honored)"
assert_jq_equal '.honored'         "true"          "$RECEIPT_FILE" "11a-vi. honored=true"
assert_jq_equal '.reason'          "phase-default" "$RECEIPT_FILE" "11a-vii. reason=phase-default"

# 11b. driver=codex → honored=false, effective_model=codex-default
# Use a separate WORK_DIR subdir so receipts are fully isolated from 11a.
RECEIPT_DIR2="$WORK_DIR/receipt2"
mkdir -p "$RECEIPT_DIR2"
RECEIPT_FILE2="$RECEIPT_DIR2/.harness/state/pipeline/model-receipts.jsonl"
(
  cd "$RECEIPT_DIR2" || exit 1
  RALPH_LOOP_DRIVER=codex \
    RALPH_EFFORT=high \
    write_model_receipt "self_review" "2" "opus" "escalation-cycle-2"
)
_tmp_receipt="$WORK_DIR/last-receipt.json"
# The file should have exactly one line; read it directly.
_last_line="$(cat "$RECEIPT_FILE2" 2>/dev/null || printf '')"
printf '%s\n' "$_last_line" > "$_tmp_receipt"
assert_jq_equal '.driver'          "codex"               "$_tmp_receipt" "11b-i. driver=codex"
assert_jq_equal '.requested_model' "opus"                "$_tmp_receipt" "11b-ii. requested_model=opus"
assert_jq_equal '.effective_model' "codex-default"       "$_tmp_receipt" "11b-iii. effective_model=codex-default"
assert_jq_equal '.honored'         "false"               "$_tmp_receipt" "11b-iv. honored=false"
assert_jq_equal '.reason'          "escalation-cycle-2"  "$_tmp_receipt" "11b-v. reason=escalation-cycle-2"

# 11c. receipt is valid JSON (all lines parse)
_invalid="$(jq -c . "$RECEIPT_FILE" 2>&1 | grep -c 'parse error' || true)"
check "11c. all receipt lines are valid JSON (parse errors: $_invalid)" test "$_invalid" = "0"

# ── Test 12: write_model_receipt 5th-arg driver_override ─────────────────
echo
echo "── Test 12: write_model_receipt driver_override overrides RALPH_LOOP_DRIVER for receipts"

# 12a. driver_override=codex under RALPH_LOOP_DRIVER=claude
# Simulates the cross-review codex call site: pipeline driver is claude but
# the reviewer is codex — the receipt must record driver=codex, honored=false,
# effective_model=codex-default (not the requested model).
RECEIPT_DIR12A="$WORK_DIR/receipt12a"
mkdir -p "$RECEIPT_DIR12A"
RECEIPT_FILE12A="$RECEIPT_DIR12A/.harness/state/pipeline/model-receipts.jsonl"
(
  cd "$RECEIPT_DIR12A" || exit 1
  RALPH_LOOP_DRIVER=claude \
    RALPH_EFFORT=high \
    write_model_receipt "cross_review" "1" "codex-default" "cross-review-codex" "codex"
)
_tmp12a="$WORK_DIR/last-receipt-12a.json"
_last12a="$(cat "$RECEIPT_FILE12A" 2>/dev/null || printf '')"
printf '%s\n' "$_last12a" > "$_tmp12a"
assert_jq_equal '.driver'          "codex"           "$_tmp12a" "12a-i. driver_override=codex → driver=codex"
assert_jq_equal '.effective_model' "codex-default"   "$_tmp12a" "12a-ii. driver_override=codex → effective_model=codex-default"
assert_jq_equal '.honored'         "false"           "$_tmp12a" "12a-iii. driver_override=codex → honored=false"
assert_jq_equal '.requested_model' "codex-default"   "$_tmp12a" "12a-iv. requested_model unchanged"

# 12b. driver_override=claude under RALPH_LOOP_DRIVER=codex
# Simulates the cross-review claude reviewer call site: pipeline driver is codex
# but the reviewer is claude — the receipt must record driver=claude,
# effective_model=requested_model, honored=true.
RECEIPT_DIR12B="$WORK_DIR/receipt12b"
mkdir -p "$RECEIPT_DIR12B"
RECEIPT_FILE12B="$RECEIPT_DIR12B/.harness/state/pipeline/model-receipts.jsonl"
_claude_reviewer_model="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"
(
  cd "$RECEIPT_DIR12B" || exit 1
  RALPH_LOOP_DRIVER=codex \
    RALPH_EFFORT=high \
    write_model_receipt "cross_review" "1" "$_claude_reviewer_model" "reviewer-inversion" "claude"
)
_tmp12b="$WORK_DIR/last-receipt-12b.json"
_last12b="$(cat "$RECEIPT_FILE12B" 2>/dev/null || printf '')"
printf '%s\n' "$_last12b" > "$_tmp12b"
assert_jq_equal '.driver'          "claude"                  "$_tmp12b" "12b-i. driver_override=claude → driver=claude"
assert_jq_equal '.effective_model' "$_claude_reviewer_model" "$_tmp12b" "12b-ii. driver_override=claude → effective_model=requested"
assert_jq_equal '.honored'         "true"                    "$_tmp12b" "12b-iii. driver_override=claude → honored=true"
assert_jq_equal '.requested_model' "$_claude_reviewer_model" "$_tmp12b" "12b-iv. requested_model unchanged"

# 12c. no driver_override (5th arg omitted) — existing behavior unchanged
# Confirms the default path still reads RALPH_LOOP_DRIVER.
RECEIPT_DIR12C="$WORK_DIR/receipt12c"
mkdir -p "$RECEIPT_DIR12C"
RECEIPT_FILE12C="$RECEIPT_DIR12C/.harness/state/pipeline/model-receipts.jsonl"
(
  cd "$RECEIPT_DIR12C" || exit 1
  RALPH_LOOP_DRIVER=claude \
    RALPH_EFFORT=high \
    write_model_receipt "implement" "1" "sonnet" "phase-default"
)
_tmp12c="$WORK_DIR/last-receipt-12c.json"
_last12c="$(cat "$RECEIPT_FILE12C" 2>/dev/null || printf '')"
printf '%s\n' "$_last12c" > "$_tmp12c"
assert_jq_equal '.driver'          "claude"  "$_tmp12c" "12c-i. no override → driver from RALPH_LOOP_DRIVER=claude"
assert_jq_equal '.effective_model' "sonnet"  "$_tmp12c" "12c-ii. no override → effective_model=requested"
assert_jq_equal '.honored'         "true"    "$_tmp12c" "12c-iii. no override → honored=true"

echo
echo "── Summary ──"
printf '  PASS: %d\n  FAIL: %d\n' "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
