#!/usr/bin/env sh
# Regression tests for the branch secret scan step wired into
# scripts/run-verify.sh: static/all run it, test mode does not; a finding
# fails the run even when no language verifier ran; the skip setting
# [verify].skip_branch_secret_scan of ralph.local.toml, which a stand-in
# ralph answers (tests/lib/fake-ralph.sh); and the docs-only summary message
# is unaffected.
#
# All fixture values are assembled at test RUNTIME from split pieces (see
# tests/test-secret-scan-branch.sh for why).
#
# Hermeticity: HOME/GIT_CONFIG_GLOBAL/GIT_CONFIG_SYSTEM are isolated to a
# throwaway dir (same pattern as tests/test-terraform-gitignore.sh and
# tests/test-secret-scan-branch.sh), because every fixture repo below runs
# `git commit`: a host or CI runner with a global `commit.gpgsign = true`
# would otherwise make every commit in this file fail with a gpg-signing
# error unrelated to the script under test.
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

hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_TERMINAL_PROMPT

# The stand-in ralph first on PATH in every run_verify: it answers
# `ralph config get` for the fixtures' scripts/ralph-config.sh, so no ralph
# on the host's PATH takes part.
# shellcheck source=lib/fake-ralph.sh
. "$PROJECT_ROOT/tests/lib/fake-ralph.sh"
fake_bin="$workdir/fake-bin"
fake_ralph_install "$fake_bin"

# make_repo <name> -- a fresh git repo with scripts/run-verify.sh, the
# secret-scan scripts it needs, and scripts/ralph-config.sh, no language
# packs and no verify.local.sh (so ran_any stays 0 unless a caller adds
# one), and a base "main" branch.
make_repo() {
  _repo="$workdir/$1"
  mkdir -p "$_repo/scripts"
  cp "$PROJECT_ROOT/scripts/run-verify.sh" "$_repo/scripts/run-verify.sh"
  cp "$PROJECT_ROOT/scripts/secret-scan-branch.sh" "$_repo/scripts/secret-scan-branch.sh"
  cp "$PROJECT_ROOT/scripts/secret-scan.sh" "$_repo/scripts/secret-scan.sh"
  cp "$PROJECT_ROOT/scripts/xreview-helpers.sh" "$_repo/scripts/xreview-helpers.sh"
  cp "$PROJECT_ROOT/scripts/ralph-config.sh" "$_repo/scripts/ralph-config.sh"
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

# run_verify <repo> <mode> [<skip answer> [<name=value>...]] -- runs
# run-verify.sh from <repo> with the stand-in ralph first on PATH, the given
# variables added to its environment, and a clean slate for GITHUB_BASE_REF
# (this suite's own CI sets it for real), and captures combined output +
# exit code. The stand-in's log starts empty, it answers "" for
# standard.base_branch (detect_base_branch in secret-scan-branch.sh asks for
# it), and <skip answer> (default false) says how it answers
# verify.skip_branch_secret_scan: a value such as true or false, exit:<code>
# for a failure with that exit code and a line on stderr, or old for a
# ralph older than `config get` (every key then fails as it does).
run_verify() {
  _repo="$1"
  _mode="$2"
  _skip="${3:-false}"
  shift 2
  [ "$#" -eq 0 ] || shift
  fake_ralph_reset "$fake_bin"
  fake_ralph_value "$fake_bin" standard.base_branch ""
  case "$_skip" in
    exit:*)
      fake_ralph_fail "$fake_bin" verify.skip_branch_secret_scan "${_skip#exit:}" \
        "config get: ralph.local.toml: [plan_visual].browser must be an absolute path"
      ;;
    old) fake_ralph_old "$fake_bin" ;;
    *) fake_ralph_value "$fake_bin" verify.skip_branch_secret_scan "$_skip" ;;
  esac
  set +e
  (
    cd "$_repo"
    # This suite may itself run under run-test.sh, which exports
    # RALPH_VERIFY_SCOPE=changed for ITSELF -- pin scope to full so a
    # fake repo without scripts/detect-changed-languages.sh does not
    # inherit a "changed" scope fallback that forces docs_only=false.
    unset GITHUB_BASE_REF RALPH_SECRET_ALLOWLIST
    env PATH="$fake_bin:$PATH" RALPH_VERIFY_SCOPE=full HARNESS_VERIFY_MODE="$_mode" "$@" ./scripts/run-verify.sh
  ) >"$out_file" 2>&1
  run_rc=$?
  set -e
}

# skip_calls -- the stand-in's log lines for verify.skip_branch_secret_scan.
skip_calls() {
  fake_ralph_calls "$fake_bin" | grep -F 'config get verify.skip_branch_secret_scan' || true
}

# assert_skip_read <desc> <repo> <times> -- verify.skip_branch_secret_scan
# was read <times> times (0 or 1), in <repo>'s root.
assert_skip_read() {
  _want=""
  if [ "$3" -eq 1 ]; then
    _want="$(cd "$2" && pwd -P)|config get verify.skip_branch_secret_scan"
  fi
  _got="$(skip_calls)"
  if [ "$_got" = "$_want" ]; then
    ok "$1"
  else
    not_ok "$1 (log '$_got', want '$_want')"
  fi
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
# Edge case: test_mode_routing_clean above only proves test mode is silent
# on a CLEAN branch. This proves the stronger claim AC-4 actually makes --
# test mode never invokes the branch scan even on a branch that WOULD fail
# it -- by running mode=test against a secret-carrying branch and asserting
# both a clean exit and the absence of any scan output.
# ---------------------------------------------------------------------------
test_mode_test_never_runs_even_with_a_finding() {
  repo="$(make_repo mode-test-with-finding)"
  (cd "$repo" && git checkout -q -b feature && add_secret_commit)

  run_verify "$repo" test
  assert_exit "test mode, secret-carrying branch: still exits 0" 0
  assert_not_contains "test mode+finding: does not run the branch scan" "Running branch secret scan"
  assert_not_contains "test mode+finding: no findings text leaks through" "findings"
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
# Only [verify].skip_branch_secret_scan = true skips the scan (plan
# 2026-10-30-script-local-config). false, exit 127 (what ralph-config.sh
# gets when ralph is not on PATH), a ralph older than `config get`, and a
# settings error (exit 3, after which ralph_setting gives the default false
# with a line on stderr) run it: an unforgiving
# reading is the safe direction for a leak gate. The setting is read once,
# in the repository's root, and only in the modes that run the scan.
# RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN, which earlier versions read, skips
# nothing.
# ---------------------------------------------------------------------------
test_skip_setting() {
  repo="$(make_repo skip-setting)"
  (cd "$repo" && git checkout -q -b feature && add_secret_commit)

  run_verify "$repo" static true
  assert_exit "skip setting true: exits 0 despite the secret commit" 0
  assert_contains "skip setting true: reports the skip and the key" \
    "==> Skipping branch secret scan ([verify].skip_branch_secret_scan = true)"
  assert_not_contains "skip setting true: does not run the scan" "Running branch secret scan"
  assert_skip_read "skip setting true: read once, in the root" "$repo" 1

  run_verify "$repo" all true
  assert_exit "skip setting true, all mode: exits 0 despite the secret commit" 0
  assert_contains "skip setting true, all mode: reports the skip" "Skipping branch secret scan"

  run_verify "$repo" test true
  assert_skip_read "test mode: the skip setting is not read" "$repo" 0

  for answer in false exit:127 old exit:3; do
    run_verify "$repo" static "$answer"
    assert_exit "skip answer $answer: does NOT skip, still fails" 1
    assert_contains "skip answer $answer: runs the scan" "Running branch secret scan"
    assert_not_contains "skip answer $answer: does not report a skip" "Skipping branch secret scan"
  done
  assert_contains "skip answer exit:3: ralph-config.sh names the key and the exit code" \
    "ralph config get verify.skip_branch_secret_scan exited 3"
  assert_contains "skip answer exit:3: ralph-config.sh says it uses the default false" \
    "using the default false"

  run_verify "$repo" static false RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1
  assert_exit "RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1 with the setting false: does NOT skip, still fails" 1
  assert_contains "RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1: runs the scan" "Running branch secret scan"
  assert_not_contains "RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1: does not report a skip" "Skipping branch secret scan"

  # A project with no scripts/ralph-config.sh (scaffolded before it
  # existed) reads no setting: the scan runs, and ralph is not asked.
  rm -f "$repo/scripts/ralph-config.sh"
  run_verify "$repo" static true
  assert_exit "no ralph-config.sh: does NOT skip, still fails" 1
  assert_contains "no ralph-config.sh: runs the scan" "Running branch secret scan"
  assert_skip_read "no ralph-config.sh: the skip setting is not read" "$repo" 0
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
test_mode_test_never_runs_even_with_a_finding
test_finding_fails_even_with_ran_any_zero
test_finding_fails_with_ran_any_one
test_skip_setting
test_docs_only_output_unchanged
test_missing_scanner_is_non_fatal

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
