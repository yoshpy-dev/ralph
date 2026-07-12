#!/usr/bin/env sh
# tests/test-insights-append.sh — AC1 coverage: insights-append.sh appends
# schema-v1-valid JSON lines; rejects invalid inputs with exit != 0 and
# produces no file; enum validation enforced; counts land in findings/triage
# objects; --events-dir override works.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APPEND_SH="${REPO_ROOT}/scripts/insights-append.sh"

PASS=0
FAIL=0

pass() {
  PASS=$((PASS + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_eq() {
  _name="$1"
  _want="$2"
  _got="$3"
  if [ "$_want" = "$_got" ]; then
    pass "$_name"
  else
    fail "$_name"
    printf '    want: %s\n' "$_want"
    printf '     got: %s\n' "$_got"
  fi
}

assert_file_exists() {
  _name="$1"
  _path="$2"
  if [ -s "$_path" ]; then
    pass "$_name"
  else
    fail "$_name"
    printf '    missing or empty: %s\n' "$_path"
  fi
}

assert_file_not_exists() {
  _name="$1"
  _path="$2"
  # Check the directory for any file matching the pattern (use glob)
  _dir="$(dirname "$_path")"
  _base="$(basename "$_path")"
  if [ -d "$_dir" ] && ls "${_dir}/${_base}" >/dev/null 2>&1; then
    fail "$_name"
    printf '    file should not exist but does: %s/%s\n' "$_dir" "$_base"
  else
    pass "$_name"
  fi
}

OLD_DIR="$(pwd)"

# ── Case 1: valid append → line parses with jq and has expected fields ────────
printf '\n==> Case 1: valid append — single line, all required fields\n'

TMP1="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP1"' EXIT INT TERM

DATE="$(date -u '+%Y-%m-%d')"
OUTFILE="${TMP1}/${DATE}-my-task.jsonl"

bash "$APPEND_SH" \
  --slug my-task \
  --flow loop \
  --phase self_review \
  --verdict pass \
  --source pipeline \
  --cycle 1 \
  --critical 0 --high 0 --medium 0 --low 0 \
  --driver claude \
  --requested-model opus \
  --effective-model opus \
  --honored true \
  --events-dir "$TMP1"

assert_file_exists "1a. output file created" "$OUTFILE"

_line="$(head -1 "$OUTFILE")"

# Validate JSON
if printf '%s\n' "$_line" | jq -e . >/dev/null 2>&1; then
  pass "1b. line parses as valid JSON"
else
  fail "1b. line parses as valid JSON"
fi

assert_eq "1c. schema=1"       "1"           "$(printf '%s\n' "$_line" | jq -r '.schema')"
assert_eq "1d. slug"           "my-task"     "$(printf '%s\n' "$_line" | jq -r '.slug')"
assert_eq "1e. flow"           "loop"        "$(printf '%s\n' "$_line" | jq -r '.flow')"
assert_eq "1f. phase"          "self_review" "$(printf '%s\n' "$_line" | jq -r '.phase')"
assert_eq "1g. verdict"        "pass"        "$(printf '%s\n' "$_line" | jq -r '.verdict')"
assert_eq "1h. source"         "pipeline"    "$(printf '%s\n' "$_line" | jq -r '.source')"
assert_eq "1i. cycle"          "1"           "$(printf '%s\n' "$_line" | jq -r '.cycle')"
assert_eq "1j. driver"         "claude"      "$(printf '%s\n' "$_line" | jq -r '.driver')"
assert_eq "1k. requested_model" "opus"       "$(printf '%s\n' "$_line" | jq -r '.requested_model')"
assert_eq "1l. effective_model" "opus"       "$(printf '%s\n' "$_line" | jq -r '.effective_model')"
assert_eq "1m. honored"        "true"        "$(printf '%s\n' "$_line" | jq -r '.honored | tostring')"
assert_eq "1n. findings.critical" "0"        "$(printf '%s\n' "$_line" | jq -r '.findings.critical')"
assert_eq "1o. triage.action_required" "0"   "$(printf '%s\n' "$_line" | jq -r '.triage.action_required')"

cd "$OLD_DIR"; rm -rf "$TMP1"; trap - EXIT INT TERM

# ── Case 2: second append to same slug appends (2 lines) ──────────────────────
printf '\n==> Case 2: second append to same slug → 2 lines in file\n'

TMP2="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP2"' EXIT INT TERM

DATE="$(date -u '+%Y-%m-%d')"
OUTFILE2="${TMP2}/${DATE}-proj.jsonl"

bash "$APPEND_SH" \
  --slug proj --flow loop --phase verify --verdict pass --source pipeline \
  --events-dir "$TMP2"

bash "$APPEND_SH" \
  --slug proj --flow loop --phase test --verdict pass --source pipeline \
  --events-dir "$TMP2"

_count="$(wc -l < "$OUTFILE2" | tr -d ' ')"
assert_eq "2a. two lines in file" "2" "$_count"

_phase1="$(sed -n '1p' "$OUTFILE2" | jq -r '.phase')"
_phase2="$(sed -n '2p' "$OUTFILE2" | jq -r '.phase')"
assert_eq "2b. first line phase=verify" "verify" "$_phase1"
assert_eq "2c. second line phase=test"  "test"   "$_phase2"

cd "$OLD_DIR"; rm -rf "$TMP2"; trap - EXIT INT TERM

# ── Case 3: missing required flag → exit 1 + no file created ─────────────────
printf '\n==> Case 3: missing required flag → exit 1, no file\n'

TMP3="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP3"' EXIT INT TERM

# Missing --verdict
_exit=0
bash "$APPEND_SH" \
  --slug test-slug --flow loop --phase verify --source pipeline \
  --events-dir "$TMP3" >/dev/null 2>&1 || _exit=$?

if [ "$_exit" -ne 0 ]; then
  pass "3a. exit non-zero when --verdict missing"
else
  fail "3a. exit non-zero when --verdict missing (got exit 0)"
fi

# Missing --slug
_exit2=0
bash "$APPEND_SH" \
  --flow loop --phase verify --verdict pass --source pipeline \
  --events-dir "$TMP3" >/dev/null 2>&1 || _exit2=$?

if [ "$_exit2" -ne 0 ]; then
  pass "3b. exit non-zero when --slug missing"
else
  fail "3b. exit non-zero when --slug missing (got exit 0)"
fi

# No files should have been created in TMP3
_any_files="$(find "$TMP3" -type f 2>/dev/null | wc -l | tr -d ' ')"
assert_eq "3c. no files created on validation failure" "0" "$_any_files"

cd "$OLD_DIR"; rm -rf "$TMP3"; trap - EXIT INT TERM

# ── Case 4: invalid enum → exit 1 ────────────────────────────────────────────
printf '\n==> Case 4: invalid enum values → exit 1\n'

TMP4="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP4"' EXIT INT TERM

# Invalid flow
_exit=0
bash "$APPEND_SH" \
  --slug s --flow invalid_flow --phase verify --verdict pass --source pipeline \
  --events-dir "$TMP4" >/dev/null 2>&1 || _exit=$?
if [ "$_exit" -ne 0 ]; then
  pass "4a. invalid flow → exit 1"
else
  fail "4a. invalid flow → exit 1 (got exit 0)"
fi

# Invalid phase
_exit=0
bash "$APPEND_SH" \
  --slug s --flow loop --phase bad_phase --verdict pass --source pipeline \
  --events-dir "$TMP4" >/dev/null 2>&1 || _exit=$?
if [ "$_exit" -ne 0 ]; then
  pass "4b. invalid phase → exit 1"
else
  fail "4b. invalid phase → exit 1 (got exit 0)"
fi

# Invalid verdict
_exit=0
bash "$APPEND_SH" \
  --slug s --flow loop --phase verify --verdict bad_verdict --source pipeline \
  --events-dir "$TMP4" >/dev/null 2>&1 || _exit=$?
if [ "$_exit" -ne 0 ]; then
  pass "4c. invalid verdict → exit 1"
else
  fail "4c. invalid verdict → exit 1 (got exit 0)"
fi

# Invalid source
_exit=0
bash "$APPEND_SH" \
  --slug s --flow loop --phase verify --verdict pass --source bad_source \
  --events-dir "$TMP4" >/dev/null 2>&1 || _exit=$?
if [ "$_exit" -ne 0 ]; then
  pass "4d. invalid source → exit 1"
else
  fail "4d. invalid source → exit 1 (got exit 0)"
fi

_any_files="$(find "$TMP4" -type f 2>/dev/null | wc -l | tr -d ' ')"
assert_eq "4e. no files created on enum failure" "0" "$_any_files"

cd "$OLD_DIR"; rm -rf "$TMP4"; trap - EXIT INT TERM

# ── Case 5: counts land in findings/triage objects ───────────────────────────
printf '\n==> Case 5: counts land in findings and triage objects\n'

TMP5="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP5"' EXIT INT TERM

bash "$APPEND_SH" \
  --slug counts-test --flow loop --phase cross_review --verdict action_required \
  --source pipeline \
  --critical 2 --high 3 --medium 1 --low 4 \
  --action-required 2 --worth-considering 1 --dismissed 5 \
  --events-dir "$TMP5"

DATE="$(date -u '+%Y-%m-%d')"
_line5="$(head -1 "${TMP5}/${DATE}-counts-test.jsonl")"

assert_eq "5a. findings.critical"          "2" "$(printf '%s\n' "$_line5" | jq -r '.findings.critical')"
assert_eq "5b. findings.high"              "3" "$(printf '%s\n' "$_line5" | jq -r '.findings.high')"
assert_eq "5c. findings.medium"            "1" "$(printf '%s\n' "$_line5" | jq -r '.findings.medium')"
assert_eq "5d. findings.low"               "4" "$(printf '%s\n' "$_line5" | jq -r '.findings.low')"
assert_eq "5e. triage.action_required"     "2" "$(printf '%s\n' "$_line5" | jq -r '.triage.action_required')"
assert_eq "5f. triage.worth_considering"   "1" "$(printf '%s\n' "$_line5" | jq -r '.triage.worth_considering')"
assert_eq "5g. triage.dismissed"           "5" "$(printf '%s\n' "$_line5" | jq -r '.triage.dismissed')"

cd "$OLD_DIR"; rm -rf "$TMP5"; trap - EXIT INT TERM

# ── Case 6: --events-dir override ────────────────────────────────────────────
printf '\n==> Case 6: --events-dir override\n'

TMP6="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP6"' EXIT INT TERM

_custom_dir="${TMP6}/custom/path"
bash "$APPEND_SH" \
  --slug override-test --flow standard --phase verify --verdict pass --source skill \
  --events-dir "$_custom_dir"

DATE="$(date -u '+%Y-%m-%d')"
assert_file_exists "6a. file created in custom dir" "${_custom_dir}/${DATE}-override-test.jsonl"

_line6="$(head -1 "${_custom_dir}/${DATE}-override-test.jsonl")"
assert_eq "6b. flow=standard in custom dir" "standard" "$(printf '%s\n' "$_line6" | jq -r '.flow')"
assert_eq "6c. source=skill in custom dir"  "skill"    "$(printf '%s\n' "$_line6" | jq -r '.source')"

cd "$OLD_DIR"; rm -rf "$TMP6"; trap - EXIT INT TERM

# ── Case 7: optional routing fields omitted when not provided ─────────────────
printf '\n==> Case 7: optional fields absent when not provided\n'

TMP7="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP7"' EXIT INT TERM

bash "$APPEND_SH" \
  --slug minimal --flow loop --phase verify --verdict pass --source pipeline \
  --events-dir "$TMP7"

DATE="$(date -u '+%Y-%m-%d')"
_line7="$(head -1 "${TMP7}/${DATE}-minimal.jsonl")"

# driver, requested_model, effective_model, honored should be absent (null/missing)
_driver7="$(printf '%s\n' "$_line7" | jq -r '.driver // "ABSENT"')"
_model7="$(printf '%s\n' "$_line7" | jq -r '.requested_model // "ABSENT"')"
assert_eq "7a. driver absent when not provided"          "ABSENT" "$_driver7"
assert_eq "7b. requested_model absent when not provided" "ABSENT" "$_model7"

# cycle defaults to 1 when --cycle is omitted (Fix 2: default cycle=1)
_cycle7="$(printf '%s\n' "$_line7" | jq -r '.cycle | tostring')"
assert_eq "7c. cycle=1 when --cycle omitted (default)" "1" "$_cycle7"

cd "$OLD_DIR"; rm -rf "$TMP7"; trap - EXIT INT TERM

# ── Summary ───────────────────────────────────────────────────────────────────
printf '\ninsights-append tests: %d passed, %d failed\n' "$PASS" "$FAIL"

if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
exit 0
