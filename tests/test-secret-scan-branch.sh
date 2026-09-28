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
# unrelated to the script under test. GIT_CONFIG_NOSYSTEM also covers a git
# older than 2.32, which ignores GIT_CONFIG_SYSTEM, and XDG_CONFIG_HOME is
# unset so no user-level git config or attributes file under it is read.
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
GIT_CONFIG_NOSYSTEM=1
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT
unset XDG_CONFIG_HOME RALPH_SECRET_SCAN_ATTR_SOURCE GIT_ATTR_SOURCE

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

# run_with_env <cwd> <NAME=value> [args...] -- like run, with one extra
# environment assignment: a PATH that puts a stub git first, or an
# inherited RALPH_SECRET_SCAN_ATTR_SOURCE.
run_with_env() {
  _cwd="$1"
  _assign="$2"
  shift 2
  set +e
  (
    cd "$_cwd" || exit 127
    unset GITHUB_BASE_REF RALPH_XREVIEW_BASE RALPH_SECRET_ALLOWLIST
    env "$_assign" "$SCANNER_BRANCH" "$@" >"$out_file" 2>"$err_file"
  )
  run_rc=$?
  set -e
}

# The merge-derived attribute cases need a real git 2.41 or later, which
# GIT_ATTR_SOURCE needs; on an older git the script refuses them by design
# (test_attr_real_old_git checks that instead). The version is read the
# same way tests/test-secret-scan.sh reads it.
git_version="$(git --version | awk '{ print $3 }')"
git_major=${git_version%%.*}
git_minor=${git_version#*.}
git_minor=${git_minor%%.*}
if [ "$git_major" -gt 2 ] || { [ "$git_major" -eq 2 ] && [ "$git_minor" -ge 41 ]; }; then
  merge_attr_supported=1
else
  merge_attr_supported=0
fi

# skip_without_merge_attr <description> -- prints a SKIP line and succeeds
# when the host git is older than 2.41, so the caller returns before a
# case that needs the merge-derived path.
skip_without_merge_attr() {
  [ "$merge_attr_supported" -eq 0 ] || return 1
  printf '  SKIP: %s (git %s is older than 2.41)\n' "$1" "$git_version"
}

# Stub binaries for the attribute-source cases. Each one handles a single
# call, matched on its whole argument list since the script puts -c
# options before the subcommand, and runs the real binary for everything
# else.
real_git="$(command -v git)"
stub_merge_tree_fails="$workdir/stub-merge-tree-fails"
stub_old_git="$workdir/stub-old-git"
stub_no_write_tree="$workdir/stub-no-write-tree"
stub_driver_list_fails="$workdir/stub-driver-list-fails"
stub_attr_diff_fails="$workdir/stub-attr-diff-fails"
stub_malformed_version="$workdir/stub-malformed-version"
stub_empty_version="$workdir/stub-empty-version"
stub_grep_fails="$workdir/stub-grep-fails"
no_stub="$workdir/no-stub"
mkdir -p "$stub_merge_tree_fails" "$stub_old_git" "$stub_no_write_tree" \
  "$stub_driver_list_fails" "$stub_attr_diff_fails" "$stub_malformed_version" \
  "$stub_empty_version" "$stub_grep_fails" "$no_stub"
cat > "$stub_merge_tree_fails/git" <<EOF
#!/bin/sh
case " \$* " in
  *" merge-tree --write-tree "*) printf 'fatal: stub merge-tree failure\n' >&2; exit 128 ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_old_git/git" <<EOF
#!/bin/sh
case "\$1" in
  version | --version) printf 'git version 2.40.1\n'; exit 0 ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_no_write_tree/git" <<EOF
#!/bin/sh
case " \$* " in
  *" merge-tree "*)
    printf 'usage: git merge-tree <base-tree> <branch1> <branch2>\n' >&2
    exit 129
    ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_driver_list_fails/git" <<EOF
#!/bin/sh
case " \$* " in
  *" --get-regexp ^merge"*) exit 5 ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_attr_diff_fails/git" <<EOF
#!/bin/sh
case " \$* " in
  *" diff --quiet "*) exit 5 ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_malformed_version/git" <<EOF
#!/bin/sh
case "\$1" in
  version | --version) printf 'git version 2.x\n'; exit 0 ;;
esac
exec "$real_git" "\$@"
EOF
cat > "$stub_empty_version/git" <<EOF
#!/bin/sh
case "\$1" in
  version | --version) exit 0 ;;
esac
exec "$real_git" "\$@"
EOF
real_grep="$(command -v grep)"
cat > "$stub_grep_fails/grep" <<EOF
#!/bin/sh
case "\$1" in
  -Eqv) exit 2 ;;
esac
exec "$real_grep" "\$@"
EOF
chmod +x "$stub_merge_tree_fails/git" "$stub_old_git/git" "$stub_no_write_tree/git" \
  "$stub_driver_list_fails/git" "$stub_attr_diff_fails/git" "$stub_malformed_version/git" \
  "$stub_empty_version/git" "$stub_grep_fails/grep"
# A stub mktemp fails only for the merge-tree error file, so the allowlist
# temp file before it is still created.
real_mktemp="$(command -v mktemp)"
stub_merge_mktemp_fails="$workdir/stub-merge-mktemp-fails"
mkdir -p "$stub_merge_mktemp_fails"
cat > "$stub_merge_mktemp_fails/mktemp" <<EOF
#!/bin/sh
case "\$*" in
  *ralph-secret-scan-branch-merge*) exit 1 ;;
esac
exec "$real_mktemp" "\$@"
EOF
chmod +x "$stub_merge_mktemp_fails/mktemp"

# attr_base_adds_repo <dir> -- feature (HEAD) forks from main and adds a
# token in leak.txt; main then commits a .gitattributes marking *.txt
# -diff. HEAD's attributes let the scan find the token; the merge result's
# (CI's pull_request checkout) hide it.
attr_base_adds_repo() {
  git_repo "$1"
  (
    cd "$1"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'ATTRMERGEabcdefghijklmnopqrstuv')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add leak.txt
    git commit -q -m 'add leaked token'
    git checkout -q main
    printf '*.txt -diff\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'mark txt files -diff'
    git checkout -q feature
  )
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
test_ac3b_tag_shadows_base_branch() {
  repo="$workdir/ac3b-tag-shadow"
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
test_ac3b_local_branch_shadows_remote_tracking_ref() {
  repo="$workdir/ac3b-remote-shadow"
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
# content a checkout of HEAD would show CI's scanner. Reading
# HEAD:.gitallowed on a symlink entry yields the link's TARGET PATH, not
# file content, so reading it as-is (rather than resolving the target)
# would use that path string as a bogus allowlist rule.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_resolves_target() {
  repo="$workdir/ac2d-symlink-resolves"
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
test_ac2d_symlinked_allowlist_denies_fixture() {
  repo="$workdir/ac2d-symlink-denies"
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
test_ac2d_symlinked_allowlist_dangling() {
  repo="$workdir/ac2d-symlink-dangling"
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
test_ac2d_symlinked_allowlist_target_name_not_used_as_regex() {
  repo="$workdir/ac2d-symlink-self-name"
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
# script's own cwd. HEAD:.gitallowed (the object name the content read
# uses) is already root-relative regardless of cwd; `git ls-tree HEAD --
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
# content read next (HEAD:<dir>/) is a tree, not a file.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_directory_target_rejected() {
  repo="$workdir/ac2d-symlink-dir-target"
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
test_ac2d_symlinked_allowlist_dot_target_rejected() {
  repo="$workdir/ac2d-symlink-dot-target"
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
# Self-review cycle 2 (C2-M1 / coverage gap 1): a .gitallowed symlink
# whose target names a directory WITHOUT a trailing slash. `git ls-tree
# --full-tree HEAD -- rules` returns exactly one entry -- the directory's
# own tree entry, recorded path "rules" -- so it passes both the
# single-line check and the exact-path check; the mode case (040000 is
# not 100644|100755) is the guard that actually rejects it. This test is
# the one the self-review's mutation matrix showed missing: widening the
# mode case to also accept 040000 must make it fail.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_bare_directory_target_rejected() {
  repo="$workdir/ac2d-symlink-bare-dir"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'BAREDIRabcdefghijklmnopqrstuvwx')"
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
    # No trailing slash: `git ls-tree` resolves this to the directory's
    # own single tree entry, not its children.
    ln -s rules .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink targeting a bare directory'
  )
  run "$repo"
  assert_exit "bare directory target: finding not suppressed" 1
  assert_stderr_contains "bare directory target: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# Self-review cycle 2 (coverage gap 2): a symlink target that is an
# ABSOLUTE path must be rejected by the script's own
# allowlist_target_is_safe check, not merely by git's own pathspec
# refusal for a path outside the repository.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_absolute_target_rejected() {
  repo="$workdir/ac2d-symlink-absolute-target"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'ABSOLUTEabcdefghijklmnopqrstuvw')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    printf 'deploy token %s\n' "$token" > secret.txt
    absolute_target="/$(printf 'scratch-%s' "$$")"
    ln -s "$absolute_target" .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and an absolute-target symlink'
  )
  run "$repo"
  assert_exit "absolute target: finding not suppressed" 1
  assert_stderr_contains "absolute target: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# Self-review cycle 2 (coverage gap 2): a symlink target containing a
# ".." path component must be rejected by allowlist_target_is_safe.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_dotdot_target_rejected() {
  repo="$workdir/ac2d-symlink-dotdot-target"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'DOTDOTabcdefghijklmnopqrstuvwxy')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    printf 'deploy token %s\n' "$token" > secret.txt
    dotdot_target="$(printf '..%sscratch-target' '/')"
    ln -s "$dotdot_target" .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink target containing ..'
  )
  run "$repo"
  assert_exit "dotdot target: finding not suppressed" 1
  assert_stderr_contains "dotdot target: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# Self-review cycle 2 (C2-L2): a symlink target with a leading "./" must
# be rejected by the script's own notice, not surface a raw git error.
# Stripping "./" from ".//x" yields "/x" -- validating BEFORE stripping
# would let this absolute path slip past allowlist_target_is_safe and
# reach git as an out-of-repository pathspec instead.
# ---------------------------------------------------------------------------
test_ac2d_symlinked_allowlist_leading_dotslash_target_rejected() {
  repo="$workdir/ac2d-symlink-dotslash-target"
  git_repo "$repo"
  token="$(printf 'ghp_%s' 'DOTSLASHabcdefghijklmnopqrstuvw')"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init

    git checkout -q -b feature
    printf 'deploy token %s\n' "$token" > secret.txt
    dotslash_target="$(printf './/%s' 'scratch-target')"
    ln -s "$dotslash_target" .gitallowed
    git add secret.txt .gitallowed
    git commit -q -m 'add fixture and a symlink target with a leading ./'
  )
  run "$repo"
  assert_exit "leading ./ target: finding not suppressed" 1
  assert_stderr_contains "leading ./ target: could-not-read notice" "could not be read as a file"
  assert_stderr_not_contains "leading ./ target: no raw git error leaks" "fatal:"
}

# ---------------------------------------------------------------------------
# AC-11 (#176): a symlink target is read byte for byte. The target here is
# "rules" plus a trailing newline -- a different committed file, without
# the rule, which a checkout of HEAD (CI) follows. Command substitution
# would strip the newline and resolve to "rules" (holding the rule)
# instead; a target holding a newline is not resolved at all.
# ---------------------------------------------------------------------------
test_ac11_symlinked_allowlist_target_trailing_newline() {
  repo="$workdir/ac11-symlink-target-newline"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'targetnewline05')"
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
    printf '%s=%s\n' "$bracket_key" "$val" > rules
    target="$(printf 'rules\nx')"
    target="${target%x}"
    printf '# no matching rule\n' > "$target"
    ln -s "$target" .gitallowed
    git add secret.txt rules "$target" .gitallowed
    git commit -q -m 'add fixture and a symlink whose target ends with a newline'
  )
  run "$repo" --strict
  assert_exit "AC-11 target ending with a newline: the newline-less file's rule is not used" 1
  assert_stderr_contains "AC-11 target ending with a newline: could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# AC-12 (#176): a non-ASCII symlink target resolves. `git ls-tree` quotes a
# non-ASCII path under the default core.quotePath, which would never equal
# the target and fall back to an empty allowlist.
# ---------------------------------------------------------------------------
test_ac12_symlinked_allowlist_non_ascii_target() {
  repo="$workdir/ac12-symlink-non-ascii-target"
  git_repo "$repo"
  val="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'nonasciitarget06')"
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
    target="$(printf 'r\303\274les')"
    printf '%s=%s\n' "$bracket_key" "$val" > "$target"
    ln -s "$target" .gitallowed
    git add secret.txt "$target" .gitallowed
    git commit -q -m 'add fixture and a symlink to a non-ASCII allowlist file'
  )
  run "$repo" --strict
  assert_exit "AC-12 non-ASCII target: its rule suppresses the finding" 0
  assert_scanned_clean "AC-12 non-ASCII target: reports scanned+clean" "main"
  assert_stderr_not_contains "AC-12 non-ASCII target: no could-not-read notice" "could not be read as a file"
}

# ---------------------------------------------------------------------------
# AC-13 (#176): a repo-level color.ui=always must not hide a finding (end to
# end through secret-scan.sh --range).
# ---------------------------------------------------------------------------
test_ac13_color_ui_always_still_finds() {
  repo="$workdir/ac13-color-ui-always"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'COLORUIabcdefghijklmnopqrstuvwx')"
    printf 'deploy token %s\n' "$token" > leaked.txt
    git add leaked.txt
    git commit -q -m 'add leaked token'
    git config color.ui always
  )
  run "$repo" --strict
  assert_exit "AC-13 color.ui=always: finding still detected" 1
  assert_stderr_contains "AC-13 color.ui=always: reports findings" "findings"
}

# ---------------------------------------------------------------------------
# AC-17 (#176): when git cannot read the range (here a diff.orderFile that
# does not exist), secret-scan.sh exits 3; this script must report a
# scanner failure with that exit code, never "clean".
# ---------------------------------------------------------------------------
test_ac17_scanner_git_failure_is_not_clean() {
  repo="$workdir/ac17-scanner-git-failure"
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
    git config diff.orderFile "$workdir/does-not-exist.order"
  )
  run "$repo" --strict
  assert_exit "AC-17 git log failure: strict mode exits with the scanner's 3" 3
  assert_stderr_contains "AC-17 git log failure: labeled a scanner failure" "scanner failed with exit 3"
  assert_stderr_not_contains "AC-17 git log failure: never reported clean" ": clean"
  run "$repo"
  assert_exit "AC-17 git log failure: default mode also exits 3" 3
  assert_stderr_not_contains "AC-17 git log failure: default mode never reported clean" ": clean"
}

# ---------------------------------------------------------------------------
# Attribute source. CI's pull_request checkout is the PR merge commit, so
# when the base changed .gitattributes after the branch forked, the scan
# reads the merge result's attributes (git merge-tree --write-tree), in
# both directions: a -diff the base added hides the token, and a -diff the
# base removed stops hiding it.
# ---------------------------------------------------------------------------
test_attr_base_adds_diff_reads_merge() {
  skip_without_merge_attr "base adds -diff cases" && return 0
  repo="$workdir/attr-base-adds"
  attr_base_adds_repo "$repo"
  run "$repo" --strict
  assert_exit "base adds -diff: strict scan reads the merge's attributes and is clean" 0
  assert_scanned_clean "base adds -diff: reports scanned+clean" "main"
  assert_stderr_contains "base adds -diff: notice names the merge" "attributes read from the merge of main and HEAD (main changed .gitattributes since "
  run "$repo"
  assert_exit "base adds -diff: default mode reads the merge's attributes too" 0
}

test_attr_base_removes_diff_reads_merge() {
  skip_without_merge_attr "base removes -diff cases" && return 0
  repo="$workdir/attr-base-removes"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf '*.txt -diff\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'mark txt files -diff'
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'ATTRREMOVEabcdefghijklmnopqrstu')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add leak.txt
    git commit -q -m 'add leaked token under -diff'
    git checkout -q main
    git rm -q .gitattributes
    git commit -q -m 'drop the -diff attribute'
    git checkout -q feature
  )
  run "$repo" --strict
  assert_exit "base removes -diff: strict scan reads the merge's attributes and finds the token" 1
  assert_stderr_contains "base removes -diff: reports findings" "findings"
  assert_stderr_contains "base removes -diff: notice names the merge" "attributes read from the merge of main and HEAD (main changed .gitattributes since "
}

# A .gitattributes below the root counts as a change too, from any cwd,
# even with diff.relative=true.
test_attr_nested_change_reads_merge() {
  skip_without_merge_attr "nested .gitattributes change cases" && return 0
  repo="$workdir/attr-nested"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    mkdir sub docs
    printf 'placeholder\n' > sub/placeholder.md
    printf 'placeholder\n' > docs/placeholder.md
    git add sub docs
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'ATTRNESTEDabcdefghijklmnopqrstu')"
    printf 'deploy token %s\n' "$token" > sub/leak.txt
    git add sub/leak.txt
    git commit -q -m 'add leaked token below sub'
    git checkout -q main
    printf '*.txt -diff\n' > sub/.gitattributes
    git add sub/.gitattributes
    git commit -q -m 'mark txt files below sub -diff'
    git checkout -q feature
  )
  run "$repo" --strict
  assert_exit "nested .gitattributes change: strict scan reads the merge's attributes" 0
  assert_stderr_contains "nested .gitattributes change: notice names the merge" "attributes read from the merge of main and HEAD"
  git -C "$repo" config diff.relative true
  run "$repo/docs" --strict
  assert_exit "nested .gitattributes change from another subdirectory with diff.relative=true" 0
  assert_stderr_contains "nested .gitattributes change from another subdirectory: notice names the merge" "attributes read from the merge of main and HEAD"
}

# When the base changed no .gitattributes, the merge keeps HEAD's, so the
# scan reads HEAD's without running merge-tree (a stub git that fails on
# merge-tree proves it is not run), prints no attribute notice, and never
# uses an inherited RALPH_SECRET_SCAN_ATTR_SOURCE.
test_attr_base_unchanged_reads_head() {
  repo="$workdir/attr-base-unchanged"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf '*.txt -diff\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'mark txt files -diff'
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'ATTRSAMEabcdefghijklmnopqrstuvw')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add leak.txt
    git commit -q -m 'add leaked token under -diff'
    git checkout -q main
    printf 'more\n' > other.md
    git add other.md
    git commit -q -m 'unrelated base change'
    git checkout -q feature
  )
  run_with_env "$repo" "PATH=$stub_merge_tree_fails:$PATH" --strict
  assert_exit "base unchanged .gitattributes: strict scan reads HEAD's attributes without merge-tree" 0
  assert_scanned_clean "base unchanged .gitattributes: reports scanned+clean" "main"
  assert_stderr_not_contains "base unchanged .gitattributes: no attribute notice" "attributes read from"
  assert_stderr_line_count "base unchanged .gitattributes: exactly one status line" "secret-scan-branch:" 1
  empty_tree="$(git -C "$repo" mktree </dev/null)"
  run_with_env "$repo" "RALPH_SECRET_SCAN_ATTR_SOURCE=$empty_tree" --strict
  assert_exit "base unchanged .gitattributes: an inherited RALPH_SECRET_SCAN_ATTR_SOURCE is not used" 0

  # The base's .gitattributes change is already merged into HEAD: the
  # merge-base is the base tip, so nothing changed since it.
  repo="$workdir/attr-base-merged"
  attr_base_adds_repo "$repo"
  git -C "$repo" merge -q --no-edit main
  run_with_env "$repo" "PATH=$stub_merge_tree_fails:$PATH" --strict
  assert_exit "base .gitattributes change already merged: strict scan reads HEAD's attributes without merge-tree" 0
  assert_stderr_not_contains "base .gitattributes change already merged: no attribute notice" "attributes read from"
}

# A conflicting merge reads HEAD's attributes in both modes, with a notice
# (CI does not run on a conflicting pull request). HEAD's attributes let
# the scan find the token; the conflicted merge tree would hide it.
test_attr_merge_conflict_reads_head() {
  skip_without_merge_attr "conflicting merge cases" && return 0
  repo="$workdir/attr-conflict"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf '# attributes\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'add .gitattributes'
    git checkout -q -b feature
    printf '*.md text\n' > .gitattributes
    token="$(printf 'ghp_%s' 'ATTRCONFLICTabcdefghijklmnopqrs')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add .gitattributes leak.txt
    git commit -q -m 'change .gitattributes and add leaked token'
    git checkout -q main
    printf '*.txt -diff\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'mark txt files -diff'
    git checkout -q feature
  )
  run "$repo" --strict
  assert_exit "conflicting merge: strict scan reads HEAD's attributes and finds the token" 1
  assert_stderr_contains "conflicting merge: notice says HEAD's attributes are read" "but the merge conflicts; attributes read from HEAD (CI does not run on a conflicting pull request)"
  run "$repo"
  assert_exit "conflicting merge: default mode reads HEAD's attributes too" 1
}

# A local merge driver would run inside merge-tree but not in CI's merge:
# --strict exits 3 naming the key; default mode reads HEAD's attributes.
test_attr_local_merge_driver() {
  for key in merge.default merge.keep.driver; do
    repo="$workdir/attr-driver-$key"
    attr_base_adds_repo "$repo"
    git -C "$repo" config "$key" union
    run "$repo" --strict
    assert_exit "local $key: strict mode exits 3" 3
    assert_stderr_contains "local $key: strict reason names the key" "cannot scan, main changed .gitattributes since "
    assert_stderr_contains "local $key: strict reason names the config" "local merge driver config ($key)"
    assert_stderr_not_contains "local $key: strict never reports clean" ": clean"
    assert_stderr_contains "local $key: strict reason names the remedy" "$attr_remedy"
    run "$repo"
    assert_exit "local $key: default mode reads HEAD's attributes and finds the token" 1
    assert_stderr_contains "local $key: default notice says HEAD's attributes are read" "($key) would run in git merge-tree but not in CI's merge; attributes read from HEAD; $attr_remedy"
  done
}

# attr_remedy -- the way past every refusal or fallback of the base-changed
# case: once main is merged or rebased into the branch, HEAD's attributes
# are the merge's.
attr_remedy="merge or rebase main into this branch so HEAD carries its .gitattributes, then re-run"

# expect_attr_unguaranteed <description> <PATH prefix> <reason> -- with the
# base-adds fixture and a stub first in PATH, --strict exits 3 with
# <reason> and the remedy, and default mode reads HEAD's attributes, says
# so with the remedy, and finds the token.
expect_attr_unguaranteed() {
  repo="$workdir/attr-unguaranteed-$(printf '%s' "$1" | tr -c 'a-z0-9' '-')"
  attr_base_adds_repo "$repo"
  run_with_env "$repo" "PATH=$2:$PATH" --strict
  assert_exit "$1: strict mode exits 3" 3
  assert_stderr_contains "$1: strict reason" "$3; $attr_remedy"
  assert_stderr_not_contains "$1: strict never reports clean" ": clean"
  run_with_env "$repo" "PATH=$2:$PATH"
  assert_exit "$1: default mode reads HEAD's attributes and finds the token" 1
  assert_stderr_contains "$1: default notice says HEAD's attributes are read" "$3; attributes read from HEAD; $attr_remedy"
  assert_stderr_contains "$1: default mode still prints the scan's status line" "against main: findings"
}

test_attr_merge_unavailable() {
  # These reasons come before the version check, so any git reaches them.
  expect_attr_unguaranteed "git older than 2.41" "$stub_old_git" \
    "git version 2.40.1 is not 2.41 or later, which GIT_ATTR_SOURCE needs"
  expect_attr_unguaranteed "a version string that does not parse" "$stub_malformed_version" \
    "git version 2.x is not 2.41 or later, which GIT_ATTR_SOURCE needs"
  expect_attr_unguaranteed "an empty version string" "$stub_empty_version" \
    "an unreadable git version is not 2.41 or later, which GIT_ATTR_SOURCE needs"
  expect_attr_unguaranteed "listing the merge driver config fails" "$stub_driver_list_fails" \
    "listing the local merge driver config failed (git config exited with 5)"
  expect_attr_unguaranteed "comparing .gitattributes with the base fails" "$stub_attr_diff_fails" \
    "(git diff exited with 5)"

  # These need a real git 2.41 or later to get past the version check.
  skip_without_merge_attr "merge-tree failure cases" && return 0
  expect_attr_unguaranteed "merge-tree exits 128" "$stub_merge_tree_fails" \
    "git merge-tree --write-tree exited with 128: fatal: stub merge-tree failure"
  expect_attr_unguaranteed "merge-tree without --write-tree" "$stub_no_write_tree" \
    "this git's merge-tree has no --write-tree"
  expect_attr_unguaranteed "no temporary file for merge-tree's errors" "$stub_merge_mktemp_fails" \
    "a temporary file for git merge-tree's errors could not be created"

  # The real merge-tree fails when it cannot write the merged objects.
  if [ "$(id -u)" -eq 0 ]; then
    printf '  SKIP: read-only object database (root can write anyway)\n'
    return 0
  fi
  repo="$workdir/attr-read-only-objects"
  attr_base_adds_repo "$repo"
  chmod -R a-w "$repo/.git/objects"
  run "$repo" --strict
  chmod -R u+w "$repo/.git/objects"
  assert_exit "read-only object database: strict mode exits 3" 3
  assert_stderr_contains "read-only object database: strict reason names merge-tree's exit" "git merge-tree --write-tree exited with 128"
}

# On a real git older than 2.41, the documented behavior holds: --strict
# exits 3 naming the version, and default mode reads HEAD's attributes.
# A git 2.41 or later reaches the same path only through the stub above.
test_attr_real_old_git() {
  if [ "$merge_attr_supported" -eq 1 ]; then
    printf '  SKIP: the real git older than 2.41 case (git %s is 2.41 or later; the stubbed old git covers it)\n' "$git_version"
    return 0
  fi
  expect_attr_unguaranteed "real git $git_version" "$no_stub" \
    "is not 2.41 or later, which GIT_ATTR_SOURCE needs"
}

# attr_conflict_repo <dir> -- main and feature (HEAD) both edit
# .gitattributes after the fork, so git's default merge conflicts there;
# feature adds a token in leak.txt, and main's side marks *.txt -diff.
# HEAD's attributes let the scan find the token; a merge that keeps both
# sides (a union merge) hides it.
attr_conflict_repo() {
  git_repo "$1"
  (
    cd "$1"
    git checkout -q -B main
    printf '# attributes\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'add .gitattributes'
    git checkout -q -b feature
    printf '*.md text\n' > .gitattributes
    token="$(printf 'ghp_%s' 'ATTRISOLATEabcdefghijklmnopqrst')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add .gitattributes leak.txt
    git commit -q -m 'change .gitattributes and add leaked token'
    git checkout -q main
    printf '*.txt -diff\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'mark txt files -diff'
    git checkout -q feature
  )
}

# git merge-tree runs isolated from local settings that change how a
# .gitattributes merges; CI's merge has none of them. Each case must give
# the default merge's result.
test_attr_merge_isolated_from_local_config() {
  skip_without_merge_attr "merge-tree isolation cases" && return 0

  # merge.renormalize with a clean filter for .gitattributes, chosen by a
  # user-level attributes file: the filter rewrites every side to
  # "*.txt -diff". Both sides edit different lines, so the default merge
  # is clean and keeps no -diff.
  repo="$workdir/attr-isolate-renormalize"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf '# top\n# middle\n# bottom\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'add .gitattributes'
    git checkout -q -b feature
    printf '# top\n# middle\n*.md text\n' > .gitattributes
    token="$(printf 'ghp_%s' 'ATTRRENORMabcdefghijklmnopqrstu')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add .gitattributes leak.txt
    git commit -q -m 'change the last line and add leaked token'
    git checkout -q main
    printf '*.cfg text\n# middle\n# bottom\n' > .gitattributes
    git add .gitattributes
    git commit -q -m 'change the first line'
    git checkout -q feature
  )
  run "$repo" --strict
  assert_exit "renormalize fixture under the default config: the merge keeps no -diff" 1
  printf '#!/bin/sh\ncat >/dev/null\nprintf %s\n' "'*.txt -diff\\n'" > "$workdir/hide-attributes-filter"
  chmod +x "$workdir/hide-attributes-filter"
  printf '.gitattributes filter=hide\n' > "$workdir/user-attributes-filter"
  git -C "$repo" config core.attributesFile "$workdir/user-attributes-filter"
  # git runs the filter through the shell, and $workdir holds spaces.
  git -C "$repo" config filter.hide.clean "'$workdir/hide-attributes-filter'"
  git -C "$repo" config merge.renormalize true
  run "$repo" --strict
  assert_exit "merge.renormalize with a local clean filter: the merge matches the default and the token is found" 1
  assert_stderr_contains "merge.renormalize with a local clean filter: notice names the merge" "attributes read from the merge of main and HEAD"

  # A user-level attributes file giving .gitattributes merge=union.
  repo="$workdir/attr-isolate-user-union"
  attr_conflict_repo "$repo"
  printf '.gitattributes merge=union\n' > "$workdir/user-attributes-union"
  git -C "$repo" config core.attributesFile "$workdir/user-attributes-union"
  run "$repo" --strict
  assert_exit "user attributes file with merge=union: the merge conflicts as by default and HEAD's attributes find the token" 1
  assert_stderr_contains "user attributes file with merge=union: conflict notice as by default" "but the merge conflicts; attributes read from HEAD"

  # An uncommitted working-tree .gitattributes giving itself merge=union.
  repo="$workdir/attr-isolate-worktree-union"
  attr_conflict_repo "$repo"
  printf '*.md text\n.gitattributes merge=union\n' > "$repo/.gitattributes"
  run "$repo" --strict
  assert_exit "working-tree .gitattributes with merge=union: the merge conflicts as by default" 1
  assert_stderr_contains "working-tree .gitattributes with merge=union: conflict notice as by default" "but the merge conflicts; attributes read from HEAD"

  # An inherited GIT_ATTR_SOURCE naming a tree whose .gitattributes gives
  # itself merge=union, or marks *.txt -diff.
  repo="$workdir/attr-isolate-inherited-union"
  attr_conflict_repo "$repo"
  union_blob="$(printf '.gitattributes merge=union\n' | git -C "$repo" hash-object -w --stdin)"
  union_tree="$(printf '100644 blob %s\t.gitattributes\n' "$union_blob" | git -C "$repo" mktree)"
  run_with_env "$repo" "GIT_ATTR_SOURCE=$union_tree" --strict
  assert_exit "inherited GIT_ATTR_SOURCE with merge=union: the merge conflicts as by default" 1
  assert_stderr_contains "inherited GIT_ATTR_SOURCE with merge=union: conflict notice as by default" "but the merge conflicts; attributes read from HEAD"
  hiding_blob="$(printf '*.txt -diff\n' | git -C "$repo" hash-object -w --stdin)"
  hiding_tree="$(printf '100644 blob %s\t.gitattributes\n' "$hiding_blob" | git -C "$repo" mktree)"
  run_with_env "$repo" "GIT_ATTR_SOURCE=$hiding_tree" --strict
  assert_exit "inherited GIT_ATTR_SOURCE marking *.txt -diff: no effect, the token is found" 1
}

# CI's clone has no .git/info/attributes, and git reads a local one under
# every setting, so while it holds a rule line this script does not scan:
# "cannot scan", exit 3 under --strict and exit 0 without a scan in default
# mode. Blank lines and "#" comments are not rules. A path that exists but
# is not a readable regular file cannot be checked and stops the scan too;
# a symlink to a missing file is nothing to read, as git treats it.
test_info_attributes_refused() {
  repo="$workdir/info-attributes"
  git_repo "$repo"
  (
    cd "$repo"
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
    git checkout -q -b feature
    token="$(printf 'ghp_%s' 'INFOATTRabcdefghijklmnopqrstuvw')"
    printf 'deploy token %s\n' "$token" > leak.txt
    git add leak.txt
    git commit -q -m 'add leaked token'
  )
  info_attributes="$(git -C "$repo" rev-parse --absolute-git-dir)/info/attributes"
  mkdir -p "$(dirname "$info_attributes")"

  printf '*.txt -diff\n' > "$info_attributes"
  run "$repo" --strict
  assert_exit ".git/info/attributes rule: strict mode exits 3" 3
  assert_stderr_contains ".git/info/attributes rule: the reason names the file and the fix" "cannot scan, .git/info/attributes holds attribute rules that CI does not read; move them to .gitattributes or remove the file"
  assert_stderr_not_contains ".git/info/attributes rule: strict never reports clean" ": clean"
  assert_stderr_not_contains ".git/info/attributes rule: strict does not scan" "scanned "
  run "$repo"
  assert_exit ".git/info/attributes rule: default mode exits 0" 0
  assert_stderr_contains ".git/info/attributes rule: default mode gives the same reason" "cannot scan, .git/info/attributes holds attribute rules that CI does not read"
  assert_stderr_not_contains ".git/info/attributes rule: default mode does not scan" "scanned "

  printf '*.md linguist-documentation\n' > "$info_attributes"
  run "$repo" --strict
  assert_exit ".git/info/attributes rule unrelated to diffs: strict mode still exits 3" 3

  printf '# a comment\n\n   # an indented comment\n\t\n' > "$info_attributes"
  run "$repo" --strict
  assert_exit ".git/info/attributes with only comments and blank lines: strict scans and finds the token" 1
  assert_stderr_contains ".git/info/attributes with only comments and blank lines: reports findings" "against main: findings"
  : > "$info_attributes"
  run "$repo" --strict
  assert_exit "empty .git/info/attributes: strict scans and finds the token" 1

  rm -f "$info_attributes"
  printf '*.txt -diff\n' > "$workdir/info-attributes-rules"
  ln -s "$workdir/info-attributes-rules" "$info_attributes"
  run "$repo" --strict
  assert_exit ".git/info/attributes symlinked to a rules file: strict mode exits 3" 3
  assert_stderr_contains ".git/info/attributes symlinked to a rules file: the reason names the file" "holds attribute rules that CI does not read"
  run "$repo"
  assert_exit ".git/info/attributes symlinked to a rules file: default mode exits 0" 0
  rm -f "$info_attributes"
  ln -s "$workdir/does-not-exist.attributes" "$info_attributes"
  run "$repo" --strict
  assert_exit ".git/info/attributes symlinked to a missing file: strict scans and finds the token" 1

  rm -f "$info_attributes"
  mkdir "$info_attributes"
  run "$repo" --strict
  assert_exit "a directory at .git/info/attributes: strict mode exits 3" 3
  assert_stderr_contains "a directory at .git/info/attributes: the reason names it" "cannot scan, .git/info/attributes exists but cannot be read as a file"
  run "$repo"
  assert_exit "a directory at .git/info/attributes: default mode exits 0" 0
  rmdir "$info_attributes"

  printf '# only a comment\n' > "$info_attributes"
  chmod 000 "$info_attributes"
  if [ "$(id -u)" -eq 0 ]; then
    printf '  SKIP: an unreadable .git/info/attributes (root can read it anyway)\n'
  else
    run "$repo" --strict
    assert_exit "an unreadable .git/info/attributes: strict mode exits 3" 3
    assert_stderr_contains "an unreadable .git/info/attributes: the reason names it" "cannot scan, .git/info/attributes exists but cannot be read as a file"
    run "$repo"
    assert_exit "an unreadable .git/info/attributes: default mode exits 0" 0
  fi
  chmod 644 "$info_attributes"

  # grep failing on the file (exit 2) cannot tell whether it holds a rule.
  run_with_env "$repo" "PATH=$stub_grep_fails:$PATH" --strict
  assert_exit "reading .git/info/attributes fails: strict mode exits 3" 3
  assert_stderr_contains "reading .git/info/attributes fails: the reason names grep's exit" "cannot scan, reading .git/info/attributes failed (grep exited with 2)"
  run_with_env "$repo" "PATH=$stub_grep_fails:$PATH"
  assert_exit "reading .git/info/attributes fails: default mode exits 0" 0
  rm -f "$info_attributes"
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
test_ac3b_tag_shadows_base_branch
test_ac3b_local_branch_shadows_remote_tracking_ref
test_cannot_scan_no_base_ref
test_cannot_scan_unrelated_histories
test_cannot_scan_outside_repo
test_cannot_scan_scanner_missing
test_scanner_failure_is_not_findings
test_allowlist_only_committed_at_head
test_allowlist_self_match
test_ac2d_symlinked_allowlist_resolves_target
test_ac2d_symlinked_allowlist_denies_fixture
test_ac2d_symlinked_allowlist_dangling
test_ac2d_symlinked_allowlist_target_name_not_used_as_regex
test_ls_tree_mode_check_is_root_relative
test_ac2d_symlinked_allowlist_directory_target_rejected
test_ac2d_symlinked_allowlist_dot_target_rejected
test_ac2d_symlinked_allowlist_bare_directory_target_rejected
test_ac2d_symlinked_allowlist_absolute_target_rejected
test_ac2d_symlinked_allowlist_dotdot_target_rejected
test_ac2d_symlinked_allowlist_leading_dotslash_target_rejected
test_ac11_symlinked_allowlist_target_trailing_newline
test_ac12_symlinked_allowlist_non_ascii_target
test_ac13_color_ui_always_still_finds
test_ac17_scanner_git_failure_is_not_clean
test_attr_base_adds_diff_reads_merge
test_attr_base_removes_diff_reads_merge
test_attr_nested_change_reads_merge
test_attr_base_unchanged_reads_head
test_attr_merge_conflict_reads_head
test_attr_local_merge_driver
test_attr_merge_unavailable
test_attr_real_old_git
test_attr_merge_isolated_from_local_config
test_info_attributes_refused
test_usage_and_space_path

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
