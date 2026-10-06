#!/usr/bin/env sh
# tests/test-verify-local-hook-tests.sh — run_hook_tests in
# scripts/verify.local.sh counts a tests/test-*.sh that would not run in CI
# as FAIL, names the file and the fix, and makes verify.local.sh exit
# non-zero, instead of silently skipping it:
#   1. not executable in the working tree           -> FAIL
#   2. executable, but the git index mode is 100644  -> FAIL
#   3. executable, and the git index mode is 100755  -> runs, exit 0
#   4. executable and untracked inside a git repo    -> runs, exit 0
#      (no index mode; the working-tree bit decides)
# Each case copies verify.local.sh into its own fixture under mktemp -d
# (the script cd's to its own ..) and runs it with HARNESS_VERIFY_MODE=test.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERIFY_LOCAL="${PROJECT_ROOT}/scripts/verify.local.sh"

# The fixtures are their own git repos; keep git from following a repo that
# the caller (a git hook, a worktree wrapper) pointed it at.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR GIT_OBJECT_DIRECTORY

TMP_ROOT="$(mktemp -d)"
trap 'rm -r -f "$TMP_ROOT"' EXIT
trap 'exit 1' HUP INT TERM

_pass=0
_fail=0
_total=0

pass() {
  _total=$((_total + 1))
  _pass=$((_pass + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _total=$((_total + 1))
  _fail=$((_fail + 1))
  printf '  FAIL: %s\n' "$1"
}

# make_fixture NAME — create a fixture with scripts/verify.local.sh, an
# executable passing tests/test-a.sh, and a passing tests/test-b.sh with no
# exec bit. Prints the fixture directory.
make_fixture() {
  _dir="${TMP_ROOT}/$1"
  mkdir -p "${_dir}/scripts" "${_dir}/tests"
  cp "$VERIFY_LOCAL" "${_dir}/scripts/verify.local.sh"
  printf '#!/bin/sh\nexit 0\n' > "${_dir}/tests/test-a.sh"
  printf '#!/bin/sh\nexit 0\n' > "${_dir}/tests/test-b.sh"
  chmod 755 "${_dir}/tests/test-a.sh"
  chmod 644 "${_dir}/tests/test-b.sh"
  printf '%s\n' "$_dir"
}

# init_fixture_repo DIR — make DIR a git repo with its files staged.
init_fixture_repo() {
  git -C "$1" init -q
  git -C "$1" config user.name 'verify-local test'
  git -C "$1" config user.email 'verify-local-test@example.invalid'
  git -C "$1" add scripts tests
}

# index_mode DIR PATH — print the git index mode of PATH in DIR.
index_mode() {
  git -C "$1" ls-files -s -- "$2" | awk 'NR == 1 { print $1 }'
}

# run_verify_local DIR — run the fixture's verify.local.sh in test mode.
# Sets _rc to its exit code and _out to the file holding its output.
run_verify_local() {
  _out="$1.out"
  _rc=0
  HARNESS_VERIFY_MODE='test' sh "$1/scripts/verify.local.sh" > "$_out" 2>&1 || _rc=$?
}

# output_has TEXT — the last run_verify_local output contains TEXT.
output_has() {
  grep -F -- "$1" "$_out" >/dev/null 2>&1
}

show_output() {
  sed 's/^/      | /' "$_out"
}

# expect_mode_fail DESC REASON — the last run exited non-zero and printed
# the FAIL line with REASON for tests/test-b.sh plus both fix commands,
# while tests/test-a.sh still ran.
expect_mode_fail() {
  _desc="$1"
  _reason="$2"
  if [ "$_rc" -eq 0 ]; then
    fail "${_desc}: verify.local.sh exited 0, expected non-zero"
    show_output
    return 0
  fi
  _missing=""
  for _needle in \
    "FAIL: ${_reason} (tests/test-b.sh)" \
    "fix: chmod +x tests/test-b.sh" \
    "git update-index --chmod=+x tests/test-b.sh" \
    "==> tests/test-a.sh"
  do
    output_has "$_needle" || _missing="${_missing} [${_needle}]"
  done
  if [ -n "$_missing" ]; then
    fail "${_desc}: output lacks${_missing}"
    show_output
    return 0
  fi
  pass "${_desc} (exit ${_rc}, file and fix named)"
}

# expect_runs DESC — the last run exited 0 and ran tests/test-b.sh.
expect_runs() {
  _desc="$1"
  if [ "$_rc" -ne 0 ]; then
    fail "${_desc}: verify.local.sh exited ${_rc}, expected 0"
    show_output
    return 0
  fi
  if ! output_has "==> tests/test-b.sh" || output_has "FAIL"; then
    fail "${_desc}: tests/test-b.sh did not run cleanly"
    show_output
    return 0
  fi
  pass "${_desc} (exit 0, tests/test-b.sh ran)"
}

printf '==> run_hook_tests treats a test that would not run in CI as FAIL\n'

# Case 1: no exec bit in the working tree, outside any git repo.
_fx="$(make_fixture not-executable)"
run_verify_local "$_fx"
expect_mode_fail "not executable in the working tree" "not executable in the working tree"

# Case 2: exec bit in the working tree, index mode 100644.
_fx="$(make_fixture index-644)"
chmod 755 "${_fx}/tests/test-b.sh"
init_fixture_repo "$_fx"
git -C "$_fx" update-index --chmod=-x tests/test-b.sh
_mode="$(index_mode "$_fx" tests/test-b.sh)"
if [ "$_mode" != "100644" ]; then
  fail "index mode 100644: fixture setup left index mode '${_mode}'"
else
  run_verify_local "$_fx"
  if output_has "not executable in the working tree"; then
    fail "index mode 100644: reported a working-tree problem for an executable file"
    show_output
  else
    expect_mode_fail "executable, index mode 100644" "git index mode is 100644"
  fi
fi

# Case 3: exec bit in the working tree, index mode 100755.
_fx="$(make_fixture index-755)"
chmod 755 "${_fx}/tests/test-b.sh"
init_fixture_repo "$_fx"
git -C "$_fx" update-index --chmod=+x tests/test-b.sh
_mode="$(index_mode "$_fx" tests/test-b.sh)"
if [ "$_mode" != "100755" ]; then
  fail "index mode 100755: fixture setup left index mode '${_mode}'"
else
  run_verify_local "$_fx"
  expect_runs "executable, index mode 100755"
fi

# Case 4: exec bit in the working tree, untracked inside a git repo.
_fx="$(make_fixture untracked)"
chmod 755 "${_fx}/tests/test-b.sh"
git -C "$_fx" init -q
_mode="$(index_mode "$_fx" tests/test-b.sh)"
if [ -n "$_mode" ]; then
  fail "untracked: fixture setup tracked tests/test-b.sh with mode '${_mode}'"
else
  run_verify_local "$_fx"
  expect_runs "executable, untracked in a git repo"
fi

printf '\nverify-local-hook-tests tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
