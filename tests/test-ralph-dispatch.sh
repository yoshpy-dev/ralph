#!/usr/bin/env bash
# test-ralph-dispatch.sh — smoke tests for
# templates/base/.claude/hooks/ralph-dispatch.sh.
#
# Cases:
#   a. PreToolUse deny decision (real permissionDecision JSON payload) is
#      passed through verbatim; a later script does not run.
#   b. PostToolUse additionalContext JSON + plain text from two scripts
#      merge into a single valid JSON hook response.
#   c. Single-script output is passed through byte-exact.
#   d. Non-zero exit from a script propagates and stops later scripts.
#   e. stdin (the hook payload) reaches every script.
#   f. A local drop-in under .ralph/local/hooks/<event>.d/ runs after core
#      .claude/hooks/<event>.d/ drop-ins.
#   f2. All three layers (core .d -> .ralph/local/hooks/<event>.d ->
#       .claude/hooks/local/<event>.d) execute in that order in one fixture.
#   g. A missing event directory produces exit 0 and empty output.
#   h. run-verify.sh / run-static-verify.sh / run-test.sh execute the
#      .ralph/local/verify.d|test.d drop-ins after core verification, with
#      mode selecting which local dir(s) run and a failing drop-in failing
#      the run (AC-5).
#   i. SIGTERM sent directly to the dispatcher process (not a wrapping
#      subshell) mid-run terminates it promptly (does not resume, is not
#      merely deferred until the running hook script finishes on its own)
#      with exit 143, and its cleanup trap leaves no stray temp files in a
#      dispatcher-private TMPDIR or a cwd-relative ".first" in the fixture.
#      A concurrently-running second dispatcher (separate fixture repo,
#      separate event/hook name) holds its own temp files in the suite's
#      simulated shared TMPDIR for the whole check, proving the private
#      TMPDIR -- not the shared one -- is what the cleanup check inspects.
#      A mid-run check confirms the target dispatcher's own files actually
#      land in its private TMPDIR (not just that they are absent from it
#      afterward, which cleanup alone cannot distinguish from never having
#      been written there at all). Both dispatchers' hook scripts carry a
#      per-run name (this test process's own PID), so two runs of this
#      suite on one host never match each other's pgrep/pkill and cannot
#      kill or "see" each other's process. The same SIGTERM also kills the
#      hook script that was actively running underneath the dispatcher,
#      verified two ways: it never reaches the marker it would only write
#      on natural completion, and (defense in depth) no matching process
#      remains alive per pgrep.

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DISPATCHER_SRC="$REPO_ROOT/templates/base/.claude/hooks/ralph-dispatch.sh"
LIB_JSON_SRC="$REPO_ROOT/templates/base/.claude/hooks/lib_json.sh"
RUN_VERIFY_SRC="$REPO_ROOT/templates/base/scripts/run-verify.sh"
RUN_STATIC_VERIFY_SRC="$REPO_ROOT/templates/base/scripts/run-static-verify.sh"
RUN_TEST_SRC="$REPO_ROOT/templates/base/scripts/run-test.sh"

if [ ! -x "$DISPATCHER_SRC" ]; then
  echo "FAIL: dispatcher not found or not executable at $DISPATCHER_SRC" >&2
  exit 1
fi

pass=0
fail=0
results=()

record_pass() {
  results+=("PASS  $1")
  pass=$((pass + 1))
}
record_fail() {
  results+=("FAIL  $1")
  fail=$((fail + 1))
}

assert_exit() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    record_pass "$label (exit $actual)"
  else
    record_fail "$label (expected exit $expected, got $actual)"
  fi
}

assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    record_pass "$label"
  else
    record_fail "$label (expected [$expected], got [$actual])"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-dispatch-test.XXXXXX")"
concurrent_pid=""
cleanup() {
  # Stop case I's concurrent-dispatcher fixture if the suite exits (early
  # failure, signal) before case I's own explicit stop runs. concurrent_pid
  # is set right after the fixture is launched, before its start is even
  # confirmed, and stays set until case I's own explicit stop clears it back
  # to "" -- see case I below. Killing/waiting on an already-dead PID here
  # is a harmless no-op. No pkill safety net is needed: the fixture hook
  # ends in "exec sleep 30" (see case I), so the dispatcher's own kill_child
  # reaches the sleep directly instead of leaving it orphaned.
  if [ -n "$concurrent_pid" ]; then
    kill -TERM "$concurrent_pid" 2>/dev/null
    wait "$concurrent_pid" 2>/dev/null
  fi
  rm -rf "$workdir"
}
trap cleanup EXIT

# Simulated shared TMPDIR (issue #182): every dispatcher this suite runs --
# cases A-H, case H's run-verify.sh/run-test.sh mktemp calls, and case I's
# concurrent-dispatcher fixture below (plus its pre-fix reproduction, done
# separately in a throwaway scratch copy of this file, never committed) --
# gets this directory instead of the host's real ${TMPDIR:-/tmp}. That is
# fine because nothing in this suite ever needs to see another process's
# real temp files: cases A-H only assert on their own fixtures' output, and
# case I's own dispatcher run below gets a further-private TMPDIR on top of
# this one. The result: this suite never lists or deletes another session's
# real temp files, and a leaked temp file can never land outside $workdir.
# $workdir itself was created above using the host's TMPDIR (or /tmp) --
# that is the only host-shared-dir use in this suite.
shared_tmp="$workdir/shared-tmp"
mkdir -p "$shared_tmp"
TMPDIR="$shared_tmp"
export TMPDIR

# Fixture repo: copy dispatcher + lib_json.sh under .claude/hooks/, with
# per-case event .d directories created as needed.
fixture="$workdir/repo"
mkdir -p "$fixture/.claude/hooks"
cp "$DISPATCHER_SRC" "$fixture/.claude/hooks/ralph-dispatch.sh"
cp "$LIB_JSON_SRC" "$fixture/.claude/hooks/lib_json.sh"
chmod +x "$fixture/.claude/hooks/ralph-dispatch.sh"

write_script() {
  # write_script <path> <heredoc body via stdin>
  local path="$1"
  mkdir -p "$(dirname "$path")"
  cat > "$path"
  chmod +x "$path"
}

dispatch() {
  # dispatch <event> <payload>; sets DISPATCH_OUT, DISPATCH_RC
  local event="$1" payload="$2"
  DISPATCH_OUT="$(printf '%s' "$payload" | (cd "$fixture" && ./.claude/hooks/ralph-dispatch.sh "$event"))"
  DISPATCH_RC=$?
}

# ── Case A: PreToolUse deny decision, real payload ──────────────────────
rm -rf "$fixture/.claude/hooks/PreToolUse.d"
write_script "$fixture/.claude/hooks/PreToolUse.d/10-deny.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"blocked by fixture"}}\n'
EOF
write_script "$fixture/.claude/hooks/PreToolUse.d/20-never.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
echo "MUST_NOT_RUN" >> "$MARKER_FILE"
printf 'unexpected\n'
EOF
marker="$workdir/case-a-marker"
rm -f "$marker"
DISPATCH_OUT=""
DISPATCH_RC=0
DISPATCH_OUT="$(printf '%s' '{"tool_name":"Bash","tool_input":{"command":"sudo rm -rf /"}}' | (cd "$fixture" && MARKER_FILE="$marker" ./.claude/hooks/ralph-dispatch.sh PreToolUse))"
DISPATCH_RC=$?
assert_exit "A. PreToolUse deny exits 0" 0 "$DISPATCH_RC"
expected_deny='{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"blocked by fixture"}}'
assert_eq "A. PreToolUse deny output is exact JSON" "$expected_deny" "$DISPATCH_OUT"
if [ -f "$marker" ]; then
  record_fail "A. later script did not run after decision (marker exists)"
else
  record_pass "A. later script skipped after first decision"
fi

# ── Case B: PostToolUse additionalContext JSON + plain text merge ──────
rm -rf "$fixture/.claude/hooks/PostToolUse.d"
write_script "$fixture/.claude/hooks/PostToolUse.d/10-a.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"Code file edited. Run ./scripts/run-verify.sh before claiming done."}}\n'
EOF
write_script "$fixture/.claude/hooks/PostToolUse.d/20-b.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
printf 'plain text reminder from script B\n'
EOF
dispatch "PostToolUse" '{"tool_name":"Edit","tool_input":{"file_path":"foo.go"}}'
assert_exit "B. PostToolUse merge exits 0" 0 "$DISPATCH_RC"
if command -v jq >/dev/null 2>&1; then
  if printf '%s' "$DISPATCH_OUT" | jq -e '.hookSpecificOutput.additionalContext' >/dev/null 2>&1; then
    record_pass "B. merged output is valid JSON with additionalContext"
  else
    record_fail "B. merged output is not valid JSON with additionalContext: $DISPATCH_OUT"
  fi
  ctx="$(printf '%s' "$DISPATCH_OUT" | jq -r '.hookSpecificOutput.additionalContext')"
  case "$ctx" in
    *"Run ./scripts/run-verify.sh"*"plain text reminder from script B"*)
      record_pass "B. merged additionalContext contains both scripts' text"
      ;;
    *)
      record_fail "B. merged additionalContext missing expected text: $ctx"
      ;;
  esac
else
  record_fail "B. jq not available to verify merged JSON (test environment gap)"
fi

# ── Case C: single-script byte-exact passthrough ────────────────────────
rm -rf "$fixture/.claude/hooks/UserPromptSubmit.d"
write_script "$fixture/.claude/hooks/UserPromptSubmit.d/10-only.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"single reminder"}}\n'
EOF
dispatch "UserPromptSubmit" '{"prompt":"hello"}'
assert_exit "C. single-output passthrough exits 0" 0 "$DISPATCH_RC"
expected_single='{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"single reminder"}}'
assert_eq "C. single-output passthrough is byte-exact" "$expected_single" "$DISPATCH_OUT"

# ── Case D: non-zero exit propagates, later scripts skipped ────────────
rm -rf "$fixture/.claude/hooks/PostToolUseFailure.d"
write_script "$fixture/.claude/hooks/PostToolUseFailure.d/10-fail.sh" <<'EOF'
#!/usr/bin/env sh
cat >/dev/null
echo "failing script stdout"
exit 3
EOF
marker_d="$workdir/case-d-marker"
rm -f "$marker_d"
write_script "$fixture/.claude/hooks/PostToolUseFailure.d/20-never.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null
echo "MUST_NOT_RUN" >> "$marker_d"
EOF
dispatch "PostToolUseFailure" '{"tool_name":"Bash"}'
assert_exit "D. non-zero exit propagates" 3 "$DISPATCH_RC"
assert_eq "D. failing script stdout emitted verbatim" "failing script stdout" "$DISPATCH_OUT"
if [ -f "$marker_d" ]; then
  record_fail "D. later script ran after failure (should be skipped)"
else
  record_pass "D. later script skipped after non-zero exit"
fi

# ── Case E: stdin reaches every script ───────────────────────────────────
rm -rf "$fixture/.claude/hooks/SessionStart.d"
stdin_log="$workdir/stdin.log"
rm -f "$stdin_log"
write_script "$fixture/.claude/hooks/SessionStart.d/10-first.sh" <<EOF
#!/usr/bin/env sh
p="\$(cat)"
printf 'first:%s\n' "\$p" >> "$stdin_log"
EOF
write_script "$fixture/.claude/hooks/SessionStart.d/20-second.sh" <<EOF
#!/usr/bin/env sh
p="\$(cat)"
printf 'second:%s\n' "\$p" >> "$stdin_log"
EOF
dispatch "SessionStart" 'session-payload-xyz'
if grep -Fqx "first:session-payload-xyz" "$stdin_log" && grep -Fqx "second:session-payload-xyz" "$stdin_log"; then
  record_pass "E. stdin payload reached both scripts"
else
  record_fail "E. stdin payload missing from one or both scripts: $(cat "$stdin_log" 2>/dev/null)"
fi

# ── Case F: .ralph/local drop-in runs after core .d ─────────────────────
rm -rf "$fixture/.claude/hooks/SessionEnd.d" "$fixture/.ralph"
order_log="$workdir/order.log"
rm -f "$order_log"
write_script "$fixture/.claude/hooks/SessionEnd.d/10-core.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null
echo core >> "$order_log"
EOF
write_script "$fixture/.ralph/local/hooks/SessionEnd.d/10-local.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null
echo local >> "$order_log"
EOF
dispatch "SessionEnd" '{}'
actual_order="$(cat "$order_log" 2>/dev/null | tr '\n' ',')"
assert_eq "F. core .d runs before .ralph/local/hooks .d" "core,local," "$actual_order"

# ── Case F2: all 3 layers run core → .ralph/local → .claude/hooks/local ─
rm -rf "$fixture/.claude/hooks/SessionEnd.d" "$fixture/.ralph" "$fixture/.claude/hooks/local"
o2="$workdir/order2.log"; rm -f "$o2"
write_script "$fixture/.claude/hooks/SessionEnd.d/10-core.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null; echo core >> "$o2"
EOF
write_script "$fixture/.ralph/local/hooks/SessionEnd.d/10-local.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null; echo local >> "$o2"
EOF
write_script "$fixture/.claude/hooks/local/SessionEnd.d/10-gi.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null; echo gitignored >> "$o2"
EOF
dispatch "SessionEnd" '{}'
assert_eq "F2. three-layer order core,local,gitignored" "core,local,gitignored," "$(cat "$o2" 2>/dev/null | tr '\n' ',')"

# ── Case G: missing event dir → exit 0, empty output ────────────────────
dispatch "NoSuchEventAtAll" '{}'
assert_exit "G. missing event dir exits 0" 0 "$DISPATCH_RC"
assert_eq "G. missing event dir produces empty output" "" "$DISPATCH_OUT"

# ── Case H: run-verify.sh / run-static-verify.sh / run-test.sh drop-ins ──
verify_fixture="$workdir/verify-repo"
mkdir -p "$verify_fixture/scripts" "$verify_fixture/.ralph/local/verify.d" "$verify_fixture/.ralph/local/test.d"
cp "$RUN_VERIFY_SRC" "$verify_fixture/scripts/run-verify.sh"
cp "$RUN_STATIC_VERIFY_SRC" "$verify_fixture/scripts/run-static-verify.sh"
cp "$RUN_TEST_SRC" "$verify_fixture/scripts/run-test.sh"
chmod +x "$verify_fixture/scripts/"*.sh

write_script "$verify_fixture/scripts/detect-languages.sh" <<'EOF'
#!/usr/bin/env sh
printf ''
EOF

vd_log="$workdir/verify-dropin.log"
write_script "$verify_fixture/.ralph/local/verify.d/10-mark.sh" <<EOF
#!/usr/bin/env sh
echo verify.d >> "$vd_log"
EOF
write_script "$verify_fixture/.ralph/local/test.d/10-mark.sh" <<EOF
#!/usr/bin/env sh
echo test.d >> "$vd_log"
EOF

# H1: HARNESS_VERIFY_MODE=static (via run-static-verify.sh) runs verify.d only.
rm -f "$vd_log"
(cd "$verify_fixture" && RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh >/dev/null 2>&1)
h1_actual="$(cat "$vd_log" 2>/dev/null | tr '\n' ',')"
assert_eq "H1. run-static-verify.sh runs only verify.d" "verify.d," "$h1_actual"

# H2: HARNESS_VERIFY_MODE=test (via run-test.sh) runs test.d only.
rm -f "$vd_log"
(cd "$verify_fixture" && RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh >/dev/null 2>&1)
h2_actual="$(cat "$vd_log" 2>/dev/null | tr '\n' ',')"
assert_eq "H2. run-test.sh runs only test.d" "test.d," "$h2_actual"

# H3: direct run-verify.sh (mode=all default) runs both verify.d and test.d.
# HARNESS_VERIFY_MODE is explicitly unset here (rather than left to default)
# so this case stays hermetic when the outer test run itself was launched
# via run-test.sh, which exports HARNESS_VERIFY_MODE=test into this very
# process tree and would otherwise leak into the fixture invocation below.
rm -f "$vd_log"
(cd "$verify_fixture" && unset HARNESS_VERIFY_MODE; RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh >/dev/null 2>&1)
h3_actual="$(cat "$vd_log" 2>/dev/null | sort | tr '\n' ',')"
assert_eq "H3. run-verify.sh (mode=all) runs both verify.d and test.d" "test.d,verify.d," "$h3_actual"

# H4: a failing drop-in fails the run.
write_script "$verify_fixture/.ralph/local/verify.d/20-fail.sh" <<'EOF'
#!/usr/bin/env sh
echo "drop-in failure" >&2
exit 1
EOF
h4_rc=0
(cd "$verify_fixture" && RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh >/dev/null 2>&1) || h4_rc=$?
if [ "$h4_rc" -ne 0 ]; then
  record_pass "H4. failing verify.d drop-in fails run-static-verify.sh (exit $h4_rc)"
else
  record_fail "H4. failing verify.d drop-in did not fail the run (exit 0)"
fi
rm -f "$verify_fixture/.ralph/local/verify.d/20-fail.sh"

# ── Case I: SIGTERM to the dispatcher itself (not a wrapping subshell)
#           terminates it (does not resume) and its cleanup trap removes
#           every mktemp'd temp file from a dispatcher-private TMPDIR,
#           leaving nothing stray there or a cwd-relative ".first" in the
#           fixture, even while a second, concurrently-running dispatcher
#           holds its own temp files in the suite's simulated shared
#           TMPDIR ───────────────────────────────────────────────────
rm -rf "$fixture/.claude/hooks/PreCompact.d"
started_marker="$workdir/case-i-started"
finished_marker="$workdir/case-i-finished"
i_stdin="$workdir/case-i-stdin.json"
i_out="$workdir/case-i-out.log"
rm -f "$started_marker" "$finished_marker" "$fixture/.first" "$i_stdin" "$i_out"
printf '{}' > "$i_stdin"
# finished_marker is only written once "sleep 5" runs to completion. This is
# the assertion that actually distinguishes fixed from unfixed dispatcher
# behavior: POSIX shells defer a trapped signal's action until the current
# *foreground external command* returns, so sending TERM to a dispatcher
# blocked on "$script" < ... > ... (no backgrounding) does not interrupt
# that wait -- it silently queues, and the trap (and this script's "wait
# $i_pid" below) only unblocks once the child finishes on its own 5s later,
# by which point finished_marker already exists and a same-tick pgrep-based
# check would find nothing to fail on. Checking finished_marker immediately
# after "wait $i_pid" returns catches this regardless of how quickly (or
# slowly) that wait unblocks.
#
# The hook's file name carries this test process's own PID ($$), the same
# as the concurrent fixture's hook below: the dispatcher runs hooks by a
# $workdir-relative path with nothing per-run in the command line, so a
# bare "10-slow.sh" would let the pgrep/pkill checks below match (and the
# pkill kill) a sibling run's own hook process when two runs of this suite
# race on one host.
write_script "$fixture/.claude/hooks/PreCompact.d/10-slow-$$.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null
touch "$started_marker"
sleep 5
touch "$finished_marker"
EOF

# Concurrent-dispatcher fixture: a second, independent fixture repo running
# its own dispatcher against the suite's simulated shared TMPDIR ($shared_tmp,
# exported as TMPDIR above), started before and stopped after case I's own
# dispatcher run below. This models the condition behind the flake in #182:
# another Claude Code/Codex session's hook run through the same
# ralph-dispatch.sh leaves its own "ralph-dispatch-*" temp files in a shared
# TMPDIR while case I's dispatcher is mid-run. Its hook lives under a
# different event (Stop, not PreCompact) and a per-run-named script (below)
# so it never matches case I's own pgrep/finished_marker checks.
concurrent_fixture="$workdir/repo2"
mkdir -p "$concurrent_fixture/.claude/hooks"
cp "$DISPATCHER_SRC" "$concurrent_fixture/.claude/hooks/ralph-dispatch.sh"
cp "$LIB_JSON_SRC" "$concurrent_fixture/.claude/hooks/lib_json.sh"
chmod +x "$concurrent_fixture/.claude/hooks/ralph-dispatch.sh"
concurrent_started="$workdir/case-i-concurrent-started"
concurrent_out="$workdir/case-i-concurrent-out.log"
rm -f "$concurrent_started" "$concurrent_out"
# The hook's file name carries this test process's own PID ($$), the same
# as case I's own hook above, so it can never collide with another
# concurrent run of this same suite on the host -- a bare
# "10-concurrent-slow.sh" would, since the dispatcher runs hooks by a
# $workdir-relative path with nothing per-run in the command line, and a
# pkill/pgrep on that path would then match (and could kill) a sibling
# run's fixture too.
# "exec sleep 30" (not a plain "sleep 30") replaces this script's own
# process image with sleep, so the TERM the dispatcher's kill_child sends
# to this script's PID lands on the sleep itself instead of leaving it
# orphaned once this script's own shell would otherwise have exited.
write_script "$concurrent_fixture/.claude/hooks/Stop.d/10-concurrent-slow-$$.sh" <<EOF
#!/usr/bin/env sh
cat >/dev/null
touch "$concurrent_started"
exec sleep 30
EOF

# Baseline: nothing in this suite writes to $shared_tmp before this point
# (cases A-H's dispatcher runs and case H's run-verify.sh/run-test.sh calls
# have all already returned and cleaned up their own mktemp files), so it
# must be empty before the concurrent fixture starts. A failure here means
# something in cases A-H (or a prior run of this suite) left files behind,
# not that the concurrent fixture is misbehaving.
shared_tmp_baseline="$(find "$shared_tmp" -maxdepth 1 -name 'ralph-dispatch-*' 2>/dev/null | sort | paste -sd ' ' -)"
if [ -n "$shared_tmp_baseline" ]; then
  record_fail "I. simulated shared TMPDIR already had ralph-dispatch-* files before the concurrent fixture started: $shared_tmp_baseline"
else
  record_pass "I. simulated shared TMPDIR had no ralph-dispatch-* files before the concurrent fixture started"
fi

(
  cd "$concurrent_fixture" || exit 1
  TMPDIR="$shared_tmp"
  export TMPDIR
  exec ./.claude/hooks/ralph-dispatch.sh Stop < /dev/null > "$concurrent_out" 2>&1
) &
concurrent_pid=$!
concurrent_started_ok=0
for _ in $(seq 1 50); do
  [ -f "$concurrent_started" ] && { concurrent_started_ok=1; break; }
  sleep 0.1
done
if [ "$concurrent_started_ok" -eq 1 ]; then
  record_pass "I. concurrent-dispatcher fixture started"
else
  record_fail "I. concurrent-dispatcher fixture did not start within 5s (started marker never appeared)"
fi

# Sanity check that the fixture is actually effective: the concurrent
# dispatcher's own mktemp'd files must be visible in the simulated shared
# TMPDIR while it is running, since that is the exact condition case I's
# private-TMPDIR fix must be immune to. Count only (not the actual paths,
# which are per-run and would make two green result blocks never match) --
# see the FAIL branches below for paths when something is actually wrong.
concurrent_tmp_count="$(find "$shared_tmp" -maxdepth 1 -name 'ralph-dispatch-*' 2>/dev/null | wc -l | tr -d ' ')"
if [ "$concurrent_tmp_count" -gt 0 ]; then
  record_pass "I. concurrent dispatcher's temp files present in simulated shared TMPDIR ($concurrent_tmp_count files)"
else
  record_fail "I. concurrent dispatcher's temp files missing from simulated shared TMPDIR (fixture not effective)"
fi

# case I's own dispatcher run gets a private TMPDIR ($workdir/case-i-tmp),
# set inside the backgrounded subshell before exec, so its cleanup check
# below lists only that directory -- never the shared TMPDIR the
# concurrent-dispatcher fixture above is using at the same time.
case_i_tmpdir="$workdir/case-i-tmp"
mkdir -p "$case_i_tmpdir"

# Background the dispatcher process itself, not a wrapping subshell: `exec`
# after `cd` replaces the backgrounded subshell's own process image with
# the dispatcher, so the PID captured in $! is the dispatcher's PID and
# `kill -TERM` below signals it directly (a plain
# `( cd ... && ... ) &` would instead signal the shell that ran `cd`,
# leaving the dispatcher itself, forked deeper, untouched).
(
  cd "$fixture" || exit 1
  TMPDIR="$case_i_tmpdir"
  export TMPDIR
  exec ./.claude/hooks/ralph-dispatch.sh PreCompact < "$i_stdin" > "$i_out" 2>&1
) &
i_pid=$!
for _ in $(seq 1 50); do
  [ -f "$started_marker" ] && break
  sleep 0.1
done

# These two mid-run checks are what actually pin the private-TMPDIR fix.
# The post-TERM cleanup checks below cannot: ralph-dispatch.sh's own trap
# removes its mktemp'd files on exit regardless of which directory TMPDIR
# pointed at when they were created, so if the "TMPDIR=$case_i_tmpdir"
# override above were silently dropped, the target would write into the
# shared $shared_tmp instead and still leave both directories looking
# clean afterward -- only a live look while the target is still running
# can tell the two cases apart.
i_tmpdir_live_count="$(find "$case_i_tmpdir" -mindepth 1 -maxdepth 1 2>/dev/null | wc -l | tr -d ' ')"
if [ "$i_tmpdir_live_count" -gt 0 ]; then
  record_pass "I. target dispatcher's temp files present in its private TMPDIR while running ($i_tmpdir_live_count files)"
else
  record_fail "I. target dispatcher's temp files missing from its private TMPDIR ($case_i_tmpdir) while running"
fi

i_shared_tmp_count="$(find "$shared_tmp" -maxdepth 1 -name 'ralph-dispatch-*' 2>/dev/null | wc -l | tr -d ' ')"
if [ "$i_shared_tmp_count" -eq "$concurrent_tmp_count" ]; then
  record_pass "I. simulated shared TMPDIR still holds only the concurrent fixture's temp files while the target runs ($i_shared_tmp_count files)"
else
  i_shared_tmp_entries="$(find "$shared_tmp" -maxdepth 1 -name 'ralph-dispatch-*' 2>/dev/null | sort | paste -sd ' ' -)"
  record_fail "I. simulated shared TMPDIR gained unexpected entries while the target ran (expected $concurrent_tmp_count from the concurrent fixture, got $i_shared_tmp_count): $i_shared_tmp_entries"
fi

i_term_start="$(date +%s)"
kill -TERM "$i_pid" 2>/dev/null
i_rc=0
wait "$i_pid" 2>/dev/null || i_rc=$?
i_term_elapsed=$(( $(date +%s) - i_term_start ))
assert_exit "I. SIGTERM to the dispatcher terminates it with non-zero exit (not resumed)" 143 "$i_rc"

# The dispatcher must be interrupted promptly, not just eventually: sending
# TERM must not merely queue behind the 5s hook script the dispatcher is
# running. 3s leaves generous margin over typical prompt-interruption
# latency while still being well under the fixture's 5s sleep.
if [ "$i_term_elapsed" -le 3 ]; then
  record_pass "I. SIGTERM interrupted the dispatcher promptly (${i_term_elapsed}s, not deferred until the hook child finished on its own)"
else
  record_fail "I. SIGTERM took ${i_term_elapsed}s to terminate the dispatcher (want <=3s; suggests the trap was deferred until the hook child finished on its own)"
fi

if [ -f "$finished_marker" ]; then
  record_fail "I. SIGTERM did not stop the running hook script child -- it ran to completion (finished_marker exists)"
else
  record_pass "I. SIGTERM stopped the running hook script child before it completed (finished_marker absent)"
fi

if [ -f "$fixture/.first" ]; then
  record_fail "I. SIGTERM cleanup left a stray .first file in fixture cwd"
else
  record_pass "I. SIGTERM cleanup left no stray .first file in fixture cwd"
fi

# $case_i_tmpdir belongs to case I's own dispatcher alone (nothing else in
# this suite writes there), so any entry at all -- not just a
# "ralph-dispatch-*" one -- is a leak.
if [ ! -d "$case_i_tmpdir" ]; then
  record_fail "I. private TMPDIR $case_i_tmpdir is missing after the dispatcher exited (cannot check for leftovers)"
else
  i_tmp_leftover="$(find "$case_i_tmpdir" -mindepth 1 -maxdepth 1 2>/dev/null | sort | paste -sd ' ' -)"
  if [ -n "$i_tmp_leftover" ]; then
    record_fail "I. SIGTERM cleanup left stray entries in its private TMPDIR: $i_tmp_leftover"
  else
    record_pass "I. SIGTERM cleanup left no stray entries in its private TMPDIR"
  fi
fi

# I (child-kill, defense in depth): no process matching the hook script's
# own path should remain alive after the dispatcher has exited. This is a
# secondary check -- finished_marker above is the one that reliably catches
# a reverted fix, since (per the comment at its declaration) an unfixed
# dispatcher's "wait $i_pid" does not return until the child has already
# exited on its own, so pgrep alone would find nothing to fail on either way.
if pgrep -f "PreCompact.d/10-slow-$$.sh" >/dev/null 2>&1; then
  record_fail "I. SIGTERM left the running hook script child alive (dispatcher did not kill it)"
else
  record_pass "I. SIGTERM killed the running hook script child (no longer alive after dispatcher exit)"
fi

# Reap the orphaned drop-in (still sleeping past its parent's death) so it
# does not outlive this test run — a no-op once the child-kill assertion
# above passes; kept as a safety net if it does not. The per-run path
# (this test process's own $$, matching the hook's file name above) means
# this pkill can only ever match this run's own hook, never a sibling run's.
pkill -f "PreCompact.d/10-slow-$$.sh" >/dev/null 2>&1 || true

# Stop the concurrent-dispatcher fixture and confirm its own temp files are
# gone from the simulated shared TMPDIR. This runs after all of case I's
# own private-TMPDIR checks above, which must not depend on the concurrent
# fixture having stopped first.
concurrent_rc=0
kill -TERM "$concurrent_pid" 2>/dev/null
wait "$concurrent_pid" 2>/dev/null || concurrent_rc=$?
concurrent_pid=""

# The dispatcher's own TERM trap (kill_child, then exit 143 -- see
# ralph-dispatch.sh) TERMs its hook's PID and *waits* for it to exit before
# the trap itself returns. Because the hook ends in "exec sleep 30" above,
# that PID *is* the sleep process by the time TERM arrives, so seeing exit
# 143 here means the exec'd sleep was already reaped before this dispatcher
# exited, not merely signaled and abandoned. A pgrep-based check is not
# possible here the way case I's own child-kill check (above) does it:
# once "exec" replaces the hook script with "sleep 30", its command line is
# just "sleep 30", indistinguishable from an unrelated sleep on the host.
if [ "$concurrent_rc" -eq 143 ]; then
  record_pass "I. concurrent dispatcher's SIGTERM handling reaped its exec'd sleep child before exiting (exit 143)"
else
  record_fail "I. concurrent dispatcher did not exit 143 on SIGTERM (exit $concurrent_rc); its exec'd sleep child's fate is unverified"
fi

concurrent_tmp_after="$(find "$shared_tmp" -maxdepth 1 -name 'ralph-dispatch-*' 2>/dev/null | sort | paste -sd ' ' -)"
if [ -z "$concurrent_tmp_after" ]; then
  record_pass "I. concurrent dispatcher's temp files removed from simulated shared TMPDIR after it was stopped"
else
  record_fail "I. concurrent dispatcher's temp files remained in simulated shared TMPDIR after it was stopped: $concurrent_tmp_after"
fi

# ── Summary ───────────────────────────────────────────────────────────
echo
echo "=== ralph-dispatch.sh test results ==="
for line in "${results[@]}"; do
  echo "  $line"
done
echo
echo "  PASS: $pass"
echo "  FAIL: $fail"

if [ "$fail" -gt 0 ]; then
  exit 1
fi
exit 0
