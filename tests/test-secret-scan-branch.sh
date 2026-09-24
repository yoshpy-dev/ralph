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
#
# Hermeticity: HOME/GIT_CONFIG_GLOBAL/GIT_CONFIG_SYSTEM are isolated to a
# throwaway dir (same pattern as tests/test-terraform-gitignore.sh), because
# every fixture repo below runs `git commit`: a host or CI runner with a
# global `commit.gpgsign = true` (or any other global config) would
# otherwise make every commit in this file fail with a gpg-signing error
# unrelated to the script under test.
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

hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_TERMINAL_PROMPT

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

assert_stderr_not_contains() {
  _desc="$1"
  _needle="$2"
  if grep -F -- "$_needle" "$err_file" >/dev/null 2>&1; then
    not_ok "$_desc (unexpectedly present in stderr: $_needle)"
    printf '%s\n' "--- stderr ---"
    cat "$err_file"
  else
    ok "$_desc"
  fi
}

# assert_stderr_line_count -- counts lines that literally START WITH
# _prefix (awk's index(), not a regex match, so a prefix containing regex
# metacharacters is still matched literally) and defaults to 0 rather than
# an empty string so `set -eu` cannot trip on a failed/empty command
# substitution.
assert_stderr_line_count() {
  _desc="$1"
  _prefix="$2"
  _want="$3"
  _got="$(awk -v p="$_prefix" 'index($0, p) == 1 { c++ } END { print c+0 }' "$err_file" 2>/dev/null)" || _got=0
  if [ "$_got" -eq "$_want" ]; then
    ok "$_desc"
  else
    not_ok "$_desc (got $_got lines starting with '$_prefix', want $_want)"
    printf '%s\n' "--- stderr ---"
    cat "$err_file"
  fi
}

# assert_scanned_clean <desc> <base_ref> -- the literal phrase
# "against <ref>: clean" appears ONLY in the success report line
# (`scanned <a>..<b> against <ref>: clean`). A `cannot_scan` reason can
# independently mention the ref name (e.g. "checked refs/remotes/origin/
# <ref>"), so an assertion that only checks for the ref name, or only for
# "clean" or "scanned" in isolation, can pass on a skip instead of a real
# clean scan; this phrase together is what actually distinguishes them.
assert_scanned_clean() {
  _desc="$1"
  _ref="$2"
  assert_stderr_contains "$_desc" "against ${_ref}: clean"
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
  assert_scanned_clean "clean feature branch reports scanned+clean" "main"
  assert_stderr_line_count "clean feature branch prints exactly one status line" "secret-scan-branch:" 1
}

# ---------------------------------------------------------------------------
# On the base branch itself, and an empty range: default mode exits 0 for
# both (early-warning gate, never fails on an unscannable/empty state).
# --strict now exits 3 for both (self-review cycle 1, M2): exit 0 under
# --strict must mean exactly "scanned, clean", never "there was nothing to
# look at".
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
  assert_exit "on base branch: strict mode exits 3" 3
  assert_stderr_contains "on base branch: strict mode keeps the same reason" "nothing to scan"

  (
    cd "$repo"
    git checkout -q -b feature
  )
  run "$repo"
  assert_exit "empty range: default mode exits 0" 0
  assert_stderr_contains "empty range: reports nothing to scan" "nothing to scan"
  run "$repo" --strict
  assert_exit "empty range: strict mode exits 3" 3
  assert_stderr_contains "empty range: strict mode keeps the same reason" "nothing to scan"
}

# ---------------------------------------------------------------------------
# self-review cycle 1 (M2): two ways an operator following the /pr skill
# could previously reach a strict-mode exit 0 while the branch still carried
# unpushed, fixture-carrying commits -- both must now exit 3.
# ---------------------------------------------------------------------------
test_strict_closes_base_override_bypass() {
  repo="$workdir/strict-bypass-base-override"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'BYPASSbaseoverrideabcdefghijklm')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
  )
  # Pointing the base at the CURRENT branch's own name makes
  # "HEAD is the base branch" fire even though feature carries an unpushed
  # commit with a fixture: this used to exit 0 under --strict too.
  run_with_xreview_base "$repo" feature
  assert_exit "base-override bypass: default mode still exits 0" 0
  run_with_xreview_base "$repo" feature --strict
  assert_exit "base-override bypass: strict mode now exits 3" 3
  assert_stderr_contains "base-override bypass: reason is HEAD is the base branch" "HEAD is the base branch"
}

test_strict_closes_local_base_fast_forward_bypass() {
  repo="$workdir/strict-bypass-fast-forward"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'BYPASSfastforwardabcdefghijklmn')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
    # No origin/main ref exists, so fast-forwarding the LOCAL main onto the
    # feature tip makes the range empty (merge-base == HEAD) even though
    # the branch still carries the fixture relative to where main actually
    # was: this used to exit 0 under --strict too.
    git branch -f main feature
  )
  run "$repo"
  assert_exit "fast-forward bypass: default mode still exits 0" 0
  run "$repo" --strict
  assert_exit "fast-forward bypass: strict mode now exits 3" 3
  assert_stderr_contains "fast-forward bypass: reason is no commits between" "no commits between"
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
  # "origin/main" alone is not enough: a cannot_scan regression here would
  # print "cannot scan, no base ref for 'main' (checked refs/remotes/
  # origin/main and refs/heads/main)", which also contains "origin/main"
  # and also exits 0 in default mode.
  assert_scanned_clean "origin preference: report names origin/main" "origin/main"
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
# Edge case: a base branch name containing a slash (e.g. "release/1.0")
# must resolve correctly through both refs/heads/<name> and
# refs/remotes/origin/<name> -- the base name is spliced directly into the
# ref path (`refs/heads/$base_name`), so a naive future refactor that
# tried to parse or split base_name on "/" could break this silently.
# ---------------------------------------------------------------------------
test_base_name_with_slash() {
  repo="$workdir/slash-base"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B "release/1.0"
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'SLASHBASEabcdefghijklmnopqrstuv')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
  )
  run_with_xreview_base "$repo" "release/1.0"
  assert_exit "slash-containing local base: still detects" 1
  assert_stderr_contains "slash-containing local base: reports the full name" "against release/1.0"

  (
    cd "$repo"
    git update-ref "refs/remotes/origin/release/1.0" "release/1.0"
  )
  run_with_xreview_base "$repo" "release/1.0"
  assert_exit "slash-containing origin base: still detects" 1
  assert_stderr_contains "slash-containing origin base: reports the full name" "against origin/release/1.0"
}

# ---------------------------------------------------------------------------
# AC-3b (cross-review cycle 1, AR-2): a tag named exactly like the base
# branch, pointing at a commit AFTER a secret-adding commit on the feature
# branch, must not divert the range away from the leak. git resolves a
# bare "<base>" through refs/tags/<base> before refs/heads/<base>, so
# passing the short base ref to merge-base picks the tag instead of the
# base branch that was actually validated to exist.
# ---------------------------------------------------------------------------
test_ar2_tag_shadows_base_branch() {
  repo="$workdir/ar2-tag-shadow"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'TAGSHADOWabcdefghijklmnopqrstuv')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
    printf 'advance one\n' >> README.md
    git add README.md
    git commit -q -m 'advance feature (tag point)'
    # A tag named exactly like the base points at this commit, after the
    # leak but before HEAD: resolving the bare base name through
    # refs/tags/main (checked before refs/heads/main) would compute
    # merge-base here instead of at the real main branch, excluding the
    # leaking commit from the range.
    git tag main HEAD
    printf 'advance two\n' >> README.md
    git add README.md
    git commit -q -m 'advance feature (head)'
  )
  run_with_xreview_base "$repo" main
  assert_exit "AC-3b tag shadow: finding still detected" 1
  assert_stderr_contains "AC-3b tag shadow: reports findings" "findings"
}

# ---------------------------------------------------------------------------
# AC-3b (cross-review cycle 1, AR-2): a local branch literally named
# "origin/<base>" must not shadow the actual refs/remotes/origin/<base>
# ref. git resolves a bare "origin/<base>" through
# refs/heads/origin/<base> before refs/remotes/origin/<base>, so passing
# the short base ref to merge-base picks the local branch instead of the
# remote-tracking ref that was actually validated to exist.
# ---------------------------------------------------------------------------
test_ar2_local_branch_shadows_remote_tracking_ref() {
  repo="$workdir/ar2-remote-shadow"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    base_sha="$(git rev-parse HEAD)"

    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'REMOTESHADOWabcdefghijklmnopqrs')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
    leak_sha="$(git rev-parse HEAD)"
    printf 'more clean\n' >> README.md
    git add README.md
    git commit -q -m 'advance feature'

    # refs/remotes/origin/main correctly reflects where the real remote
    # left off (before the leak): the correct range from there includes
    # the leaking commit.
    git update-ref refs/remotes/origin/main "$base_sha"
    # A local branch literally named "origin/main" points at the leak
    # commit itself, so a range computed from it (an exclusive lower
    # bound) would exclude the leak and cover only the later commit.
    git branch origin/main "$leak_sha"
  )
  run_with_xreview_base "$repo" main
  assert_exit "AC-3b remote shadow: finding still detected" 1
  assert_stderr_contains "AC-3b remote shadow: reports findings" "findings"
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
# self-review cycle 1 (LOW): only a scanner exit of 1 means "findings".
# Any other non-zero exit (a scanner bug, a signal) is a scanner failure --
# fail closed (the exit code still propagates and blocks the caller) but
# say so accurately, instead of sending the operator looking for a leak
# that does not exist. Uses a stub secret-scan.sh copied into a temp dir;
# the repo's real scanner is never touched.
# ---------------------------------------------------------------------------
test_scanner_failure_is_not_findings() {
  repo="$workdir/scanner-failure-repo"
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

  for rc in 2 130; do
    copy_dir="$workdir/scanner-failure-copy-$rc"
    mkdir -p "$copy_dir"
    cp "$PROJECT_ROOT/scripts/secret-scan-branch.sh" "$copy_dir/secret-scan-branch.sh"
    cp "$PROJECT_ROOT/scripts/xreview-helpers.sh" "$copy_dir/xreview-helpers.sh"
    printf '#!/usr/bin/env sh\nexit %s\n' "$rc" > "$copy_dir/secret-scan.sh"
    chmod +x "$copy_dir/secret-scan-branch.sh" "$copy_dir/secret-scan.sh"

    run_bin "$copy_dir/secret-scan-branch.sh" "$repo"
    assert_exit "scanner exit $rc: propagated as-is" "$rc"
    assert_stderr_contains "scanner exit $rc: labeled a scanner failure, not findings" "scanner failed with exit $rc"
    assert_stderr_not_contains "scanner exit $rc: never labeled findings" "findings"
  done
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
  assert_scanned_clean "AC-2c: committed allowlist suppresses the finding" "main"
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
  assert_scanned_clean "AC-5: bracket-expression allowlist line is not a finding" "main"
}

# ---------------------------------------------------------------------------
# AC-2d (cross-review cycle 1, AR-1): a committed symlink `.gitallowed`
# resolves its target one level inside the committed tree, the same
# content a checkout of HEAD would show CI's scanner. `git show
# HEAD:.gitallowed` on a symlink entry prints the link's TARGET PATH, not
# file content, so reading it as-is (rather than resolving the target)
# would use that path string as a bogus allowlist rule.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_resolves_target() {
  repo="$workdir/ar1-symlink-resolves"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'symlinkresolve01')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > secret.txt
    bracket_key="$(printf 'api_ke%s' '[y]')"
    printf '%s=%s\n' "$bracket_key" "$val" > allowlist-rules
    ln -s allowlist-rules .gitallowed
    git add secret.txt allowlist-rules .gitallowed
    git commit -q -m 'add fixture and symlinked allowlist'
  )
  run "$repo" --strict
  assert_exit "AC-2d symlink resolves: committed target's rule suppresses the finding" 0
  assert_scanned_clean "AC-2d symlink resolves: reports scanned+clean" "main"
}

# ---------------------------------------------------------------------------
# AC-2d: the same symlinked-allowlist layout, but the committed target
# file does not have a rule matching the fixture: the finding stands.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_denies_fixture() {
  repo="$workdir/ar1-symlink-denies"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'symlinkdenies02')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > secret.txt
    printf '# no matching rule\n' > allowlist-rules
    ln -s allowlist-rules .gitallowed
    git add secret.txt allowlist-rules .gitallowed
    git commit -q -m 'add fixture and a symlinked allowlist with no matching rule'
  )
  run "$repo"
  assert_exit "AC-2d symlink denies: finding stands" 1
}

# ---------------------------------------------------------------------------
# AC-2d: a committed .gitallowed symlink whose target does not exist in
# the committed tree (dangling) scans with an empty allowlist and a
# notice, in both directions: a fixture is still found, and a clean branch
# still reports clean.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_dangling() {
  repo="$workdir/ar1-symlink-dangling"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'symlinkdangle03')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b with-fixture main
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > secret.txt
    ln -s does-not-exist-in-tree .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a dangling symlinked allowlist'

    git checkout -q -b without-fixture main
    ln -s does-not-exist-in-tree .gitallowed
    git add .gitallowed
    git commit -q -m 'add a dangling symlinked allowlist, no fixture'
  )
  (cd "$repo" && git checkout -q with-fixture)
  run "$repo"
  assert_exit "AC-2d dangling symlink: fixture still found" 1
  assert_stderr_contains "AC-2d dangling symlink: could-not-read notice (fixture branch)" "could not be read as a file"

  (cd "$repo" && git checkout -q without-fixture)
  run "$repo"
  assert_exit "AC-2d dangling symlink: clean branch still reports clean" 0
  assert_stderr_contains "AC-2d dangling symlink: could-not-read notice (clean branch)" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# AC-2d (cross-review cycle 1, AR-1's exact reproduction): a symlink whose
# TARGET PATH STRING itself happens to equal the fixture value must not
# suppress the finding. Reading a symlink entry's content as-is (rather
# than resolving it) would use that path string as an allowlist rule; a
# rule equal to the fixture value would then match it as a plain
# substring and hide the leak.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_target_name_not_used_as_regex() {
  repo="$workdir/ar1-symlink-self-name"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'SYMLINKSELFNAMEabcdefghijklmnop')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    printf 'token %s\n' "$token" > secret.txt
    # No file named "$token" is ever committed: the symlink's target does
    # not resolve to anything in the tree.
    ln -s "$token" .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink whose target name matches it'
  )
  run "$repo"
  assert_exit "AC-2d symlink self-name: finding not suppressed by the target path string" 1
  assert_stderr_contains "AC-2d symlink self-name: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# Regression (adjudication after cross-review cycle 1's fix, AC-2d): the
# committed-.gitallowed lookup must be root-relative, not relative to the
# script's own cwd. `git show HEAD:.gitallowed` (the pre-existing read
# path) is already root-relative regardless of cwd; `git ls-tree HEAD --
# .gitallowed` (added to read the tree-entry mode) is NOT -- from a
# subdirectory it silently finds nothing unless the pathspec is passed to
# `git ls-tree --full-tree`.
# ---------------------------------------------------------------------------
test_ls_tree_mode_check_is_root_relative() {
  repo="$workdir/ls-tree-root-relative"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'rootrelative04')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    mkdir sub
    printf 'placeholder\n' > sub/placeholder.md
    bracket_key="$(printf 'api_ke%s' '[y]')"
    printf '%s=%s\n' "$bracket_key" "$val" > .gitallowed
    git add README.md sub/placeholder.md .gitallowed
    git commit -q -m init

    git checkout -q -b feature
    key="$(printf 'api%s' '_key')"
    printf '%s=%s\n' "$key" "$val" > secret.txt
    git add secret.txt
    git commit -q -m 'add fixture allowed by the committed .gitallowed'
  )
  run "$repo" --strict
  assert_exit "root-relative ls-tree: root cwd exits 0" 0
  assert_scanned_clean "root-relative ls-tree: root cwd reports clean" "main"
  assert_stderr_line_count "root-relative ls-tree: root cwd has no ignore notice" "secret-scan-branch: ignoring" 0

  run "$repo/sub" --strict
  assert_exit "root-relative ls-tree: subdirectory cwd exits 0 (same as root)" 0
  assert_scanned_clean "root-relative ls-tree: subdirectory cwd reports clean (same as root)" "main"
  assert_stderr_line_count "root-relative ls-tree: subdirectory cwd has no ignore notice (same as root)" "secret-scan-branch: ignoring" 0
}

# ---------------------------------------------------------------------------
# Regression (adjudication after cross-review cycle 1's fix, AC-2d): a
# .gitallowed symlink whose target names a DIRECTORY (with a trailing
# slash) must not be treated as a file. `git ls-tree --full-tree HEAD --
# <dir>/` lists the directory's CHILDREN rather than failing, so without
# requiring the resolved entry's own recorded path to equal the target
# exactly, the mode check would pass on the first child's mode while the
# content read next (`git show HEAD:<dir>/`, a tree listing) is not a
# file, and a line from that listing could end up used as an allowlist
# regex.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_directory_target_rejected() {
  repo="$workdir/ar1-symlink-dir-target"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'DIRTARGETabcdefghijklmnopqrstuv')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    mkdir rules
    printf 'irrelevant\n' > rules/deploy
    git add README.md rules/deploy
    git commit -q -m init

    git checkout -q -b feature
    printf 'deploy token %s\n' "$token" > secret.txt
    # The symlink's target names a directory (trailing slash): `git
    # ls-tree` on that pathspec lists the directory's children
    # (rules/deploy, mode 100644) instead of failing.
    ln -s rules/ .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink targeting a directory'
  )
  run "$repo"
  assert_exit "symlink directory target: finding not suppressed" 1
  assert_stderr_contains "symlink directory target: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# Pin: a symlink target of "." (a directory containing a file whose name
# is a word from the leak line) must also be rejected. `git ls-tree HEAD
# -- .` lists the tree root's own children, none of which is itself
# recorded as ".", so the exact-path-match requirement already rejects
# this on its own; this test pins that behavior.
# ---------------------------------------------------------------------------
test_ar1_symlinked_allowlist_dot_target_rejected() {
  repo="$workdir/ar1-symlink-dot-target"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'DOTTARGETabcdefghijklmnopqrstuv')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    printf 'deploy\n' > deploy
    git add README.md deploy
    git commit -q -m init

    git checkout -q -b feature
    printf 'deploy token %s\n' "$token" > secret.txt
    ln -s . .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink targeting the tree root'
  )
  run "$repo"
  assert_exit "symlink dot target: finding not suppressed" 1
  assert_stderr_contains "symlink dot target: could-not-read notice" "could not be read as a file"
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
  assert_scanned_clean "repo path with a space: clean scan exits 0" "main"
}

test_ac1_deleted_secret_still_found
test_clean_branch
test_on_base_branch_and_empty_range
test_strict_closes_base_override_bypass
test_strict_closes_local_base_fast_forward_bypass
test_origin_preference_and_fallback
test_base_override_reporting
test_base_name_with_slash
test_ar2_tag_shadows_base_branch
test_ar2_local_branch_shadows_remote_tracking_ref
test_cannot_scan_no_base_ref
test_cannot_scan_unrelated_histories
test_cannot_scan_outside_repo
test_cannot_scan_scanner_missing
test_scanner_failure_is_not_findings
test_allowlist_only_committed_at_head
test_allowlist_self_match
test_ar1_symlinked_allowlist_resolves_target
test_ar1_symlinked_allowlist_denies_fixture
test_ar1_symlinked_allowlist_dangling
test_ar1_symlinked_allowlist_target_name_not_used_as_regex
test_ls_tree_mode_check_is_root_relative
test_ar1_symlinked_allowlist_directory_target_rejected
test_ar1_symlinked_allowlist_dot_target_rejected
test_usage_and_space_path

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
