#!/usr/bin/env sh
# Integration tests for what scripts/run-verify.sh does when no verifier runs
# (plan 2026-11-19-verify-docs-only-base, issue #341): it takes docs_only
# from scripts/detect-changed-languages.sh in both scopes, so a code change
# exits 2 in CI as it does locally, a docs-only or ralph-scaffold-only change
# exits 0, and a change set that could not be read exits 2 and says why.
#
# The fixture is a repository with no language pack and no local verifier:
# README.md, index.html, a data .json, copies of this repository's
# run-verify.sh, run-static-verify.sh, run-test.sh, and
# detect-changed-languages.sh, a detect-languages.sh stub that prints
# nothing, and a .ralph/manifest.toml listing those files. No
# ralph-config.sh, so no ralph is asked for a setting. A bare remote holds it
# with the default branch develop (no main, no master). Each case checks it
# out one of two ways:
#   CI-like      what actions/checkout does for a pull request with
#                fetch-depth: 0: every branch fetched into
#                refs/remotes/origin/*, no refs/remotes/origin/HEAD, no local
#                branch, and HEAD detached at a merge commit of the feature
#                branch into develop; GITHUB_BASE_REF=develop
#   local clone  git clone, the feature branch checked out, a clean tree, no
#                GITHUB_BASE_REF
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"

# Cases pass GITHUB_BASE_REF, the scope, and the mode themselves; nothing
# comes from the caller (CI sets GITHUB_BASE_REF, and run-test.sh exports
# RALPH_VERIFY_SCOPE and HARNESS_VERIFY_MODE).
unset GITHUB_BASE_REF RALPH_VERIFY_SCOPE HARNESS_VERIFY_MODE RALPH_VERIFY_PROJECT_ROOTS

_pass=0
_fail=0
_total=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/run-verify-docs-only.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT HUP INT TERM

# Pin git away from the developer's global and system configuration. The
# hermetic global file gives the commits and merges an author.
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_CONFIG_NOSYSTEM=1
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT
git config --global user.name "Ralph Test"
git config --global user.email "ralph-test@example.com"
git config --global advice.detachedHead false
git config --global init.defaultBranch develop

# The TMPDIR of every run below, so no run writes outside $workdir.
run_tmp="$workdir/tmp"
mkdir -p "$run_tmp"

# The real git, for the stand-in of the AC11 cases.
real_git="$(command -v git)"

# --- helpers ---------------------------------------------------------------

# run_in <dir> <out> <script> [VAR=value ...] -- runs ./scripts/<script> in
# <dir> with the given variables added to its environment; sets run_rc.
run_in() {
  _dir="$1"
  _out="$2"
  _script="$3"
  shift 3
  set +e
  (
    cd "$_dir" || exit 99
    exec env TMPDIR="$run_tmp" "$@" "./scripts/$_script"
  ) > "$_out" 2>&1
  run_rc=$?
  set -e
}

assert_rc() {
  _desc="$1"
  _want="$2"
  _out="$3"
  if [ "$run_rc" -eq "$_want" ]; then
    record_pass "$_desc (exit $run_rc)"
  else
    record_fail "$_desc (exit $run_rc, want $_want)"
    sed 's/^/    out: /' "$_out"
  fi
}

assert_contains() {
  _desc="$1"
  _needle="$2"
  _out="$3"
  if grep -Fq -- "$_needle" "$_out" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle)"
    sed 's/^/    out: /' "$_out"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _out="$3"
  if grep -Fq -- "$_needle" "$_out" 2>/dev/null; then
    record_fail "$_desc (unexpected: $_needle)"
    sed 's/^/    out: /' "$_out"
  else
    record_pass "$_desc"
  fi
}

# assert_scope_file <desc> <dir> <docs_only> -- <dir>'s
# .harness/state/verify-scope holds the full scope's four lines, with
# docs_only=<docs_only> and no <lang>_roots line.
assert_scope_file() {
  _scope="$2/.harness/state/verify-scope"
  _want="$(printf 'scope=full\nreason=requested_full\ndocs_only=%s\nlanguages=' "$3")"
  if [ -f "$_scope" ] && [ "$(cat "$_scope")" = "$_want" ]; then
    record_pass "$1"
  else
    record_fail "$1"
    if [ -f "$_scope" ]; then
      sed 's/^/    verify-scope: /' "$_scope"
    else
      printf '    verify-scope: missing\n'
    fi
  fi
}

# assert_same_rc <desc> <want> <got> -- two runs ended with the same code.
assert_same_rc() {
  if [ "$2" -eq "$3" ]; then
    record_pass "$1 (both exit $2)"
  else
    record_fail "$1 (CI-like exit $2, local clone exit $3)"
  fi
}

docs_line="No language verifier ran. The changes are docs or ralph scaffold files only."
base_line="The changes are read against GITHUB_BASE_REF in CI (fetch the base branch, as actions/checkout does with fetch-depth: 0), else [verify].base of ralph.local.toml, else the default branch."
hint_static="Add a verifier: an executable ./scripts/verify.local.sh, an executable .ralph/local/verify.d/<name>.sh, or a language pack (ralph pack add <language>)."
hint_test="Add a verifier: an executable ./scripts/verify.local.sh, an executable .ralph/local/test.d/<name>.sh, or a language pack (ralph pack add <language>)."
hint_all="Add a verifier: an executable ./scripts/verify.local.sh, an executable .ralph/local/verify.d/<name>.sh (static) or .ralph/local/test.d/<name>.sh (test), or a language pack (ralph pack add <language>)."

# add_scripts <dir> -- writes the fixture's verification scripts in <dir>.
add_scripts() {
  mkdir -p "$1/scripts"
  for _name in run-verify.sh run-static-verify.sh run-test.sh detect-changed-languages.sh; do
    cp "$PROJECT_ROOT/scripts/$_name" "$1/scripts/$_name"
  done
  printf '#!/usr/bin/env sh\nexit 0\n' > "$1/scripts/detect-languages.sh"
  chmod +x "$1/scripts/"*.sh
}

# add_gitignore <dir> -- what run-verify.sh writes stays out of the tree, as
# ralph init's .gitignore keeps it.
add_gitignore() {
  printf '.harness/state/\n.harness/logs/\ndocs/evidence/*.log\n' > "$1/.gitignore"
}

# write_manifest <dir> <path>... -- writes <dir>/.ralph/manifest.toml with a
# [files.'<path>'] table for each <path>, in the form ralph init writes.
write_manifest() {
  _mdir="$1"
  shift
  mkdir -p "$_mdir/.ralph"
  {
    printf "[meta]\nversion = 'dev'\nlayout = 'v2'\n\n[files]\n"
    for _path in "$@"; do
      printf "[files.'%s']\nhash = 'sha256:0'\nmanaged = true\nowner = 'core'\n\n" "$_path"
    done
  } > "$_mdir/.ralph/manifest.toml"
}

fixture_scripts="scripts/run-verify.sh scripts/run-static-verify.sh scripts/run-test.sh scripts/detect-changed-languages.sh scripts/detect-languages.sh"

# make_bare <name> <seed> -- a bare repository holding every branch of the
# seed repository <seed>, with develop as its default branch. Prints its
# path.
make_bare() {
  _bare="$workdir/$1.git"
  git init -q --bare "$_bare"
  git -C "$_bare" symbolic-ref HEAD refs/heads/develop
  git -C "$2" push -q "$_bare" '+refs/heads/*:refs/heads/*'
  printf '%s\n' "$_bare"
}

# ci_checkout <bare> <feature> <dir> -- the CI-like checkout (see the header)
# of the merge of <feature> into develop, in <dir>. git 2.48 and later
# create refs/remotes/origin/HEAD on fetch; it is removed, so the base can
# come only from GITHUB_BASE_REF.
ci_checkout() {
  git init -q "$3"
  (
    cd "$3"
    git remote add origin "$1"
    git fetch -q origin '+refs/heads/*:refs/remotes/origin/*'
    git symbolic-ref --delete refs/remotes/origin/HEAD 2>/dev/null || true
    git checkout -q --detach origin/develop
    git merge -q --no-ff --no-edit "origin/$2"
  )
}

# local_clone <bare> <feature> <dir> -- the local clone (see the header) on
# <feature>, in <dir>.
local_clone() {
  git clone -q "$1" "$3"
  git -C "$3" checkout -q "$2"
}

# --- the init-like remote --------------------------------------------------
# develop holds what ralph init would leave in a pack-less project, plus
# scripts/old-tool.sh, a non-.md scaffold file listed in the manifest. The
# feature branches: code adds app.js (AC1), docs edits README.md and adds
# docs/x.md (AC2), upgrade deletes scripts/old-tool.sh together with its
# manifest entry (AC12), and lonely is an orphan branch (AC5).
seed="$workdir/seed"
mkdir -p "$seed"
(
  cd "$seed"
  git init -q
  git checkout -q -B develop
  printf '# site\n' > README.md
  printf '<!doctype html>\n<title>site</title>\n' > index.html
  printf '{"items": [1, 2, 3]}\n' > data.json
  add_scripts .
  add_gitignore .
  printf '#!/usr/bin/env sh\nexit 0\n' > scripts/old-tool.sh
  chmod +x scripts/old-tool.sh
  # shellcheck disable=SC2086  # fixture_scripts is a list of plain paths
  write_manifest . $fixture_scripts .gitignore scripts/old-tool.sh
  git add .
  git commit -q -m "init"

  git checkout -q -b code
  printf 'console.log("hi");\n' > app.js
  git add app.js
  git commit -q -m "add app.js"

  git checkout -q develop
  git checkout -q -b docs
  printf 'More words.\n' >> README.md
  mkdir -p docs
  printf '# x\n' > docs/x.md
  git add README.md docs/x.md
  git commit -q -m "docs"

  git checkout -q develop
  git checkout -q -b upgrade
  git rm -q scripts/old-tool.sh
  # shellcheck disable=SC2086  # fixture_scripts is a list of plain paths
  write_manifest . $fixture_scripts .gitignore
  git add .ralph/manifest.toml
  git commit -q -m "upgrade removes old-tool.sh"

  git checkout -q --orphan lonely
  git rm -rfq .
  printf 'alone\n' > lonely.txt
  git add lonely.txt
  git commit -q -m "lonely"
  git checkout -q develop
)
bare="$(make_bare site "$seed")"

# --- the bare-base remote (AC3) --------------------------------------------
# The case of issue #341: develop holds README.md and index.html only, and
# the feature branch scaffold adds what ralph init writes (the scripts, a
# .gitignore, and .ralph/manifest.toml listing every added path but itself).
seed_init="$workdir/seed-init"
mkdir -p "$seed_init"
(
  cd "$seed_init"
  git init -q
  git checkout -q -B develop
  printf '# site\n' > README.md
  printf '<!doctype html>\n<title>site</title>\n' > index.html
  git add .
  git commit -q -m "site"

  git checkout -q -b scaffold
  add_scripts .
  add_gitignore .
  # shellcheck disable=SC2086  # fixture_scripts is a list of plain paths
  write_manifest . $fixture_scripts .gitignore
  git add .
  git commit -q -m "ralph init"
  git checkout -q develop
)
bare_init="$(make_bare site-init "$seed_init")"

# --- fixture checks --------------------------------------------------------
printf 'Fixture: the CI-like checkout and the local clone\n'
ci="$workdir/ci-code"
ci_checkout "$bare" code "$ci"
if ! git -C "$ci" symbolic-ref -q HEAD >/dev/null 2>&1 &&
  [ -z "$(git -C "$ci" for-each-ref refs/heads)" ] &&
  ! git -C "$ci" rev-parse -q --verify refs/remotes/origin/HEAD >/dev/null &&
  ! git -C "$ci" rev-parse -q --verify refs/remotes/origin/main >/dev/null &&
  ! git -C "$ci" rev-parse -q --verify refs/remotes/origin/master >/dev/null &&
  git -C "$ci" rev-parse -q --verify 'HEAD^2' >/dev/null; then
  record_pass "CI-like: HEAD is a detached merge commit, no local branch, no origin/HEAD, main, or master"
else
  record_fail "CI-like: HEAD is a detached merge commit, no local branch, no origin/HEAD, main, or master"
  git -C "$ci" for-each-ref | sed 's/^/    ref: /'
fi
clone="$workdir/clone-code"
local_clone "$bare" code "$clone"
if [ "$(git -C "$clone" symbolic-ref -q HEAD)" = "refs/heads/code" ] &&
  [ -z "$(git -C "$clone" status --porcelain --untracked-files=all)" ]; then
  record_pass "local clone: on the feature branch with a clean tree"
else
  record_fail "local clone: on the feature branch with a clean tree"
fi

# --- AC1: a code change --------------------------------------------------
printf 'AC1: a code change with no verifier exits 2 in the CI-like checkout\n'
out="$workdir/ac1-ci.out"
run_in "$ci" "$out" run-verify.sh GITHUB_BASE_REF=develop RALPH_VERIFY_SCOPE=full
ci_code_rc=$run_rc
assert_rc "AC1 CI-like code change exits 2" 2 "$out"
assert_contains "AC1 CI-like code change names the code-like path" \
  "No verifier ran for code-like changes (for example: app.js)." "$out"
assert_not_contains "AC1 CI-like code change does not say docs only" "docs or ralph scaffold" "$out"
assert_contains "AC1 CI-like code change says how to add a verifier (all mode)" "$hint_all" "$out"
assert_scope_file "AC1 CI-like full scope records docs_only=false" "$ci" false

# --- AC2: a docs change ----------------------------------------------------
printf 'AC2: a docs change with no verifier exits 0 in the CI-like checkout\n'
ci_docs="$workdir/ci-docs"
ci_checkout "$bare" docs "$ci_docs"
out="$workdir/ac2-ci.out"
run_in "$ci_docs" "$out" run-verify.sh GITHUB_BASE_REF=develop RALPH_VERIFY_SCOPE=full
ci_docs_rc=$run_rc
assert_rc "AC2 CI-like docs change exits 0" 0 "$out"
assert_contains "AC2 CI-like docs change says docs only" "$docs_line" "$out"
assert_not_contains "AC2 CI-like docs change gives no verifier hint" "Add a verifier" "$out"
assert_scope_file "AC2 CI-like full scope records docs_only=true" "$ci_docs" true

# --- AC3: a scaffold-only change -------------------------------------------
printf 'AC3: a change that adds only ralph scaffold exits 0 in the CI-like checkout\n'
ci_scaffold="$workdir/ci-scaffold"
ci_checkout "$bare_init" scaffold "$ci_scaffold"
out="$workdir/ac3-ci.out"
run_in "$ci_scaffold" "$out" run-verify.sh GITHUB_BASE_REF=develop RALPH_VERIFY_SCOPE=full
ci_scaffold_rc=$run_rc
assert_rc "AC3 CI-like scaffold-only change exits 0" 0 "$out"
assert_contains "AC3 CI-like scaffold-only change says docs or ralph scaffold" "$docs_line" "$out"

# --- AC12: an upgrade that deletes a scaffold file -------------------------
printf 'AC12: deleting a scaffold file with its manifest entry exits 0\n'
ci_upgrade="$workdir/ci-upgrade"
ci_checkout "$bare" upgrade "$ci_upgrade"
out="$workdir/ac12-ci.out"
run_in "$ci_upgrade" "$out" run-verify.sh GITHUB_BASE_REF=develop RALPH_VERIFY_SCOPE=full
assert_rc "AC12 CI-like deletion of scripts/old-tool.sh exits 0" 0 "$out"
assert_contains "AC12 CI-like deletion says docs or ralph scaffold" "$docs_line" "$out"
clone_upgrade="$workdir/clone-upgrade"
local_clone "$bare" upgrade "$clone_upgrade"
for scope in full changed; do
  out="$workdir/ac12-clone-$scope.out"
  run_in "$clone_upgrade" "$out" run-verify.sh RALPH_VERIFY_SCOPE="$scope"
  assert_rc "AC12 local clone ($scope scope) deletion of scripts/old-tool.sh exits 0" 0 "$out"
  assert_contains "AC12 local clone ($scope scope) says docs or ralph scaffold" "$docs_line" "$out"
done

# --- AC4: the local clone gives the CI-like exit codes ---------------------
printf 'AC4: a local clone gives the exit codes of the CI-like checkout, in both scopes\n'
clone_docs="$workdir/clone-docs"
local_clone "$bare" docs "$clone_docs"
clone_scaffold="$workdir/clone-scaffold"
local_clone "$bare_init" scaffold "$clone_scaffold"
for kind in code docs scaffold; do
  case "$kind" in
    code) dir="$clone" want=$ci_code_rc ;;
    docs) dir="$clone_docs" want=$ci_docs_rc ;;
    scaffold) dir="$clone_scaffold" want=$ci_scaffold_rc ;;
  esac
  for scope in full changed; do
    out="$workdir/ac4-$kind-$scope.out"
    run_in "$dir" "$out" run-verify.sh RALPH_VERIFY_SCOPE="$scope"
    assert_same_rc "AC4 $kind change: local clone ($scope scope) matches the CI-like checkout" "$want" "$run_rc"
  done
done
out="$workdir/ac4-code-changed.out"
assert_contains "AC4 local clone (changed scope) names the code-like path" \
  "No verifier ran for code-like changes (for example: app.js)." "$out"

# --- AC5: a change set that could not be read ------------------------------
printf 'AC5: an unreadable change set exits 2 and says why, never docs only\n'
out="$workdir/ac5-no-ref.out"
run_in "$ci_docs" "$out" run-verify.sh GITHUB_BASE_REF=nosuch RALPH_VERIFY_SCOPE=full
assert_rc "AC5 GITHUB_BASE_REF with no ref exits 2 on a docs change" 2 "$out"
assert_contains "AC5 GITHUB_BASE_REF with no ref says the changes could not be read, and why" \
  "No verifier ran, and the changes could not be read (no_base_ref:nosuch), so they are not taken as docs only." "$out"
assert_contains "AC5 GITHUB_BASE_REF with no ref says where the changes are read from" "$base_line" "$out"
assert_contains "AC5 GITHUB_BASE_REF with no ref says how to add a verifier" "$hint_all" "$out"
assert_not_contains "AC5 GITHUB_BASE_REF with no ref does not say docs only" "docs or ralph scaffold" "$out"
assert_scope_file "AC5 full scope records docs_only=unknown" "$ci_docs" unknown

out="$workdir/ac5-no-merge-base.out"
run_in "$ci_docs" "$out" run-verify.sh GITHUB_BASE_REF=lonely RALPH_VERIFY_SCOPE=full
assert_rc "AC5 GITHUB_BASE_REF with no merge-base exits 2 on a docs change" 2 "$out"
assert_contains "AC5 GITHUB_BASE_REF with no merge-base says the changes could not be read, and why" \
  "the changes could not be read (no_merge_base:refs/remotes/origin/lonely)" "$out"
assert_not_contains "AC5 GITHUB_BASE_REF with no merge-base does not say docs only" "docs or ralph scaffold" "$out"

# Why CI needs GITHUB_BASE_REF: the CI-like checkout has no origin/HEAD, no
# main or master, and no branch, so without it there is no base at all.
out="$workdir/ac5-no-base.out"
run_in "$ci_docs" "$out" run-verify.sh RALPH_VERIFY_SCOPE=full
assert_rc "AC5 CI-like checkout without GITHUB_BASE_REF exits 2 on a docs change" 2 "$out"
assert_contains "AC5 CI-like checkout without GITHUB_BASE_REF has no diff base" \
  "the changes could not be read (no_diff_base)" "$out"
assert_not_contains "AC5 CI-like checkout without GITHUB_BASE_REF does not say docs only" "docs or ralph scaffold" "$out"

# --- AC6: the verifier hint of each mode -----------------------------------
printf 'AC6: the exit-2 output says how to add a verifier for the mode\n'
out="$workdir/ac6-static-wrapper.out"
run_in "$clone" "$out" run-static-verify.sh
assert_rc "AC6 run-static-verify.sh on a code change exits 2" 2 "$out"
assert_contains "AC6 static mode (wrapper) names verify.local.sh, verify.d, and ralph pack add" "$hint_static" "$out"
assert_not_contains "AC6 static mode (wrapper) does not name test.d" "test.d" "$out"

out="$workdir/ac6-static-env.out"
run_in "$clone" "$out" run-verify.sh HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full
assert_rc "AC6 HARNESS_VERIFY_MODE=static on a code change exits 2" 2 "$out"
assert_contains "AC6 static mode (variable) names verify.local.sh, verify.d, and ralph pack add" "$hint_static" "$out"

out="$workdir/ac6-test-wrapper.out"
run_in "$clone" "$out" run-test.sh
assert_rc "AC6 run-test.sh on a code change exits 2" 2 "$out"
assert_contains "AC6 test mode names verify.local.sh, test.d, and ralph pack add" "$hint_test" "$out"
assert_not_contains "AC6 test mode does not name verify.d" "verify.d" "$out"

out="$workdir/ac6-all.out"
run_in "$clone" "$out" run-verify.sh HARNESS_VERIFY_MODE=all RALPH_VERIFY_SCOPE=full
assert_rc "AC6 HARNESS_VERIFY_MODE=all on a code change exits 2" 2 "$out"
assert_contains "AC6 all mode names verify.local.sh, verify.d and test.d, and ralph pack add" "$hint_all" "$out"

# --- AC11: a failed listing command ----------------------------------------
# A git first on PATH runs the real git, except that a call whose arguments
# match $FAKE_GIT_FAIL exits 128 after the real git printed its output. On
# the docs change, which passes otherwise (AC4), a failed
# `git diff --name-only --cached` must not leave a docs-only partial list.
printf 'AC11: a failed listing command exits 2 with diff_failed, in both scopes\n'
fake_git_bin="$workdir/fake-git-bin"
mkdir -p "$fake_git_bin"
{
  printf '#!/bin/sh\n'
  printf 'real_git="%s"\n' "$real_git"
  cat <<'EOF'
case "$*" in
  $FAKE_GIT_FAIL)
    "$real_git" "$@"
    exit 128
    ;;
esac
exec "$real_git" "$@"
EOF
} > "$fake_git_bin/git"
chmod +x "$fake_git_bin/git"
for scope in full changed; do
  out="$workdir/ac11-$scope.out"
  run_in "$clone_docs" "$out" run-verify.sh PATH="$fake_git_bin:$PATH" \
    FAKE_GIT_FAIL='diff --name-only --cached' RALPH_VERIFY_SCOPE="$scope"
  assert_rc "AC11 failed staged listing ($scope scope) exits 2 on a docs change" 2 "$out"
  assert_contains "AC11 failed staged listing ($scope scope) says diff_failed" \
    "the changes could not be read (diff_failed)" "$out"
  assert_not_contains "AC11 failed staged listing ($scope scope) does not say docs only" "docs or ralph scaffold" "$out"
done

# --- the detector missing, failing, or printing an odd docs_only -----------
printf 'Detector: missing, failing, or an odd docs_only gives unknown and exit 2\n'
clone_broken="$workdir/clone-broken"
local_clone "$bare" docs "$clone_broken"
rm -f "$clone_broken/scripts/detect-changed-languages.sh"
for scope in full changed; do
  out="$workdir/detector-missing-$scope.out"
  run_in "$clone_broken" "$out" run-verify.sh RALPH_VERIFY_SCOPE="$scope"
  assert_rc "missing detector ($scope scope) exits 2 on a docs change" 2 "$out"
  assert_contains "missing detector ($scope scope) says changed_detector_missing" \
    "the changes could not be read (changed_detector_missing)" "$out"
  if [ "$scope" = full ]; then
    assert_scope_file "missing detector: full scope records docs_only=unknown" "$clone_broken" unknown
  fi
done

printf '#!/usr/bin/env sh\nexit 1\n' > "$clone_broken/scripts/detect-changed-languages.sh"
chmod +x "$clone_broken/scripts/detect-changed-languages.sh"
for scope in full changed; do
  out="$workdir/detector-failed-$scope.out"
  run_in "$clone_broken" "$out" run-verify.sh RALPH_VERIFY_SCOPE="$scope"
  assert_rc "failing detector ($scope scope) exits 2 on a docs change" 2 "$out"
  assert_contains "failing detector ($scope scope) says changed_detector_failed" \
    "the changes could not be read (changed_detector_failed)" "$out"
done

printf '#!/usr/bin/env sh\nprintf "scope=full\\nreason=odd\\ndocs_only=yes\\nlanguages=\\n"\n' \
  > "$clone_broken/scripts/detect-changed-languages.sh"
out="$workdir/detector-odd.out"
run_in "$clone_broken" "$out" run-verify.sh RALPH_VERIFY_SCOPE=full
assert_rc "docs_only=yes from the detector exits 2" 2 "$out"
assert_contains "docs_only=yes from the detector is taken as unknown" "the changes could not be read (odd)" "$out"
assert_scope_file "docs_only=yes: full scope records docs_only=unknown" "$clone_broken" unknown

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
