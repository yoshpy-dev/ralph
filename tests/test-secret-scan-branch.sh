#!/usr/bin/env sh
# Regression tests for scripts/secret-scan-branch.sh: the local mirror of
# the CI branch-history secret scan (.github/workflows/verify.yml).
#
# All fixture values are assembled at test RUNTIME from split pieces (never
# a whole secret-looking literal on one source line): this file's own
# history is read by the same branch-history scan it tests.
#
# Every invocation below explicitly unsets GITHUB_BASE_REF, RALPH_XREVIEW_BASE,
# and RALPH_SECRET_ALLOWLIST before running: this suite's own CI run sets
# GITHUB_BASE_REF for real, and a leaked value would silently pick the wrong
# base branch or allowlist for these ad hoc test repos.
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"
SCANNER_BRANCH="$PROJECT_ROOT/scripts/secret-scan-branch.sh"

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

out_file="$(mktemp "${TMPDIR:-/tmp}/ralph-secret-scan-branch-test.out.XXXXXX")"
err_file="$(mktemp "${TMPDIR:-/tmp}/ralph-secret-scan-branch-test.err.XXXXXX")"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph secret scan branch.XXXXXX")"
trap 'rm -rf "$workdir" "$out_file" "$err_file"' EXIT HUP INT TERM

# run_bin <binary> <cwd> [args...] -- runs <binary> from <cwd> with a clean
# slate for the three env vars this script reads. Captures stdout/stderr
# into out_file/err_file and the exit code into run_rc.
run_bin() {
  _bin="$1"
  _cwd="$2"
  shift 2
  set +e
  (
    cd "$_cwd" || exit 127
    unset GITHUB_BASE_REF RALPH_XREVIEW_BASE RALPH_SECRET_ALLOWLIST
    "$_bin" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

run() {
  _cwd="$1"
  shift
  run_bin "$SCANNER_BRANCH" "$_cwd" "$@"
}

run_outside() {
  _cwd="$1"
  shift
  set +e
  (
    cd "$_cwd" || exit 127
    unset GITHUB_BASE_REF RALPH_XREVIEW_BASE RALPH_SECRET_ALLOWLIST
    GIT_CEILING_DIRECTORIES="$(dirname "$_cwd")" "$SCANNER_BRANCH" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

run_with_github_base_ref() {
  _cwd="$1"
  _base="$2"
  shift 2
  set +e
  (
    cd "$_cwd" || exit 127
    unset RALPH_XREVIEW_BASE RALPH_SECRET_ALLOWLIST
    GITHUB_BASE_REF="$_base" "$SCANNER_BRANCH" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

run_with_xreview_base() {
  _cwd="$1"
  _base="$2"
  shift 2
  set +e
  (
    cd "$_cwd" || exit 127
    unset GITHUB_BASE_REF RALPH_SECRET_ALLOWLIST
    RALPH_XREVIEW_BASE="$_base" "$SCANNER_BRANCH" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

run_with_allowlist_override() {
  _cwd="$1"
  _allow="$2"
  shift 2
  set +e
  (
    cd "$_cwd" || exit 127
    unset GITHUB_BASE_REF RALPH_XREVIEW_BASE
    RALPH_SECRET_ALLOWLIST="$_allow" "$SCANNER_BRANCH" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

assert_exit() {
  _desc="$1"
  _want="$2"
  if [ "$run_rc" -eq "$_want" ]; then
    ok "$_desc (exit $run_rc)"
  else
    not_ok "$_desc (exit $run_rc, want $_want)"
    printf '%s\n' "--- stdout ---"
    cat "$out_file"
    printf '%s\n' "--- stderr ---"
    cat "$err_file"
  fi
}

assert_stderr_contains() {
  _desc="$1"
  _needle="$2"
  if grep -F -- "$_needle" "$err_file" >/dev/null 2>&1; then
    ok "$_desc"
  else
    not_ok "$_desc (missing in stderr: $_needle)"
    printf '%s\n' "--- stderr ---"
    cat "$err_file"
  fi
}

assert_stderr_line_count() {
  _desc="$1"
  _prefix="$2"
  _want="$3"
  _got="$(grep -c -- "$_prefix" "$err_file" 2>/dev/null || true)"
  if [ "$_got" -eq "$_want" ]; then
    ok "$_desc"
  else
    not_ok "$_desc (got $_got lines starting with '$_prefix', want $_want)"
    printf '%s\n' "--- stderr ---"
    cat "$err_file"
  fi
}

git_repo() {
  _dir="$1"
  mkdir -p "$_dir"
  (cd "$_dir" && git init -q && git config user.email test@example.com && git config user.name "Secret Scan Branch Test")
}

# ---------------------------------------------------------------------------
# AC-1: an early commit adds a secret-looking line, a later commit deletes
# it -- the range scan still finds it via history.
# ---------------------------------------------------------------------------
test_ac1_deleted_secret_still_found() {
  repo="$workdir/ac1"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'abcdefghijklmnopqrstuvwxyzABCDE')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
    rm leaked.txt
    git add leaked.txt
    git commit -q -m 'remove leaked token'
  )
  run "$repo"
  assert_exit "AC-1: deleted secret still found by history scan" 1
  assert_stderr_contains "AC-1: reports findings" "findings"
}

# ---------------------------------------------------------------------------
# Clean feature branch scans clean and reports the scanned range as a
# single status line.
# ---------------------------------------------------------------------------
test_clean_branch() {
  repo="$workdir/clean"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    printf 'still clean\n' >> README.md
    git add README.md
    git commit -q -m 'clean change'
  )
  run "$repo"
  assert_exit "clean feature branch exits 0" 0
  assert_stderr_contains "clean feature branch reports scanned+clean" "scanned"
  assert_stderr_contains "clean feature branch reports clean verdict" "clean"
  assert_stderr_line_count "clean feature branch prints exactly one status line" "secret-scan-branch:" 1
}

# ---------------------------------------------------------------------------
# On the base branch itself, and an empty range: both exit 0 in both modes.
# ---------------------------------------------------------------------------
test_on_base_branch_and_empty_range() {
  repo="$workdir/onbase"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
  )
  run "$repo"
  assert_exit "on base branch: default mode exits 0" 0
  assert_stderr_contains "on base branch: reports nothing to scan" "nothing to scan"
  assert_stderr_line_count "on base branch: exactly one status line" "secret-scan-branch:" 1
  run "$repo" --strict
  assert_exit "on base branch: strict mode exits 0" 0

  (
    cd "$repo"
    git checkout -q -b feature
  )
  run "$repo"
  assert_exit "empty range: default mode exits 0" 0
  assert_stderr_contains "empty range: reports nothing to scan" "nothing to scan"
  run "$repo" --strict
  assert_exit "empty range: strict mode exits 0" 0
}

# ---------------------------------------------------------------------------
# No origin remote falls back to the local base branch and still detects a
# finding; adding an origin/<base> ref that predates the leak (while local
# <base> postdates it) proves origin is preferred over the local ref.
# ---------------------------------------------------------------------------
test_origin_preference_and_fallback() {
  repo="$workdir/origin-pref"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b leak
    token="$(printf 'sk_live_%s' 'abcdefghijklmnopqrstuv')"
    printf 'stripe_secret=%s\n' "$token" > leak.txt
    git add leak.txt
    git commit -q -m 'add secret'
    git rev-parse HEAD > "$workdir/leak_sha"
    rm leak.txt
    git add leak.txt
    git commit -q -m 'remove secret'

    git checkout -q main
    printf 'more clean\n' >> README.md
    git add README.md
    git commit -q -m 'advance main'

    git checkout -q leak
  )
  # No origin ref yet: falls back to local main, merge-base is the initial
  # commit, and the range includes the secret-adding commit.
  run "$repo"
  assert_exit "no origin remote: falls back to local base, detects" 1

  leak_sha="$(cat "$workdir/leak_sha")"
  (
    cd "$repo"
    git update-ref refs/remotes/origin/main "$leak_sha"
  )
  # origin/main now points at the secret-adding commit itself, so
  # merge-base(leak, origin/main) == that commit and the range only covers
  # the later "remove secret" commit: clean. This proves origin/<base> is
  # preferred over local <base> (which would still find the secret).
  run "$repo"
  assert_exit "origin/<base> ref present: preferred over local, clean" 0
  assert_stderr_contains "origin preference: report names origin/main" "origin/main"
}

# ---------------------------------------------------------------------------
# GITHUB_BASE_REF and RALPH_XREVIEW_BASE both select an alternate base, and
# the report line names the base actually used.
# ---------------------------------------------------------------------------
test_base_override_reporting() {
  repo="$workdir/base-override"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b develop
    printf 'develop clean\n' >> README.md
    git add README.md
    git commit -q -m 'develop change'

    git checkout -q -b feature
    printf 'feature clean\n' >> README.md
    git add README.md
    git commit -q -m 'feature change'
  )
  run "$repo"
  assert_exit "default base resolution exits 0" 0
  assert_stderr_contains "default base resolution uses main" "against main"

  run_with_github_base_ref "$repo" develop
  assert_exit "GITHUB_BASE_REF=develop exits 0" 0
  assert_stderr_contains "GITHUB_BASE_REF selects develop" "against develop"

  run_with_xreview_base "$repo" develop
  assert_exit "RALPH_XREVIEW_BASE=develop exits 0" 0
  assert_stderr_contains "RALPH_XREVIEW_BASE selects develop" "against develop"
}

# ---------------------------------------------------------------------------
# Cannot-scan states: default mode exits 0 with a reason, --strict exits 3.
# ---------------------------------------------------------------------------
test_cannot_scan_no_base_ref() {
  repo="$workdir/no-base-ref"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    printf 'change\n' >> README.md
    git add README.md
    git commit -q -m change
  )
  run_with_xreview_base "$repo" does-not-exist-anywhere
  assert_exit "no base ref: default mode exits 0" 0
  assert_stderr_contains "no base ref: reports cannot scan" "cannot scan"
  run_with_xreview_base "$repo" does-not-exist-anywhere --strict
  assert_exit "no base ref: strict mode exits 3" 3
}

test_cannot_scan_unrelated_histories() {
  repo="$workdir/orphan"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q --orphan unrelated
    printf 'unrelated\n' > other.md
    git add other.md
    git commit -q -m 'unrelated root commit'
  )
  run "$repo"
  assert_exit "unrelated histories: default mode exits 0" 0
  assert_stderr_contains "unrelated histories: reports cannot scan" "cannot scan"
  assert_stderr_line_count "unrelated histories: exactly one status line" "secret-scan-branch:" 1
  run "$repo" --strict
  assert_exit "unrelated histories: strict mode exits 3" 3
}

test_cannot_scan_outside_repo() {
  outside="$workdir/outside-repo"
  mkdir -p "$outside"
  run_outside "$outside"
  assert_exit "outside git work tree: default mode exits 0" 0
  assert_stderr_contains "outside git work tree: reports cannot scan" "cannot scan"
  run_outside "$outside" --strict
  assert_exit "outside git work tree: strict mode exits 3" 3
}

test_cannot_scan_scanner_missing() {
  repo="$workdir/scanner-missing-repo"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    printf 'change\n' >> README.md
    git add README.md
    git commit -q -m change
  )

  copy_dir="$workdir/scanner-missing-copy"
  mkdir -p "$copy_dir"
  cp "$PROJECT_ROOT/scripts/secret-scan-branch.sh" "$copy_dir/secret-scan-branch.sh"
  cp "$PROJECT_ROOT/scripts/xreview-helpers.sh" "$copy_dir/xreview-helpers.sh"
  cp "$PROJECT_ROOT/scripts/secret-scan.sh" "$copy_dir/secret-scan.sh"
  chmod +x "$copy_dir/secret-scan-branch.sh"
  chmod -x "$copy_dir/secret-scan.sh"

  run_bin "$copy_dir/secret-scan-branch.sh" "$repo"
  assert_exit "scanner not executable: default mode exits 0" 0
  assert_stderr_contains "scanner not executable: reports cannot scan" "cannot scan"
  run_bin "$copy_dir/secret-scan-branch.sh" "$repo" --strict
  assert_exit "scanner not executable: strict mode exits 3" 3
}

# ---------------------------------------------------------------------------
# AC-2c: only the .gitallowed committed at HEAD suppresses a finding.
# ---------------------------------------------------------------------------
test_allowlist_only_committed_at_head() {
  repo="$workdir/allowlist-committed"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'abcdef123456')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > secret.txt
    git add secret.txt
    git commit -q -m 'add secret'
  )
  run "$repo"
  assert_exit "AC-2c: finding before any allowlist" 1

  bracket_key="$(printf 'api_ke%s' '[y]')"
  other_allow="$workdir/other-allowlist"
  printf '%s=%s\n' "$bracket_key" "$val" > "$other_allow"
  (
    cd "$repo"
    printf '%s=%s\n' "$bracket_key" "$val" > .gitallowed
  )
  # Uncommitted .gitallowed edit + a RALPH_SECRET_ALLOWLIST override, both
  # of which would suppress the finding if honored: neither is, since CI
  # only reads the .gitallowed committed at HEAD.
  run_with_allowlist_override "$repo" "$other_allow"
  assert_exit "AC-2c: uncommitted allowlist + env override still finds" 1
  assert_stderr_contains "AC-2c: reports the ignore notice" "ignoring"

  (
    cd "$repo"
    git add .gitallowed
    git commit -q -m 'allow the fixture value'
  )
  run "$repo"
  assert_exit "AC-2c: committed allowlist suppresses the finding" 0
}

# ---------------------------------------------------------------------------
# AC-5 (first half): a self-matching .gitallowed line is itself a finding;
# a bracket-expression rewrite of the same rule is not.
#
# secret-scan-branch.sh always scans with the .gitallowed committed at
# CURRENT HEAD, applied uniformly across every commit in the range (not
# per-commit history state). So a self-matching rule only surfaces as a
# finding once HEAD's own .gitallowed no longer contains it -- exactly the
# PR #168 shape: a later commit "fixes" .gitallowed forward, but the range
# scan still reads the commit that introduced the bad line.
# ---------------------------------------------------------------------------
test_allowlist_self_match() {
  repo="$workdir/allowlist-self-match"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
  )
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'ghijkl654321')"

  (
    cd "$repo"
    git checkout -q -b self-match-bad main
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > .gitallowed
    git add .gitallowed
    git commit -q -m 'allowlist rule (self-matching, not bracketed)'
    : > .gitallowed
    git add .gitallowed
    git commit -q -m 'empty out the allowlist again'
  )
  run "$repo"
  assert_exit "AC-5: self-matching allowlist line is itself a finding" 1

  (
    cd "$repo"
    git checkout -q -b self-match-safe main
    bracket_key="$(printf 'api_ke%s' '[y]')"
    printf '%s=%s\n' "$bracket_key" "$val" > .gitallowed
    git add .gitallowed
    git commit -q -m 'allowlist rule (bracket-expression, self-safe)'
  )
  run "$repo"
  assert_exit "AC-5: bracket-expression allowlist line is not a finding" 0
}

# ---------------------------------------------------------------------------
# Usage errors, and correct behavior under a repo path containing a space.
# ---------------------------------------------------------------------------
test_usage_and_space_path() {
  run "$workdir" --bogus-flag
  assert_exit "unknown flag exits 2" 2
  run "$workdir" --strict extra-arg
  assert_exit "too many args exits 2" 2

  repo="$workdir/space repo"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    printf 'space path change\n' >> README.md
    git add README.md
    git commit -q -m 'clean change under a space path'
  )
  run "$repo"
  assert_exit "repo path with a space: clean scan exits 0" 0
}

test_ac1_deleted_secret_still_found
test_clean_branch
test_on_base_branch_and_empty_range
test_origin_preference_and_fallback
test_base_override_reporting
test_cannot_scan_no_base_ref
test_cannot_scan_unrelated_histories
test_cannot_scan_outside_repo
test_cannot_scan_scanner_missing
test_allowlist_only_committed_at_head
test_allowlist_self_match
test_usage_and_space_path

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
