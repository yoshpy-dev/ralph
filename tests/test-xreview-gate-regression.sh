#!/usr/bin/env bash
# tests/test-xreview-gate-regression.sh — end-to-end coverage for the
# cross-review gate-regression contract under the codex driver / claude
# reviewer path. Regression for issue #50.
#
# Before the fix, ${BASE_BRANCH} / ${REPORTS_DIR} reached the reviewer
# unsubstituted, the triage report landed at an unparseable literal
# path, the `find` parser returned empty, and the gate silently passed
# even when ACTION_REQUIRED findings existed. This test proves the
# end-to-end behavior: with a real triage report present, ACTION_REQUIRED
# findings DO regress the gate; with no findings, the gate proceeds.
#
# We do not boot the full pipeline (too much orchestrator state).
# Instead we exercise the post-render parser + decision logic from
# scripts/ralph-pipeline.sh against a controlled REPORTS_DIR / PIPELINE_DIR
# sandbox, plus we drive the awk renderer the same way the pipeline does
# to confirm rendering + parsing compose correctly.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

PIPELINE_SH="$REPO_ROOT/scripts/ralph-pipeline.sh"
DRIVER_SH="$REPO_ROOT/scripts/ralph-cli-driver.sh"
ADV_PROMPT="$REPO_ROOT/.claude/skills/cross-review/prompts/adversarial-claude.md"

if [ ! -f "$ADV_PROMPT" ]; then
  echo "FAIL: adversarial prompt not found at $ADV_PROMPT"
  exit 1
fi

# Source count_triage_findings (used by the gate to count findings).
# shellcheck source=/dev/null
. "$DRIVER_SH"

pass=0
fail=0

check() {
  local label="$1"
  shift
  if "$@"; then
    printf '  PASS  %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s\n' "$label"
    fail=$((fail + 1))
  fi
}

check_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    printf '  PASS  %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s — expected %q got %q\n' "$label" "$expected" "$actual"
    fail=$((fail + 1))
  fi
}

WORK_DIR="$(mktemp -d -t xreview-gate-XXXXXX)"
trap 'rm -rf "$WORK_DIR"' EXIT

PIPELINE_DIR="$WORK_DIR/.harness/state/pipeline"
REPORTS_DIR="$WORK_DIR/reports"
mkdir -p "$PIPELINE_DIR" "$REPORTS_DIR"

# A checkpoint older than the triage report so `find -newer` accepts the
# report. Touching to a fixed past timestamp keeps the test deterministic.
CHECKPOINT="$PIPELINE_DIR/checkpoint.json"
echo '{}' > "$CHECKPOINT"
# Backdate the checkpoint so any file created after this point passes -newer.
touch -t 200001010000 "$CHECKPOINT"

# render <base> <reports_dir> <input> <output> — identical to the awk
# renderer in scripts/ralph-pipeline.sh. The drift assertion in
# test-xreview-prompt-render.sh guarantees the production renderer
# stays in lock-step with this one.
render() {
  local base="$1" reports_dir="$2" in_file="$3" out_file="$4"
  BASE_BRANCH="$base" REPORTS_DIR="$reports_dir" \
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
    ' "$in_file" > "$out_file"
}

# allowlist_leftovers <rendered> — matches the pipeline's allowlist guard.
allowlist_leftovers() {
  grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' "$1" 2>/dev/null | sort -u
}

# write_triage <action_required> <worth_considering> <dismissed> <out_path>
#
# Emit a minimally-valid triage report whose `After triage:` summary
# line and `## <CATEGORY>` table rows together pin the counts the
# pipeline parser is expected to read.
write_triage() {
  local ar="$1" wc="$2" dm="$3" out="$4"
  {
    echo "# Cross-review triage report"
    echo
    echo "- Driver: codex"
    echo "- Reviewer: claude"
    echo "- Triager: Claude Code (loop pipeline reviewer-inversion)"
    echo "- After triage: ACTION_REQUIRED=${ar}, WORTH_CONSIDERING=${wc}, DISMISSED=${dm}"
    echo
    echo "## ACTION_REQUIRED"
    echo
    if [ "$ar" -gt 0 ]; then
      echo "| # | Finding | Severity | Resolution |"
      echo "| - | ------- | -------- | ---------- |"
      i=1
      while [ "$i" -le "$ar" ]; do
        echo "| $i | finding-${i} | HIGH | must fix |"
        i=$((i + 1))
      done
    fi
    echo
    echo "## WORTH_CONSIDERING"
    echo
    if [ "$wc" -gt 0 ]; then
      echo "| # | Finding | Severity | Resolution |"
      echo "| - | ------- | -------- | ---------- |"
      i=1
      while [ "$i" -le "$wc" ]; do
        echo "| $i | finding-w-${i} | MEDIUM | follow-up |"
        i=$((i + 1))
      done
    fi
    echo
    echo "## DISMISSED"
    echo
    if [ "$dm" -gt 0 ]; then
      echo "| # | Finding | Severity | Resolution |"
      echo "| - | ------- | -------- | ---------- |"
      i=1
      while [ "$i" -le "$dm" ]; do
        echo "| $i | finding-d-${i} | LOW | dismissed |"
        i=$((i + 1))
      done
    fi
  } > "$out"
}

# gate_decision <triage_path> — returns the same exit-code semantics as
# the inner cross-review block: 0 = proceed, 1 = regress. Implements
# only the decision logic (the parser feeds it).
gate_decision() {
  local triage="$1"
  local fix_all="${FIX_ALL:-0}"
  local action_required worth_considering
  action_required="$(count_triage_findings "$triage" ACTION_REQUIRED)"
  worth_considering="$(count_triage_findings "$triage" WORTH_CONSIDERING)"
  if [ "$action_required" -gt 0 ]; then
    return 1
  fi
  if [ "$fix_all" -eq 1 ] && [ "$worth_considering" -gt 0 ]; then
    return 1
  fi
  return 0
}

# ─── Phase 1: end-to-end render + parse + gate ──────────────────────────────
echo
echo "── Phase 1: render the real adversarial prompt, then drive parser+gate"

RENDERED="$PIPELINE_DIR/outer-1-adversarial-claude.md"
render "main" "$REPORTS_DIR" "$ADV_PROMPT" "$RENDERED"

check "1a. rendered prompt file exists" test -s "$RENDERED"
check "1b. rendered prompt has no literal \${BASE_BRANCH}" \
  bash -c '! grep -q "\${BASE_BRANCH}" "$1"' _ "$RENDERED"
check "1c. rendered prompt has no literal \${REPORTS_DIR}" \
  bash -c '! grep -q "\${REPORTS_DIR}" "$1"' _ "$RENDERED"
check "1d. rendered prompt contains literal main" \
  grep -Fq 'main' "$RENDERED"
check "1e. rendered prompt contains literal REPORTS_DIR path" \
  grep -Fq "$REPORTS_DIR" "$RENDERED"

leftover="$(allowlist_leftovers "$RENDERED" || true)"
check_eq "1f. allowlist guard finds no leftovers" "" "$leftover"

# ─── Phase 2: triage with ACTION_REQUIRED=1 must regress the gate ───────────
echo
echo "── Phase 2: ACTION_REQUIRED=1 triage → gate MUST regress (return 1)"

TRIAGE_AR="$REPORTS_DIR/cross-review-triage-test-$(date -u +%Y-%m-%d-%H%M%S).md"
write_triage 1 0 0 "$TRIAGE_AR"
# Mark it newer than the checkpoint we backdated above.
touch "$TRIAGE_AR"

check "2a. triage report exists" test -s "$TRIAGE_AR"

# Parser path the pipeline uses: find -newer the checkpoint, latest match.
discovered="$(find "$REPORTS_DIR" -name 'cross-review-triage-*' -newer "$CHECKPOINT" 2>/dev/null | tail -1 || true)"
check_eq "2b. find -newer picks up the triage report" "$TRIAGE_AR" "$discovered"

ar_count="$(count_triage_findings "$TRIAGE_AR" ACTION_REQUIRED)"
check_eq "2c. count_triage_findings ACTION_REQUIRED" "1" "$ar_count"

if gate_decision "$TRIAGE_AR"; then
  echo "  FAIL  2d. gate_decision returned 0 (proceed) on ACTION_REQUIRED=1 — silent bypass regressed!"
  fail=$((fail + 1))
else
  echo "  PASS  2d. gate_decision returned non-zero (regress) on ACTION_REQUIRED=1"
  pass=$((pass + 1))
fi

# ─── Phase 3: triage with no findings → gate proceeds ───────────────────────
echo
echo "── Phase 3: triage with no findings → gate proceeds (return 0)"

TRIAGE_CLEAN="$REPORTS_DIR/cross-review-triage-clean-$(date -u +%Y-%m-%d-%H%M%S).md"
write_triage 0 0 0 "$TRIAGE_CLEAN"
touch "$TRIAGE_CLEAN"

clean_ar="$(count_triage_findings "$TRIAGE_CLEAN" ACTION_REQUIRED)"
check_eq "3a. count_triage_findings ACTION_REQUIRED on clean triage" "0" "$clean_ar"

if gate_decision "$TRIAGE_CLEAN"; then
  echo "  PASS  3b. gate_decision returned 0 (proceed) on clean triage"
  pass=$((pass + 1))
else
  echo "  FAIL  3b. gate_decision returned non-zero on clean triage — false regression"
  fail=$((fail + 1))
fi

# ─── Phase 4: WORTH_CONSIDERING under --fix-all → gate regresses ────────────
echo
echo "── Phase 4: --fix-all + WORTH_CONSIDERING=1 → gate MUST regress"

TRIAGE_WC="$REPORTS_DIR/cross-review-triage-wc-$(date -u +%Y-%m-%d-%H%M%S).md"
write_triage 0 1 0 "$TRIAGE_WC"
touch "$TRIAGE_WC"

wc_count="$(count_triage_findings "$TRIAGE_WC" WORTH_CONSIDERING)"
check_eq "4a. count_triage_findings WORTH_CONSIDERING" "1" "$wc_count"

if FIX_ALL=0 gate_decision "$TRIAGE_WC"; then
  echo "  PASS  4b. without --fix-all, WORTH_CONSIDERING does not regress"
  pass=$((pass + 1))
else
  echo "  FAIL  4b. WORTH_CONSIDERING regressed gate without --fix-all"
  fail=$((fail + 1))
fi

if FIX_ALL=1 gate_decision "$TRIAGE_WC"; then
  echo "  FAIL  4c. --fix-all + WORTH_CONSIDERING failed to regress gate"
  fail=$((fail + 1))
else
  echo "  PASS  4c. --fix-all + WORTH_CONSIDERING regresses gate"
  pass=$((pass + 1))
fi

# ─── Phase 5: pre-fix regression — literal placeholders must NOT pass the gate
# Confirms that the very failure mode reported in issue #50 (literal
# ${REPORTS_DIR} reaching the reviewer) would be caught by the allowlist
# guard before any reviewer invocation, so the gate cannot silently pass
# in the broken path.
echo
echo "── Phase 5: pre-fix path (literal placeholders) → allowlist guard catches"

BROKEN="$PIPELINE_DIR/outer-1-broken.md"
cp "$ADV_PROMPT" "$BROKEN"   # NOT rendered — placeholders still literal
leftover="$(allowlist_leftovers "$BROKEN" || true)"
case "$leftover" in
  *'${BASE_BRANCH}'*|*'${REPORTS_DIR}'*)
    echo "  PASS  5a. allowlist guard would flag literal placeholders (gate fails closed)"
    pass=$((pass + 1))
    ;;
  *)
    echo "  FAIL  5a. allowlist guard missed literal placeholders in unrendered prompt"
    fail=$((fail + 1))
    ;;
esac

# ─── Summary ────────────────────────────────────────────────────────────────
echo
echo "Gate-regression test summary: ${pass} passed, ${fail} failed"
if [ "$fail" -ne 0 ]; then
  exit 1
fi
exit 0
