#!/usr/bin/env sh
# Regression tests for scripts/run-verify.sh's per-run handoff files. The
# block's exit code, the language scope, and the docs-only marker go through
# temp files private to one run, so:
#   - a run whose block dies before writing its exit code fails, not passes;
#   - two runs in the same worktree cannot overwrite or delete each other's
#     exit code (an old-style run that rewrites and removes the shared
#     .harness/state/verify-exit-code does not change the result either);
#   - .harness/state/verify-scope is still left behind at the end of a run;
#   - the per-run temp files are gone after success, failure, a killed block,
#     and a TERM signal;
#   - plain passing and failing runs keep their exit codes 0 and 1, and the
#     docs-only detection still tells a code change (exit 2) from a docs one.
#
# The fixture's golang verifier is a stub driven by env vars:
#   STUB_EXIT        its exit code (default 0)
#   STUB_STARTED     a file it creates when it starts
#   STUB_WAIT_FOR    a file it waits for (up to about 10 s) before exiting
#   STUB_KILL_BLOCK  1: SIGKILL its parent, run-verify.sh's { } | tee block
# and the fixture's detect-languages.sh prints STUB_LANGUAGES (default golang).
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"

# Cases pick their scope, mode, and stub settings explicitly; do not inherit
# them from the caller (run-test.sh exports RALPH_VERIFY_SCOPE=changed, and CI
# sets GITHUB_BASE_REF).
unset RALPH_VERIFY_BASE RALPH_VERIFY_PROJECT_ROOTS RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN
unset GITHUB_BASE_REF RALPH_XREVIEW_BASE
unset STUB_EXIT STUB_STARTED STUB_WAIT_FOR STUB_KILL_BLOCK STUB_LANGUAGES
RALPH_VERIFY_SCOPE=full
HARNESS_VERIFY_MODE=all
export RALPH_VERIFY_SCOPE HARNESS_VERIFY_MODE

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

workdir="$(mktemp -d "${TMPDIR:-/tmp}/run-verify-concurrency.XXXXXX")"
bg_pids=""
cleanup() {
  for _pid in $bg_pids; do
    kill "$_pid" 2>/dev/null || true
  done
  rm -rf "$workdir"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

# Pin git away from the developer's global and system configuration.
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_CONFIG_NOSYSTEM=1
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT

# The TMPDIR every run-verify.sh run below gets, so its per-run files can be
# seen and checked.
run_tmp="$workdir/tmp"
mkdir -p "$run_tmp"

# make_repo <name> -- a committed git repo with the real scripts/run-verify.sh,
# a stub detect-languages.sh, a stub golang verifier, and no
# secret-scan-branch.sh (run-verify.sh skips that scan when it is missing).
# Prints the repo path.
make_repo() {
  _repo="$workdir/$1"
  mkdir -p "$_repo/scripts" "$_repo/packs/languages/golang"
  cp "$PROJECT_ROOT/scripts/run-verify.sh" "$_repo/scripts/run-verify.sh"
  chmod +x "$_repo/scripts/run-verify.sh"
  cat > "$_repo/scripts/detect-languages.sh" <<'SH'
#!/usr/bin/env sh
printf '%s\n' "${STUB_LANGUAGES-golang}"
SH
  chmod +x "$_repo/scripts/detect-languages.sh"
  cat > "$_repo/packs/languages/golang/verify.sh" <<'SH'
#!/usr/bin/env sh
if [ -n "${STUB_STARTED:-}" ]; then
  : > "$STUB_STARTED"
fi
if [ "${STUB_KILL_BLOCK:-0}" = "1" ]; then
  kill -9 "$PPID"
  exit 0
fi
if [ -n "${STUB_WAIT_FOR:-}" ]; then
  _n=0
  while [ ! -f "$STUB_WAIT_FOR" ] && [ "$_n" -lt 200 ]; do
    sleep 0.05
    _n=$((_n + 1))
  done
fi
exit "${STUB_EXIT:-0}"
SH
  chmod +x "$_repo/packs/languages/golang/verify.sh"
  (
    cd "$_repo"
    git init -q
    git checkout -q -B main
    git config user.name "Ralph Test"
    git config user.email "ralph-test@example.com"
    printf '# test repo\n' > README.md
    printf 'app\n' > app.txt
    git add .
    git commit -q -m init
  )
  printf '%s\n' "$_repo"
}

# run_verify <repo> <out> [VAR=value ...] -- runs <repo>'s run-verify.sh in
# the foreground with the test's TMPDIR and the given settings; sets run_rc.
run_verify() {
  _repo="$1"
  _out="$2"
  shift 2
  set +e
  (
    cd "$_repo"
    exec env TMPDIR="$run_tmp" "$@" ./scripts/run-verify.sh
  ) > "$_out" 2>&1
  run_rc=$?
  set -e
}

# start_run_verify <repo> <out> [VAR=value ...] -- the same in the
# background; sets bg_pid to run-verify.sh's own pid (the subshell execs it).
start_run_verify() {
  _repo="$1"
  _out="$2"
  shift 2
  (
    cd "$_repo"
    exec env TMPDIR="$run_tmp" "$@" ./scripts/run-verify.sh
  ) > "$_out" 2>&1 &
  bg_pid=$!
  bg_pids="$bg_pids $bg_pid"
}

# wait_bg <pid> -- waits for a background run; sets run_rc.
wait_bg() {
  set +e
  wait "$1"
  run_rc=$?
  set -e
}

# wait_for_file <path> -- waits up to about 10 s for <path> to appear.
wait_for_file() {
  _n=0
  while [ ! -f "$1" ]; do
    _n=$((_n + 1))
    if [ "$_n" -gt 200 ]; then
      return 1
    fi
    sleep 0.05
  done
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

assert_rc_nonzero() {
  _desc="$1"
  _out="$2"
  if [ "$run_rc" -ne 0 ]; then
    record_pass "$_desc (exit $run_rc)"
  else
    record_fail "$_desc (exit 0, want non-zero)"
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

# assert_no_run_files <desc> <repo> -- no run-verify.sh temp file is left in
# the test's TMPDIR, and no half-published scope file in <repo>'s
# .harness/state/.
assert_no_run_files() {
  _desc="$1"
  _left="$(find "$run_tmp" -mindepth 1 -name 'ralph-*' 2>/dev/null)"
  _left="$_left$(find "$2/.harness/state" -mindepth 1 -name 'verify-scope.*' 2>/dev/null)"
  if [ -z "$_left" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (left behind: $_left)"
  fi
}

# assert_absent <desc> <path>
assert_absent() {
  if [ -e "$2" ]; then
    record_fail "$1 ($2 exists)"
  else
    record_pass "$1"
  fi
}

expected_scope="$workdir/expected-scope"
cat > "$expected_scope" <<'EOF'
scope=full
reason=requested_full
docs_only=false
languages=golang
EOF

# assert_scope_file <desc> <repo> -- <repo>'s .harness/state/verify-scope
# holds this fixture's full-scope lines, in the format readers expect.
assert_scope_file() {
  _scope="$2/.harness/state/verify-scope"
  if [ -f "$_scope" ] && cmp -s "$expected_scope" "$_scope"; then
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

# --- Case 1: the block dies before writing its exit code ---
printf 'Case 1: a block killed before it writes its exit code fails the run\n'
repo="$(make_repo killed)"
out="$workdir/killed.out"
run_verify "$repo" "$out" STUB_KILL_BLOCK=1
assert_rc_nonzero "killed block makes run-verify.sh fail" "$out"
assert_contains "killed block says the exit code could not be read" \
  "could not read this run's exit code; treating it as a failure" "$out"
assert_no_run_files "killed block leaves no per-run files" "$repo"

# --- Case 2: two concurrent runs in one worktree ---
printf 'Case 2: concurrent failing and passing runs in one worktree keep their own exit codes\n'
repo="$(make_repo concurrent)"
mkdir -p "$repo/.harness/state"
legacy_status="$repo/.harness/state/verify-exit-code"
# An old-style concurrent run: rewrites the shared exit-code file with a
# passing status and removes it, over and over, until the runs are done.
(
  while :; do
    printf '0' > "$legacy_status"
    rm -f "$legacy_status"
    printf '0' > "$legacy_status"
  done
) >/dev/null 2>&1 &
loop_pid=$!
bg_pids="$bg_pids $loop_pid"

a_out="$workdir/concurrent-a.out"
b_out="$workdir/concurrent-b.out"
a_started="$workdir/concurrent-a.started"
release="$workdir/concurrent.release"
start_run_verify "$repo" "$a_out" STUB_EXIT=1 STUB_STARTED="$a_started" STUB_WAIT_FOR="$release"
a_pid=$bg_pid
if wait_for_file "$a_started"; then
  record_pass "run A's verifier started"
else
  record_fail "run A's verifier started (no $a_started after 10 s)"
fi
# Run B starts and finishes while run A's failing verifier is still running.
run_verify "$repo" "$b_out" STUB_EXIT=0
assert_rc "run B (passing) exits 0 while run A is still running" 0 "$b_out"
: > "$release"
wait_bg "$a_pid"
assert_rc "run A (failing) exits 1 after run B passed" 1 "$a_out"
assert_not_contains "run A read its own exit code" "could not read this run's exit code" "$a_out"
kill "$loop_pid" 2>/dev/null || true
wait "$loop_pid" 2>/dev/null || true
assert_no_run_files "concurrent runs leave no per-run files" "$repo"

# --- Cases 3 and 4: plain runs, the scope file, and the temp files ---
printf 'Cases 3 and 4: plain runs keep exit codes 0 and 1 and leave verify-scope behind\n'
repo="$(make_repo plain)"
out="$workdir/plain-pass.out"
run_verify "$repo" "$out" STUB_EXIT=0
assert_rc "plain passing run exits 0" 0 "$out"
assert_contains "plain passing run reports success" "==> All verifiers passed." "$out"
assert_scope_file "passing run leaves .harness/state/verify-scope with the scope lines" "$repo"
assert_no_run_files "passing run leaves no per-run files" "$repo"
assert_absent "passing run does not write the shared exit-code file" "$repo/.harness/state/verify-exit-code"

rm -f "$repo/.harness/state/verify-scope"
out="$workdir/plain-fail.out"
run_verify "$repo" "$out" STUB_EXIT=1
assert_rc "plain failing run exits 1" 1 "$out"
assert_contains "plain failing run reports the failure" "==> Some verifiers failed." "$out"
assert_scope_file "failing run leaves .harness/state/verify-scope with the scope lines" "$repo"
assert_no_run_files "failing run leaves no per-run files" "$repo"
assert_absent "failing run does not write the shared exit-code file" "$repo/.harness/state/verify-exit-code"

# --- Case 5: the docs-only marker ---
printf 'Case 5: with no verifier, a code change still fails and a docs change passes\n'
repo="$(make_repo docs-only)"
printf 'app changed\n' > "$repo/app.txt"
out="$workdir/non-docs.out"
run_verify "$repo" "$out" STUB_LANGUAGES=
assert_rc "code change with no verifier exits 2" 2 "$out"
assert_contains "code change with no verifier says so" "No verifier ran for code-like changes." "$out"
assert_absent "code change does not write the shared non-docs marker" "$repo/.harness/state/non_docs_change"
assert_no_run_files "code change run leaves no per-run files" "$repo"
(cd "$repo" && git checkout -q -- app.txt)
printf '# test repo, edited\n' > "$repo/README.md"
out="$workdir/docs.out"
run_verify "$repo" "$out" STUB_LANGUAGES=
assert_rc "docs-only change with no verifier exits 0" 0 "$out"
assert_contains "docs-only change is reported as docs work" "This appears to be docs or scaffold-level work only." "$out"
assert_no_run_files "docs-only run leaves no per-run files" "$repo"

# --- Case 6: TERM during a run ---
printf 'Case 6: a TERM signal during a run exits 143 and removes the per-run files\n'
repo="$(make_repo term)"
out="$workdir/term.out"
term_started="$workdir/term.started"
term_release="$workdir/term.release"
start_run_verify "$repo" "$out" STUB_EXIT=0 STUB_STARTED="$term_started" STUB_WAIT_FOR="$term_release"
term_pid=$bg_pid
if wait_for_file "$term_started"; then
  record_pass "TERM run's verifier started"
else
  record_fail "TERM run's verifier started (no $term_started after 10 s)"
fi
kill -TERM "$term_pid" 2>/dev/null || true
: > "$term_release"
wait_bg "$term_pid"
assert_rc "TERM during a run exits 143" 143 "$out"
assert_no_run_files "TERM run leaves no per-run files" "$repo"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
