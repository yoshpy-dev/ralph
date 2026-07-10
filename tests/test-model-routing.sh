#!/usr/bin/env sh
# tests/test-model-routing.sh — AC3b coverage: DRY_RUN=1 pipeline pass writes
# model receipts for each routed phase; asserts requested_model values,
# RALPH_FORCE_MODEL override, and codex-driver receipt semantics.
#
# Does NOT invoke claude or codex — DRY_RUN=1 short-circuits all CLI calls
# while still executing the receipt-writing logic (write_model_receipt is
# outside DRY_RUN guards by design).

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIPELINE_SH="${REPO_ROOT}/scripts/ralph-pipeline.sh"

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

# Setup a minimal git repository fixture and a pipeline prompt so
# ralph-pipeline.sh can boot in DRY_RUN=1 mode (it needs CLAUDE.md or a
# pipeline-inner.md to not error on "No implementation prompt found").
setup_git_fixture() {
  _dir="$1"
  cd "$_dir"
  git init -q
  git config user.email "test@example.com"
  git config user.name "Test"
  printf 'initial\n' > README.md
  git add README.md
  git commit -q -m 'initial'

  # Pipeline needs an implementation prompt; provide the minimal fallback.
  mkdir -p .claude/skills/loop/prompts
  printf '# test impl prompt\nWrite COMPLETE to .harness/state/pipeline/.agent-signal\n' \
    > .claude/skills/loop/prompts/pipeline-inner.md
}

# Extract .requested_model from the Nth receipt line (1-based).
receipt_field() {
  _file="$1"
  _n="$2"
  _field="$3"
  sed -n "${_n}p" "$_file" | jq -r ".${_field} // empty" 2>/dev/null || printf ''
}

# Count lines in receipt file.
receipt_count() {
  _file="$1"
  [ -s "$_file" ] || { printf '0\n'; return; }
  wc -l < "$_file" | tr -d ' '
}

OLD_DIR="$(pwd)"

# ── Case 1: DRY_RUN=1, default models ───────────────────────────────────────
printf '\n==> Case 1: DRY_RUN=1 default run — receipt sequence\n'

TMP1="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP1"' EXIT INT TERM
setup_git_fixture "$TMP1"
cd "$TMP1"

RECEIPT="${TMP1}/.harness/state/pipeline/model-receipts.jsonl"

# Run the pipeline in dry-run mode; it should complete immediately with COMPLETE
# signal (dry-run simulates COMPLETE after each agent turn) and write receipts.
DRY_RUN=1 \
  RALPH_LOOP_DRIVER=claude \
  RALPH_MODEL=opus \
  RALPH_IMPLEMENT_MODEL=sonnet \
  RALPH_SELF_REVIEW_MODEL=opus \
  RALPH_VERIFY_MODEL=sonnet \
  RALPH_TEST_MODEL=sonnet \
  RALPH_SYNC_DOCS_MODEL=sonnet \
  RALPH_PR_MODEL=sonnet \
  RALPH_PROBE_MODEL=haiku \
  RALPH_ESCALATION_MODEL=opus \
  RALPH_FORCE_MODEL='' \
  sh "$PIPELINE_SH" --dry-run --max-iterations 3 --max-inner-cycles 2 \
    > "${TMP1}/pipeline.log" 2>&1 || true

assert_file_exists "1a. model-receipts.jsonl exists" "$RECEIPT"

# Validate every line parses with jq
_invalid=0
if [ -s "$RECEIPT" ]; then
  while IFS= read -r _line; do
    if ! printf '%s\n' "$_line" | jq -e . >/dev/null 2>&1; then
      _invalid=$((_invalid + 1))
    fi
  done < "$RECEIPT"
fi
if [ "$_invalid" -eq 0 ]; then
  pass "1b. all receipt lines parse as valid JSON"
else
  fail "1b. all receipt lines parse as valid JSON (${_invalid} invalid)"
fi

# The pipeline writes receipts per phase in order: implement, self_review, verify, test,
# then outer loop: sync_docs, (cross-review skipped in dry-run), pr.
# In DRY_RUN=1, cross-review is skipped (reviewer binary not available).
# Find receipts by phase field rather than line position for robustness.
_impl_model="$(grep '"phase":"implement"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_sr_model="$(grep '"phase":"self_review"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_verify_model="$(grep '"phase":"verify"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_test_model="$(grep '"phase":"test"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_docs_model="$(grep '"phase":"sync_docs"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"

assert_eq "1c. implement receipt → sonnet" "sonnet" "$_impl_model"
assert_eq "1d. self_review receipt → opus" "opus" "$_sr_model"
assert_eq "1e. verify receipt → sonnet" "sonnet" "$_verify_model"
assert_eq "1f. test receipt → sonnet" "sonnet" "$_test_model"
assert_eq "1g. sync_docs receipt → sonnet" "sonnet" "$_docs_model"

# honored should be true for claude driver
_impl_honored="$(grep '"phase":"implement"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.honored | tostring' 2>/dev/null || true)"
assert_eq "1h. implement honored=true (claude driver)" "true" "$_impl_honored"

# driver field should be claude
_impl_driver="$(grep '"phase":"implement"' "$RECEIPT" 2>/dev/null | head -1 | jq -r '.driver // empty' 2>/dev/null || true)"
assert_eq "1i. implement driver=claude" "claude" "$_impl_driver"

cd "$OLD_DIR"
rm -rf "$TMP1"
trap - EXIT INT TERM

# ── Case 2: RALPH_FORCE_MODEL=opus — all receipts show opus ─────────────────
printf '\n==> Case 2: RALPH_FORCE_MODEL=opus — all phase receipts forced\n'

TMP2="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP2"' EXIT INT TERM
setup_git_fixture "$TMP2"
cd "$TMP2"

RECEIPT2="${TMP2}/.harness/state/pipeline/model-receipts.jsonl"

DRY_RUN=1 \
  RALPH_LOOP_DRIVER=claude \
  RALPH_MODEL=sonnet \
  RALPH_IMPLEMENT_MODEL=sonnet \
  RALPH_SELF_REVIEW_MODEL=opus \
  RALPH_VERIFY_MODEL=sonnet \
  RALPH_TEST_MODEL=sonnet \
  RALPH_SYNC_DOCS_MODEL=sonnet \
  RALPH_PR_MODEL=sonnet \
  RALPH_PROBE_MODEL=haiku \
  RALPH_ESCALATION_MODEL=opus \
  RALPH_FORCE_MODEL=opus \
  sh "$PIPELINE_SH" --dry-run --max-iterations 3 --max-inner-cycles 2 \
    > "${TMP2}/pipeline.log" 2>&1 || true

assert_file_exists "2a. model-receipts.jsonl exists (FORCE_MODEL run)" "$RECEIPT2"

_forced_impl="$(grep '"phase":"implement"' "$RECEIPT2" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_forced_sr="$(grep '"phase":"self_review"' "$RECEIPT2" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_forced_verify="$(grep '"phase":"verify"' "$RECEIPT2" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_forced_test="$(grep '"phase":"test"' "$RECEIPT2" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"
_forced_docs="$(grep '"phase":"sync_docs"' "$RECEIPT2" 2>/dev/null | head -1 | jq -r '.requested_model // empty' 2>/dev/null || true)"

assert_eq "2b. FORCE_MODEL: implement → opus" "opus" "$_forced_impl"
assert_eq "2c. FORCE_MODEL: self_review → opus" "opus" "$_forced_sr"
assert_eq "2d. FORCE_MODEL: verify → opus" "opus" "$_forced_verify"
assert_eq "2e. FORCE_MODEL: test → opus" "opus" "$_forced_test"
assert_eq "2f. FORCE_MODEL: sync_docs → opus" "opus" "$_forced_docs"

cd "$OLD_DIR"
rm -rf "$TMP2"
trap - EXIT INT TERM

# ── Case 3: RALPH_LOOP_DRIVER=codex — receipts show codex-default, honored=false ──
printf '\n==> Case 3: RALPH_LOOP_DRIVER=codex — honored=false, effective_model=codex-default\n'

TMP3="$(mktemp -d)"
trap 'cd "$OLD_DIR"; rm -rf "$TMP3"' EXIT INT TERM
setup_git_fixture "$TMP3"
cd "$TMP3"

RECEIPT3="${TMP3}/.harness/state/pipeline/model-receipts.jsonl"

# Under the codex driver, DRY_RUN=1 still runs the receipt-writing path
# (write_model_receipt is called before run_agent, outside DRY_RUN check).
DRY_RUN=1 \
  RALPH_LOOP_DRIVER=codex \
  RALPH_MODEL=opus \
  RALPH_IMPLEMENT_MODEL=sonnet \
  RALPH_SELF_REVIEW_MODEL=opus \
  RALPH_VERIFY_MODEL=sonnet \
  RALPH_TEST_MODEL=sonnet \
  RALPH_SYNC_DOCS_MODEL=sonnet \
  RALPH_PR_MODEL=sonnet \
  RALPH_PROBE_MODEL=haiku \
  RALPH_ESCALATION_MODEL=opus \
  RALPH_FORCE_MODEL='' \
  RALPH_CODEX_SANDBOX=workspace-write \
  RALPH_CODEX_APPROVAL_POLICY=on-failure \
  sh "$PIPELINE_SH" --dry-run --max-iterations 3 --max-inner-cycles 2 \
    > "${TMP3}/pipeline.log" 2>&1 || true

assert_file_exists "3a. model-receipts.jsonl exists (codex driver)" "$RECEIPT3"

_codex_effective="$(grep '"phase":"implement"' "$RECEIPT3" 2>/dev/null | head -1 | jq -r '.effective_model // empty' 2>/dev/null || true)"
_codex_honored="$(grep '"phase":"implement"' "$RECEIPT3" 2>/dev/null | head -1 | jq -r '.honored | tostring' 2>/dev/null || true)"
_codex_driver="$(grep '"phase":"implement"' "$RECEIPT3" 2>/dev/null | head -1 | jq -r '.driver // empty' 2>/dev/null || true)"

assert_eq "3b. codex driver: effective_model=codex-default" "codex-default" "$_codex_effective"
assert_eq "3c. codex driver: honored=false" "false" "$_codex_honored"
assert_eq "3d. codex driver field" "codex" "$_codex_driver"

cd "$OLD_DIR"
rm -rf "$TMP3"
trap - EXIT INT TERM

# ── Summary ──────────────────────────────────────────────────────────────────
printf '\nmodel-routing tests: %d passed, %d failed\n' "$PASS" "$FAIL"

if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
exit 0
