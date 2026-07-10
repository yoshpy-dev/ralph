#!/usr/bin/env bash
# tests/test-xreview-prompt-render.sh — exercise the awk renderer that
# expands ${BASE_BRANCH} / ${REPORTS_DIR} in the adversarial-claude
# prompt before `claude -p` consumes it.
#
# Regression coverage for issue #50: literal placeholders previously
# reached the reviewer because `claude -p` does not template-substitute
# its stdin. The renderer must:
#   - substitute both placeholders with the real values
#   - leave no literal ${BASE_BRANCH} / ${REPORTS_DIR} tokens behind
#   - treat replacement values as literal strings (no sed-style & / \
#     interpretation), so git refs containing # / & / / / \ pass through
#   - cooperate with the allowlist guard that fails the cross-review
#     gate closed on any unresolved ${...} token outside the allowlist.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

PIPELINE_SH="scripts/ralph-pipeline.sh"

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

# render <base> <reports_dir> <input_file> <output_file>
#
# Mirrors the awk renderer embedded in scripts/ralph-pipeline.sh
# (cross-review phase). Kept in lock-step via the drift assertion below.
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

# allowlist_leftovers <rendered_file>
#
# Print any ${SHELL_STYLE_VAR} tokens that remain in the rendered prompt.
# Empty stdout means the allowlist guard would accept the file.
allowlist_leftovers() {
  grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' "$1" 2>/dev/null | sort -u
}

WORK_DIR="$(mktemp -d -t xreview-render-XXXXXX)"
trap 'rm -rf "$WORK_DIR"' EXIT

# ─── Drift guard: the renderer in ralph-pipeline.sh must use the same
# lreplace function definition (literal index/substr replacement). If
# this assertion fails, someone changed one side and not the other —
# update both so the test reflects production behavior.
echo
echo "── Drift guard: pipeline renderer matches test renderer"
check "0a. pipeline contains literal lreplace function" \
  grep -q 'function lreplace(s, needle, repl' "$PIPELINE_SH"
check "0b. pipeline calls lreplace for BASE_BRANCH" \
  grep -q 'lreplace(line, "\${BASE_BRANCH}"' "$PIPELINE_SH"
check "0c. pipeline calls lreplace for REPORTS_DIR" \
  grep -q 'lreplace(line, "\${REPORTS_DIR}"' "$PIPELINE_SH"

# Sample prompt: cover both placeholders, plus literal $ that must NOT be
# touched, plus a bare ${UNKNOWN} placeholder used in the negative test.
SAMPLE_PROMPT="$WORK_DIR/prompt.md"
cat > "$SAMPLE_PROMPT" <<'PROMPT'
Base branch: `${BASE_BRANCH}` (compare with `git diff "${BASE_BRANCH}...HEAD"`)
Reports directory: `${REPORTS_DIR}`
Write report to: `${REPORTS_DIR}/cross-review-triage-<slug>.md`
Literal money: $0.00 — must survive.
Bash array form: ${arr[0]} — lowercase must survive.
PROMPT

# Parameterized cases over base and reports_dir.
# Format: label|base|reports_dir
CASES=$(cat <<'CASES'
main|main|docs/reports
slashed-ref|release/3.5|docs/reports
hash-in-ref|feature#1|docs/reports
amp-in-ref|feature&1|docs/reports
backslash-in-ref|feature\back|docs/reports
hash-in-reports|main|docs/reports#1
amp-in-reports|main|docs/reports&backup
CASES
)

i=0
echo "$CASES" | while IFS='|' read -r label base reports_dir; do
  [ -z "$label" ] && continue
  i=$((i + 1))
  out_file="$WORK_DIR/rendered-${i}-${label}.md"

  echo
  echo "── Case ${i}: ${label} (base=${base}, reports_dir=${reports_dir})"

  render "$base" "$reports_dir" "$SAMPLE_PROMPT" "$out_file"

  # 1. No literal placeholder tokens remain.
  if grep -q '\${BASE_BRANCH}' "$out_file"; then
    printf '  FAIL  %s.a literal ${BASE_BRANCH} remained\n' "$i"
    fail=$((fail + 1))
  else
    printf '  PASS  %s.a literal ${BASE_BRANCH} removed\n' "$i"
    pass=$((pass + 1))
  fi
  if grep -q '\${REPORTS_DIR}' "$out_file"; then
    printf '  FAIL  %s.b literal ${REPORTS_DIR} remained\n' "$i"
    fail=$((fail + 1))
  else
    printf '  PASS  %s.b literal ${REPORTS_DIR} removed\n' "$i"
    pass=$((pass + 1))
  fi

  # 2. Substituted values appear verbatim (no metacharacter mutation).
  if grep -Fq "$base" "$out_file"; then
    printf '  PASS  %s.c base %q rendered literally\n' "$i" "$base"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s.c base %q did not appear in rendered output\n' "$i" "$base"
    fail=$((fail + 1))
  fi
  if grep -Fq "$reports_dir" "$out_file"; then
    printf '  PASS  %s.d reports_dir %q rendered literally\n' "$i" "$reports_dir"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s.d reports_dir %q did not appear in rendered output\n' "$i" "$reports_dir"
    fail=$((fail + 1))
  fi

  # 3. Lowercase placeholder forms and literal $ are untouched.
  if grep -Fq '${arr[0]}' "$out_file"; then
    printf '  PASS  %s.e lowercase ${arr[0]} preserved\n' "$i"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s.e lowercase ${arr[0]} was incorrectly substituted\n' "$i"
    fail=$((fail + 1))
  fi
  if grep -Fq '$0.00' "$out_file"; then
    printf '  PASS  %s.f literal $0.00 preserved\n' "$i"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s.f literal $0.00 was incorrectly substituted\n' "$i"
    fail=$((fail + 1))
  fi

  # 4. Allowlist guard accepts the rendered file.
  leftovers="$(allowlist_leftovers "$out_file" || true)"
  if [ -z "$leftovers" ]; then
    printf '  PASS  %s.g allowlist guard sees no leftovers\n' "$i"
    pass=$((pass + 1))
  else
    printf '  FAIL  %s.g allowlist guard found leftovers: %s\n' "$i" "$leftovers"
    fail=$((fail + 1))
  fi

  # Re-export the counters for the next iteration of the subshell.
  echo "$pass $fail" > "$WORK_DIR/.counts"
done

# Pull the counter back from the subshell (POSIX `while` runs in one).
if [ -f "$WORK_DIR/.counts" ]; then
  read -r pass fail < "$WORK_DIR/.counts"
fi

# ─── Negative case: ${UNKNOWN} placeholder must trip the allowlist guard.
echo
echo "── Negative case: unsupported placeholder triggers allowlist guard"
NEG_PROMPT="$WORK_DIR/neg-prompt.md"
cat > "$NEG_PROMPT" <<'PROMPT'
Base branch: ${BASE_BRANCH}
Reports: ${REPORTS_DIR}
Unknown var: ${UNKNOWN_PLACEHOLDER}
PROMPT

NEG_OUT="$WORK_DIR/neg-rendered.md"
render "main" "docs/reports" "$NEG_PROMPT" "$NEG_OUT"

leftovers="$(allowlist_leftovers "$NEG_OUT" || true)"
case "$leftovers" in
  *'${UNKNOWN_PLACEHOLDER}'*)
    echo "  PASS  neg.a allowlist guard caught \${UNKNOWN_PLACEHOLDER}"
    pass=$((pass + 1))
    ;;
  *)
    echo "  FAIL  neg.a allowlist guard missed unsupported placeholder; saw: $leftovers"
    fail=$((fail + 1))
    ;;
esac

# Allowlist guard must NOT flag supported placeholders that were rendered.
case "$leftovers" in
  *'${BASE_BRANCH}'*|*'${REPORTS_DIR}'*)
    echo "  FAIL  neg.b allowlist guard saw a supported placeholder it should have rendered"
    fail=$((fail + 1))
    ;;
  *)
    echo "  PASS  neg.b allowlist guard does not flag supported placeholders"
    pass=$((pass + 1))
    ;;
esac

# ─── Summary ─────────────────────────────────────────────────────────────
echo
echo "Renderer test summary: ${pass} passed, ${fail} failed"
if [ "$fail" -ne 0 ]; then
  exit 1
fi
exit 0
