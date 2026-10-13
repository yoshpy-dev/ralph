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
#   - a TERM to the run's whole process group ends it at once with 143, and a
#     TERM to its pid alone ends it with 143 only after the running verifier
#     finishes (sh runs the trap after the foreground pipeline), under /bin/sh
#     and, when it is installed, dash;
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

# poll_for_file <path> <polls> -- waits up to <polls> x 0.05 s for <path> to
# appear; returns 1 when it does not.
poll_for_file() {
  _n=0
  while [ ! -f "$1" ]; do
    _n=$((_n + 1))
    if [ "$_n" -gt "$2" ]; then
      return 1
    fi
    sleep 0.05
  done
}

# wait_for_file <path> -- waits up to about 10 s for <path> to appear.
wait_for_file() {
  poll_for_file "$1" 200
}

# start_signal_run <shell> <repo> <out> <done> [VAR=value ...] -- runs
# <repo>'s run-verify.sh under <shell> in the background with the test's
# TMPDIR and the given settings, in a process group of its own when
# own_group is 1 (perl's setpgrp, so a signal to that group never reaches
# this test). Writes run-verify.sh's pid to <done>.pid and, once it exits, its
# exit code to <done>, each by a rename so a reader never sees half a file;
# sets bg_pid to the pid of the wrapper that waits for it.
start_signal_run() {
  _shell="$1"
  _repo="$2"
  _out="$3"
  _done="$4"
  shift 4
  rm -f "$_done" "$_done.pid"
  (
    set +e
    (
      cd "$_repo" || exit 1
      if [ "$own_group" = "1" ]; then
        # shellcheck disable=SC2016  # perl code: $! is perl's, not the shell's
        exec env TMPDIR="$run_tmp" "$@" perl -e 'setpgrp(0, 0) or die "setpgrp: $!\n"; exec @ARGV or die "exec: $!\n"' \
          "$_shell" ./scripts/run-verify.sh
      fi
      exec env TMPDIR="$run_tmp" "$@" "$_shell" ./scripts/run-verify.sh
    ) > "$_out" 2>&1 &
    _run_pid=$!
    printf '%s\n' "$_run_pid" > "$_done.pid.tmp" && mv -f "$_done.pid.tmp" "$_done.pid"
    wait "$_run_pid"
    printf '%s\n' "$?" > "$_done.tmp" && mv -f "$_done.tmp" "$_done"
  ) &
  bg_pid=$!
  bg_pids="$bg_pids $bg_pid"
}

# finish_signal_run <done> <wrapper-pid> -- waits for the wrapper
# start_signal_run started and sets run_rc to the exit code of its
# run-verify.sh (-1 when the wrapper wrote none).
finish_signal_run() {
  wait_bg "$2"
  run_rc="$(cat "$1" 2>/dev/null || true)"
  case "$run_rc" in
    ''|*[!0-9]*) run_rc=-1 ;;
  esac
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
# sh runs a trapped signal's action only after the foreground command it is
# waiting for has ended. A TERM to the run's whole process group (what
# `ralph org gate` sends) also ends the { } | tee pipeline, so the run exits
# 143 at once, while the stub verifier is still waiting for its release. A
# TERM to the run's pid alone exits 143 only once the pipeline has finished:
# the run is still there 1 s after the signal, and ends after the release.
# Both remove the per-run files. The group part needs perl to give the run a
# group of its own; without perl the run stays in this test's group, so the
# part is skipped and the pid part runs there.
printf 'Case 6: a TERM during a run exits 143 and removes the per-run files\n'
case6_shells="/bin/sh"
if command -v dash >/dev/null 2>&1; then
  case6_shells="$case6_shells $(command -v dash)"
fi
own_group=0
if command -v perl >/dev/null 2>&1; then
  own_group=1
fi
for case6_shell in $case6_shells; do
  sh_name="$(basename "$case6_shell")"

  if [ "$own_group" = "1" ]; then
    repo="$(make_repo "term-group-$sh_name")"
    out="$workdir/term-group-$sh_name.out"
    started="$workdir/term-group-$sh_name.started"
    release="$workdir/term-group-$sh_name.release"
    done_file="$workdir/term-group-$sh_name.done"
    start_signal_run "$case6_shell" "$repo" "$out" "$done_file" STUB_EXIT=0 STUB_STARTED="$started" STUB_WAIT_FOR="$release"
    wrapper_pid=$bg_pid
    if wait_for_file "$started" && wait_for_file "$done_file.pid"; then
      run_pid="$(cat "$done_file.pid")"
      kill -s TERM -- "-$run_pid" 2>/dev/null || true
      if poll_for_file "$done_file" 40; then
        record_pass "$sh_name: TERM to the run's process group ends it within 2 s, the verifier not released"
      else
        record_fail "$sh_name: TERM to the run's process group did not end it within 2 s"
      fi
    else
      record_fail "$sh_name: the group TERM run's verifier did not start within 10 s"
    fi
    : > "$release"
    finish_signal_run "$done_file" "$wrapper_pid"
    assert_rc "$sh_name: TERM to the run's process group exits 143" 143 "$out"
    assert_no_run_files "$sh_name: TERM to the run's process group leaves no per-run files" "$repo"
    if [ -n "${run_pid:-}" ]; then
      kill -s KILL -- "-$run_pid" 2>/dev/null || true
    fi
  else
    printf '  SKIP: %s: TERM to the process group (perl is not installed)\n' "$sh_name"
  fi

  repo="$(make_repo "term-pid-$sh_name")"
  out="$workdir/term-pid-$sh_name.out"
  started="$workdir/term-pid-$sh_name.started"
  release="$workdir/term-pid-$sh_name.release"
  done_file="$workdir/term-pid-$sh_name.done"
  run_pid=""
  start_signal_run "$case6_shell" "$repo" "$out" "$done_file" STUB_EXIT=0 STUB_STARTED="$started" STUB_WAIT_FOR="$release"
  wrapper_pid=$bg_pid
  if wait_for_file "$started" && wait_for_file "$done_file.pid"; then
    run_pid="$(cat "$done_file.pid")"
    kill -s TERM "$run_pid" 2>/dev/null || true
    if poll_for_file "$done_file" 20; then
      record_fail "$sh_name: TERM to the run's pid alone ended it before the verifier was released"
    else
      record_pass "$sh_name: TERM to the run's pid alone leaves it waiting for the verifier (1 s)"
    fi
  else
    record_fail "$sh_name: the pid TERM run's verifier did not start within 10 s"
  fi
  : > "$release"
  finish_signal_run "$done_file" "$wrapper_pid"
  assert_rc "$sh_name: TERM to the run's pid alone exits 143 after the verifier finishes" 143 "$out"
  assert_no_run_files "$sh_name: TERM to the run's pid alone leaves no per-run files" "$repo"
  if [ "$own_group" = "1" ] && [ -n "$run_pid" ]; then
    kill -s KILL -- "-$run_pid" 2>/dev/null || true
  fi
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
