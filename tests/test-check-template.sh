#!/usr/bin/env bash
# tests/test-check-template.sh — regression coverage for scripts/check-template.sh's
# required_files list.
#
# Without this test, an entry can be silently dropped from required_files
# and no automated check notices: #169's /test mutation 6f removed
# scripts/xreview-helpers.sh from the list and nothing failed. This test
# closes that gap with three cases:
#   A. golden: the required_files block in scripts/check-template.sh must
#      match a hardcoded golden list, in order.
#   B. fixture: a synthetic project with every golden file present passes
#      check-template.sh; removing any one golden file (one at a time) makes
#      check-template.sh fail with "Missing required file: <entry>".
#   C. root sanity: every golden entry exists at this repo's own root.
#
# Spec: issue #183. The Go-side counterpart lives in
# internal/scaffold/embed_test.go (TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles).

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK_TEMPLATE="$REPO_ROOT/scripts/check-template.sh"

if [ ! -f "$CHECK_TEMPLATE" ]; then
  echo "FAIL: $CHECK_TEMPLATE not found"
  exit 1
fi

pass_count=0
fail_count=0

pass() {
  pass_count=$((pass_count + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  fail_count=$((fail_count + 1))
  printf '  FAIL: %s\n' "$1"
}

# The golden required_files list: the 6 non-script entries in their current
# order, followed by the 22 scripts/ entries, ordered here to match
# internal/scaffold/embed_test.go's requiredTemplateScripts as a
# readability convention only (the Go test compares both as sets, so
# reordering either list does not by itself fail anything). Update this
# list in the same commit as any change to required_files in
# scripts/check-template.sh, templates/base/scripts/check-template.sh, and
# requiredTemplateScripts in internal/scaffold/embed_test.go.
GOLDEN_ENTRIES=(
  "README.md"
  "AGENTS.md"
  "CLAUDE.md"
  ".claude/settings.json"
  "docs/research/approach-comparison.md"
  "docs/roadmap/harness-maturity-model.md"
  "scripts/run-verify.sh"
  "scripts/run-static-verify.sh"
  "scripts/run-test.sh"
  "scripts/detect-changed-languages.sh"
  "scripts/detect-languages.sh"
  "scripts/archive-plan.sh"
  "scripts/branch-name.sh"
  "scripts/ensure-pr-ready.sh"
  "scripts/ensure-pr-title-prefix.sh"
  "scripts/new-feature-plan.sh"
  "scripts/codex-check.sh"
  "scripts/ralph-config.sh"
  "scripts/ralph-worktree.sh"
  "scripts/xreview-helpers.sh"
  "scripts/secret-scan.sh"
  "scripts/secret-scan-branch.sh"
  "scripts/pre-commit-secret-guard.sh"
  "scripts/commit-msg-guard.sh"
  "scripts/prepare-commit-msg-secret-guard.sh"
  "scripts/pre-merge-commit-secret-guard.sh"
  "scripts/check-template.sh"
  "scripts/check-skill-sync.sh"
)

# extract_required_files <file> — print the entries of a check-template.sh's
# required_files="..." block, one per line, in order, with blank lines
# dropped.
extract_required_files() {
  local file="$1"
  sed -n '/^required_files="$/,/^"$/p' "$file" | sed '1d;$d' | sed '/^[[:space:]]*$/d'
}

# build_fixture <dir> — populate a minimal project tree that passes
# check-template.sh: every golden entry exists, .claude/settings.json is
# "{}", every scripts/*.sh is an executable one-line stub, and the
# directories check-template.sh scans with `find` exist (even if empty) so
# it does not warn about a missing search root.
build_fixture() {
  local dir="$1" entry
  for entry in "${GOLDEN_ENTRIES[@]}"; do
    mkdir -p "$dir/$(dirname "$entry")"
    case "$entry" in
      .claude/settings.json)
        printf '{}\n' > "$dir/$entry"
        ;;
      scripts/*.sh)
        printf '#!/bin/sh\n' > "$dir/$entry"
        chmod +x "$dir/$entry"
        ;;
      *)
        printf 'fixture placeholder\n' > "$dir/$entry"
        ;;
    esac
  done
  mkdir -p "$dir/.claude/hooks" "$dir/.claude/skills" "$dir/.claude/agents" "$dir/packs"
}

# --- A. golden list comparison ----------------------------------------------
run_case_a() {
  local actual expected

  actual="$(extract_required_files "$CHECK_TEMPLATE")"

  if [ -z "$actual" ]; then
    fail "A. required_files block is non-empty (extraction format of $CHECK_TEMPLATE may have changed)"
    return
  fi
  pass "A. required_files block is non-empty"

  expected="$(printf '%s\n' "${GOLDEN_ENTRIES[@]}")"
  if [ "$actual" = "$expected" ]; then
    pass "A. required_files matches the golden list (${#GOLDEN_ENTRIES[@]} entries, in order)"
  else
    fail "A. required_files differs from the golden list"
    echo "    --- diff (< script: $CHECK_TEMPLATE, > golden) ---"
    diff <(printf '%s\n' "$actual") <(printf '%s\n' "$expected") | sed 's/^/    /'
  fi
}

# --- B. fixture pass + per-entry removal detection --------------------------
run_case_b() {
  local fixture entry saved output rc

  fixture="$(mktemp -d "${TMPDIR:-/tmp}/check-template-fixture.XXXXXX")" || {
    fail "B. mktemp -d failed; skipping the fixture cases"
    return
  }
  # Safety net for an interrupted run or a runner timeout mid-loop; the
  # explicit rm -rf at the end of this function also runs on the normal
  # path, after which this trap is cleared.
  trap 'rm -rf "$fixture"' EXIT

  build_fixture "$fixture"

  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && printf '%s\n' "$output" | grep -qF 'Template structure looks good.'; then
    pass "B. clean fixture passes check-template.sh"
  else
    fail "B. clean fixture did not pass check-template.sh (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  for entry in "${GOLDEN_ENTRIES[@]}"; do
    saved="$fixture/${entry}.bak"
    mv "$fixture/$entry" "$saved"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF "Missing required file: $entry"; then
      pass "B. removing $entry is detected"
    else
      fail "B. removing $entry was NOT detected (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
    mv "$saved" "$fixture/$entry"
  done

  rm -rf "$fixture"
  trap - EXIT
}

# --- C. root sanity: this repo satisfies its own golden list ----------------
run_case_c() {
  local entry missing=0
  for entry in "${GOLDEN_ENTRIES[@]}"; do
    if [ ! -e "$REPO_ROOT/$entry" ]; then
      fail "C. $entry missing at repo root"
      missing=$((missing + 1))
    fi
  done
  if [ "$missing" -eq 0 ]; then
    pass "C. all ${#GOLDEN_ENTRIES[@]} golden entries exist at the repo root"
  fi
}

run_case_a
run_case_b
run_case_c

printf '\ntest-check-template: %s passed, %s failed\n' "$pass_count" "$fail_count"
[ "$fail_count" -eq 0 ]
