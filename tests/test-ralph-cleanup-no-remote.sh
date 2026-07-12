#!/usr/bin/env bash
# test-ralph-cleanup-no-remote.sh — regression for:
#   (a) cleanup_plan_artifacts eager default_branch resolution dies in trunk-only repos
#   (b) validate-clean-base double error message on failure
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RALPH="${ROOT_DIR}/scripts/ralph"
WORKTREE="${ROOT_DIR}/scripts/ralph-worktree.sh"

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
  if [ "$2" = "$3" ]; then
    pass "$1"
  else
    fail "$1" "expected='$2' actual='$3'"
  fi
}

printf '==> ralph cleanup no-remote regression tests\n'

TMP="$(mktemp -d "${TMPDIR:-/tmp}/ralph-cleanup-no-remote.XXXXXX")"
cleanup_tmp() { rm -rf "$TMP"; }
trap cleanup_tmp EXIT

# ═══════════════════════════════════════════════════════════════════
# Setup: trunk-only git repo (no remote, branch named 'trunk')
# so default_branch cannot resolve via symbolic-ref.
# ═══════════════════════════════════════════════════════════════════

REPO="${TMP}/repo"
mkdir -p "$REPO"
(
  cd "$REPO"
  git init -q -b trunk
  git config user.email test@example.com
  git config user.name 'Test User'
  printf 'seed\n' > README.md
  git add README.md
  git commit -q -m 'chore: seed'
)

# ═══════════════════════════════════════════════════════════════════
# Test 1: cleanup --dry-run completes in a trunk-only repo
# (eager default_branch previously caused exit-1 before any work was done)
# ═══════════════════════════════════════════════════════════════════

printf '\n--- cleanup --dry-run: trunk-only repo ---\n'

PLAN_DIR="${REPO}/docs/plans/active/2026-07-13-test-cleanup"
mkdir -p "$PLAN_DIR"
cat > "${PLAN_DIR}/_manifest.md" <<'MD'
# Test Cleanup Plan

- Type: fix
- Related issue: #999

## Shared-file locklist

- scripts/ralph.sh
MD
cat > "${PLAN_DIR}/slice-1-alpha.md" <<'MD'
# Slice 1

- Objective: Alpha slice
- Dependencies: none
- Affected files: scripts/ralph.sh
MD

_cleanup_log="${TMP}/cleanup.log"
_rc=0
(
  cd "$REPO"
  "$RALPH" cleanup --plan "$PLAN_DIR" --dry-run
) > "$_cleanup_log" 2>&1 || _rc=$?

if [ "$_rc" -eq 0 ]; then
  pass 'cleanup --dry-run exits 0 in trunk-only repo'
else
  fail 'cleanup --dry-run exits 0 in trunk-only repo' "exit code: ${_rc}; log: $(cat "$_cleanup_log")"
fi

# The dry-run should log "Dry run: 1" (proves cleanup_plan_artifacts ran, not died early)
if grep -q 'Dry run: 1' "$_cleanup_log" 2>/dev/null; then
  pass 'cleanup --dry-run log confirms dry-run mode reached cleanup body'
else
  fail 'cleanup --dry-run log confirms dry-run mode reached cleanup body' "log: $(cat "$_cleanup_log")"
fi

# ═══════════════════════════════════════════════════════════════════
# Test 2: validate-clean-base failure produces exactly one error line
# (dispatch AND function previously both called default_branch, producing
# two identical "base branch not found" stderr lines)
# ═══════════════════════════════════════════════════════════════════

printf '\n--- validate-clean-base: exactly one error line on failure ---\n'

REPO2="${TMP}/repo2"
mkdir -p "$REPO2"
(
  cd "$REPO2"
  git init -q -b trunk
  git config user.email test@example.com
  git config user.name 'Test User'
  printf 'seed\n' > README.md
  git add README.md
  git commit -q -m 'chore: seed'
)

_vcb_log="${TMP}/vcb.log"
_vcb_rc=0
(
  cd "$REPO2"
  # Ask for a branch that does not exist — triggers the "base branch not found"
  # error path that previously printed twice.
  "$WORKTREE" validate-clean-base nonexistent-base
) > "$_vcb_log" 2>&1 || _vcb_rc=$?

# Must exit non-zero (branch not found)
if [ "$_vcb_rc" -ne 0 ]; then
  pass 'validate-clean-base exits non-zero for missing branch'
else
  fail 'validate-clean-base exits non-zero for missing branch' "exit code was 0"
fi

# Count lines that mention the error (the key phrase from die())
_error_lines="$(grep -c 'base branch not found\|nonexistent-base' "$_vcb_log" 2>/dev/null || true)"
assert_eq 'validate-clean-base: exactly one error line' '1' "$_error_lines"

printf '\n==> ralph cleanup no-remote tests: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
