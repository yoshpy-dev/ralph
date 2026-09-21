#!/usr/bin/env sh
# Regression tests for the branch secret scan step wired into
# scripts/run-verify.sh: static/all run it, test mode does not; a finding
# fails the run even when no language verifier ran; the skip env var; and
# the docs-only summary message is unaffected.
#
# All fixture values are assembled at test RUNTIME from split pieces (see
# tests/test-secret-scan-branch.sh for why).
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"

pass=0
fail=0

ok() {
  pass=$((pass + 1))
  printf '  PASS: %s\n' "$1"
}

not_ok() {
  fail=$((fail + 1))
  printf '  FAIL: %s\n' "$1"
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-run-verify-branch-secret-scan.XXXXXX")"

# make_repo <name> -- a fresh git repo with scripts/run-verify.sh and the
# secret-scan scripts it needs, no language packs and no verify.local.sh
# (so ran_any stays 0 unless a caller adds one), and a base "main" branch.
make_repo() {
  _repo="$workdir/$1"
  mkdir -p "$_repo/scripts"
  cp "$PROJECT_ROOT/scripts/run-verify.sh" "$_repo/scripts/run-verify.sh"
  cp "$PROJECT_ROOT/scripts/secret-scan-branch.sh" "$_repo/scripts/secret-scan-branch.sh"
  cp "$PROJECT_ROOT/scripts/secret-scan.sh" "$_repo/scripts/secret-scan.sh"
  cp "$PROJECT_ROOT/scripts/xreview-helpers.sh" "$_repo/scripts/xreview-helpers.sh"
  chmod +x "$_repo/scripts/"*.sh
  cat > "$_repo/scripts/detect-languages.sh" <<'SH'
#!/usr/bin/env sh
true
SH
  chmod +x "$_repo/scripts/detect-languages.sh"
  (
    cd "$_repo"
    git init -q
    git config user.email test@example.com
    git config user.name "Run Verify Branch Secret Scan Test"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
  )
  printf '%s\n' "$_repo"
}

# run_verify <repo> <mode> [skip_value] -- runs run-verify.sh from <repo>
# with a clean slate for GITHUB_BASE_REF/RALPH_XREVIEW_BASE (this suite's
# own CI sets GITHUB_BASE_REF for real) and captures combined output +
# exit code. skip_value is assigned directly to
# RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN (run-verify.sh only treats "1" as
# "skip"; other test cases pass "0" or leave it unset via "").
run_verify() {
  _repo="$1"
  _mode="$2"
  shift 2
  _skip="${1:-}"
  set +e
  (
    cd "$_repo"
    # This suite may itself run under run-test.sh, which exports
    # RALPH_VERIFY_SCOPE=changed for ITSELF -- pin scope to full so a
    # fake repo without scripts/detect-changed-languages.sh does not
    # inherit a "changed" scope fallback that forces docs_only=false.
    unset GITHUB_BASE_REF RALPH_XREVIEW_BASE RALPH_SECRET_ALLOWLIST
    RALPH_VERIFY_SCOPE=full HARNESS_VERIFY_MODE="$_mode" RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN="$_skip" ./scripts/run-verify.sh
  ) >"$out_file" 2>&1
  run_rc=$?
  set -e
}

out_file="$(mktemp "${TMPDIR:-/tmp}/ralph-run-verify-branch-secret-scan-test.out.XXXXXX")"
trap 'rm -rf "$workdir" "$out_file"' EXIT HUP INT TERM

assert_exit() {
  _desc="$1"
  _want="$2"
  if [ "$run_rc" -eq "$_want" ]; then
    ok "$_desc (exit $run_rc)"
  else
    not_ok "$_desc (exit $run_rc, want $_want)"
    printf '%s\n' "--- output ---"
    cat "$out_file"
  fi
}

assert_contains() {
  _desc="$1"
  _needle="$2"
  if grep -F -- "$_needle" "$out_file" >/dev/null 2>&1; then
    ok "$_desc"
  else
    not_ok "$_desc (missing: $_needle)"
    printf '%s\n' "--- output ---"
    cat "$out_file"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  if grep -F -- "$_needle" "$out_file" >/dev/null 2>&1; then
    not_ok "$_desc (unexpected: $_needle)"
    printf '%s\n' "--- output ---"
    cat "$out_file"
  else
    ok "$_desc"
  fi
}

add_clean_commit() {
  printf 'more clean\n' >> README.md
  git add README.md
  git commit -q -m 'clean change'
}

add_secret_commit() {
  token="$(printf 'ghp_%s' 'abcdefghijklmnopqrstuvwxyzFGHIJ')"
  printf 'deploy token %s\n' "$token" > leaked.txt
  git add leaked.txt
  git commit -q -m 'add leaked token'
}

# ---------------------------------------------------------------------------
# static/all run the branch scan; test mode does not.
# ---------------------------------------------------------------------------
test_mode_routing_clean() {
  repo="$(make_repo mode-routing-clean)"
  (cd "$repo" && git checkout -q -b feature && add_clean_commit)

  run_verify "$repo" static
  assert_exit "static mode, clean branch: exits 0" 0
  assert_contains "static mode: runs branch secret scan" "Running branch secret scan"

  run_verify "$repo" all
  assert_exit "all mode, clean branch: exits 0" 0
  assert_contains "all mode: runs branch secret scan" "Running branch secret scan"

  run_verify "$repo" test
  assert_exit "test mode, clean branch: exits 0" 0
  assert_not_contains "test mode: does not run branch secret scan" "Running branch secret scan"
  assert_not_contains "test mode: does not even mention the branch secret scan" "branch secret scan"
}

# ---------------------------------------------------------------------------
# A finding fails the run even when no language verifier ran (ran_any=0):
# the "docs-only" summary alone is no longer the whole story -- self-review
# cycle 1 (M1) found that a docs-only-looking run with a real finding used
# to close on a purely reassuring summary. An explicit failure line must
# now appear near the end regardless of ran_any/docs_only.
# ---------------------------------------------------------------------------
test_finding_fails_even_with_ran_any_zero() {
  repo="$(make_repo finding-fails)"
  (cd "$repo" && git checkout -q -b feature && add_secret_commit)

  run_verify "$repo" all
  assert_exit "finding on a docs-only-looking run: still exits non-zero" 1
  assert_contains "finding: reports the branch scan ran" "Running branch secret scan"
  assert_contains "finding: reports findings" "findings"
  # ran_any stays 0 (no language verifier, no verify.local.sh), so the
  # generic "No language verifier ran" summary still prints alongside the
  # explicit failure line below -- both are true and both must appear.
  assert_contains "finding: the generic no-verifier-ran message still appears" "No language verifier ran"
  assert_contains "finding: an explicit failure line appears despite the reassuring summary" "Branch secret scan failed"
}

# ---------------------------------------------------------------------------
# A finding also fails the run when a language verifier DID run (ran_any=1):
# the scan failure must still flip the overall verdict to "Some verifiers
# failed", not get masked by an otherwise-passing language verifier, and
# the explicit failure line appears here too.
# ---------------------------------------------------------------------------
test_finding_fails_with_ran_any_one() {
  repo="$(make_repo finding-fails-ran-any-one)"
  mkdir -p "$repo/packs/languages/golang"
  cat > "$repo/scripts/detect-languages.sh" <<'SH'
#!/usr/bin/env sh
printf 'golang\n'
SH
  chmod +x "$repo/scripts/detect-languages.sh"
  cat > "$repo/packs/languages/golang/verify.sh" <<'SH'
#!/usr/bin/env sh
exit 0
SH
  chmod +x "$repo/packs/languages/golang/verify.sh"
  (cd "$repo" && git checkout -q -b feature && add_secret_commit)

  run_verify "$repo" all
  assert_exit "finding with a passing language verifier: still exits non-zero" 1
  assert_contains "finding+ran_any=1: reports the branch scan ran" "Running branch secret scan"
  assert_contains "finding+ran_any=1: overall verdict says some verifiers failed" "Some verifiers failed"
  assert_contains "finding+ran_any=1: explicit failure line also appears" "Branch secret scan failed"
}

# ---------------------------------------------------------------------------
# self-review cycle 1 (LOW, D3): the skip variable must act only when it is
# exactly "1" -- "0" and "true" must NOT skip (an unforgiving reading is the
# safe direction for a leak gate).
# ---------------------------------------------------------------------------
test_skip_env_var() {
  repo="$(make_repo skip-env)"
  (cd "$repo" && git checkout -q -b feature && add_secret_commit)

  run_verify "$repo" static 1
  assert_exit "skip env var =1: exits 0 despite the secret commit" 0
  assert_contains "skip env var =1: reports the skip" "Skipping branch secret scan"
  assert_not_contains "skip env var =1: does not run the scan" "Running branch secret scan"

  run_verify "$repo" static 0
  assert_exit "skip env var =0: does NOT skip, still fails" 1
  assert_contains "skip env var =0: runs the scan" "Running branch secret scan"
  assert_not_contains "skip env var =0: does not report a skip" "Skipping branch secret scan"

  run_verify "$repo" static true
  assert_exit "skip env var =true: does NOT skip, still fails" 1
  assert_contains "skip env var =true: runs the scan" "Running branch secret scan"
  assert_not_contains "skip env var =true: does not report a skip" "Skipping branch secret scan"
}

# ---------------------------------------------------------------------------
# Docs-only output is unaffected by the new step: the same message appears
# on a clean, ran_any=0 run, whether or not the scan also ran.
# ---------------------------------------------------------------------------
test_docs_only_output_unchanged() {
  repo="$(make_repo docs-only)"
  (cd "$repo" && git checkout -q -b feature && add_clean_commit)

  run_verify "$repo" static
  assert_exit "docs-only clean run: exits 0" 0
  assert_contains "docs-only clean run: still reports docs/scaffold-only" "No language verifier ran. This appears to be docs or scaffold-level work only."
}

# ---------------------------------------------------------------------------
# Older scaffold without secret-scan-branch.sh: non-fatal, one line, and
# does not block an otherwise-clean run.
# ---------------------------------------------------------------------------
test_missing_scanner_is_non_fatal() {
  repo="$(make_repo missing-scanner)"
  rm -f "$repo/scripts/secret-scan-branch.sh"
  (cd "$repo" && git checkout -q -b feature && add_clean_commit)

  run_verify "$repo" static
  assert_exit "missing secret-scan-branch.sh: still exits 0" 0
  assert_contains "missing secret-scan-branch.sh: reports the skip reason" "missing or not executable"
}

test_mode_routing_clean
test_finding_fails_even_with_ran_any_zero
test_finding_fails_with_ran_any_one
test_skip_env_var
test_docs_only_output_unchanged
test_missing_scanner_is_non_fatal

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
