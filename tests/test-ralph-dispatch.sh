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
#   g. A missing event directory produces exit 0 and empty output.
#   h. run-verify.sh / run-static-verify.sh / run-test.sh execute the
#      .ralph/local/verify.d|test.d drop-ins after core verification, with
#      mode selecting which local dir(s) run and a failing drop-in failing
#      the run (AC-5).

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
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT

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
rm -f "$vd_log"
(cd "$verify_fixture" && RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh >/dev/null 2>&1)
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
