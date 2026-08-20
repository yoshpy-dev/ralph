#!/usr/bin/env bash
# test-post-edit-verify.sh — smoke tests for .claude/hooks/post_edit_verify.sh.
#
# Cases:
#   A. Claude Code Edit payload (tool_input.file_path) -> unchanged
#      behavior: single line logged, code-class message emitted
#   B. Claude Code Edit payload for a doc-class path -> doc-class message
#   C. Codex apply_patch payload with two files (Add + Update) -> both
#      derived paths appended to edited-files.log
#   D. apply_patch payload mixing a doc-class and a code-class path ->
#      code-class message wins (precedence: code > doc > skip)
#   E. apply_patch Delete-only patch -> no crash, exit 0, deleted path
#      still logged (this hook only classifies/logs; it never touches disk)
#   F. Payload with neither tool_input.file_path nor a patch-shaped
#      tool_input.command (e.g. a Bash tool call) -> exit 0, needs-verify
#      still touched, no edited-files.log entries

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$REPO_ROOT/.claude/hooks/post_edit_verify.sh"

if [ ! -x "$HOOK" ]; then
  echo "FAIL: hook not found or not executable at $HOOK" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "FAIL: jq is required for this test" >&2
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
    record_fail "$label (expected $expected, got $actual)"
  fi
}

assert_contains() {
  local label="$1" haystack="$2" needle="$3"
  if printf '%s' "$haystack" | grep -qF "$needle"; then
    record_pass "$label"
  else
    record_fail "$label (missing '$needle')"
  fi
}

assert_not_contains() {
  local label="$1" haystack="$2" needle="$3"
  if printf '%s' "$haystack" | grep -qF "$needle"; then
    record_fail "$label (unexpectedly contains '$needle')"
  else
    record_pass "$label"
  fi
}

assert_log_lines() {
  # assert_log_lines <label> <log-file> <expected-line-count>
  local label="$1" log="$2" expected="$3" actual=0
  if [ -f "$log" ]; then
    actual="$(wc -l < "$log" | tr -d ' ')"
  fi
  if [ "$expected" = "$actual" ]; then
    record_pass "$label ($actual lines)"
  else
    record_fail "$label (expected $expected lines, got $actual)"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/post-edit-verify-test.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

# run_hook <case-dir> <payload> — invokes the hook with cwd scoped to
# $workdir/<case-dir>, mirroring how ralph-dispatch.sh always launches it
# from the repo root: .harness/state/ writes land under that scoped dir,
# not the real repo.
run_hook() {
  local dir="$1" payload="$2"
  mkdir -p "$workdir/$dir"
  ( cd "$workdir/$dir" && printf '%s' "$payload" | "$HOOK" )
}

make_apply_patch_payload() {
  jq -n --arg cmd "$1" '{"session_id":"t","tool_name":"apply_patch","tool_input":{"command":$cmd}}'
}

# ── Case A: Claude Code Edit payload, code-class path ───────────────
dir="case-a"
payload='{"session_id":"t","tool_name":"Edit","tool_input":{"file_path":"internal/cli/doctor.go"}}'
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "A. file_path payload exits 0" 0 "$actual"
assert_log_lines "A. edited-files.log has one entry" "$workdir/$dir/.harness/state/edited-files.log" 1
assert_contains "A. edited-files.log contains the file_path" "$(cat "$workdir/$dir/.harness/state/edited-files.log" 2>/dev/null)" "internal/cli/doctor.go"
assert_contains "A. code-class message emitted" "$stdout" "Run ./scripts/run-verify.sh"

# ── Case B: Claude Code Edit payload, doc-class path ─────────────────
# The doc-class match requires a leading "/" before "docs/" (see the
# *"/docs/"* case in post_edit_verify.sh), so the path needs a directory
# component ahead of it -- a bare "docs/..." at the payload root would
# fall through to the *.md skip-list branch instead.
dir="case-b"
payload='{"session_id":"t","tool_name":"Edit","tool_input":{"file_path":"repo/docs/plans/active/example.md"}}'
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "B. doc-class file_path payload exits 0" 0 "$actual"
assert_contains "B. doc-class message emitted" "$stdout" "Instruction or documentation files changed"
assert_not_contains "B. no code-class message" "$stdout" "run-verify.sh"

# ── Case C: apply_patch with two files -> both logged ────────────────
dir="case-c"
patch_body="$(printf '*** Begin Patch\n*** Add File: docs/notes/new.md\n+hello\n*** Update File: internal/cli/other.go\n@@\n-old\n+new\n*** End Patch\n')"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "C. apply_patch payload exits 0" 0 "$actual"
log="$(cat "$workdir/$dir/.harness/state/edited-files.log" 2>/dev/null)"
assert_log_lines "C. edited-files.log has two entries" "$workdir/$dir/.harness/state/edited-files.log" 2
assert_contains "C. edited-files.log contains the Add File path" "$log" "docs/notes/new.md"
assert_contains "C. edited-files.log contains the Update File path" "$log" "internal/cli/other.go"

# ── Case D: apply_patch mixing doc-class and code-class -> code wins ─
dir="case-d"
patch_body="$(printf '*** Begin Patch\n*** Update File: repo/docs/plans/active/example.md\n@@\n-old\n+new\n*** Update File: internal/cli/other.go\n@@\n-old\n+new\n*** End Patch\n')"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "D. mixed-class apply_patch payload exits 0" 0 "$actual"
assert_contains "D. code-class message wins over doc-class" "$stdout" "Run ./scripts/run-verify.sh"

# ── Case E: Delete-only apply_patch payload -> no crash ──────────────
dir="case-e"
patch_body="$(printf '*** Begin Patch\n*** Delete File: internal/cli/other.go\n*** End Patch\n')"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "E. Delete-only apply_patch payload exits 0 (no crash)" 0 "$actual"
assert_log_lines "E. deleted path still logged" "$workdir/$dir/.harness/state/edited-files.log" 1
assert_contains "E. edited-files.log contains the Delete File path" "$(cat "$workdir/$dir/.harness/state/edited-files.log" 2>/dev/null)" "internal/cli/other.go"

# ── Case F: no file_path, no patch-shaped command (Bash tool call) ───
dir="case-f"
payload='{"session_id":"t","tool_name":"Bash","tool_input":{"command":"echo hi"}}'
actual=0
stdout="$(run_hook "$dir" "$payload" 2>/dev/null)" || actual=$?
assert_exit "F. Bash tool payload exits 0" 0 "$actual"
if [ -f "$workdir/$dir/.harness/state/edited-files.log" ]; then
  record_fail "F. no edited-files.log entries created"
else
  record_pass "F. no edited-files.log entries created"
fi
if [ -f "$workdir/$dir/.harness/state/needs-verify" ]; then
  record_pass "F. needs-verify still touched"
else
  record_fail "F. needs-verify still touched"
fi
assert_not_contains "F. no reminder message emitted" "$stdout" "hookSpecificOutput"

# ── Summary ─────────────────────────────────────────────────────────
echo
echo "=== post_edit_verify.sh test results ==="
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
