#!/usr/bin/env bash
# test-pre-bash-guard.sh — smoke tests for .claude/hooks/pre_bash_guard.sh
# against the real PreToolUse payload shape.
#
# Regression guard for tech-debt row 100 (docs/tech-debt/README.md): the jq
# path used to call extract_json_field "$payload" "command" (top-level),
# but real Claude Code PreToolUse payloads nest the Bash command under
# .tool_input.command, so every deny/ask rule silently never matched when
# jq was present. The guard only worked via the sed fallback, which matches
# the leaf key anywhere in the payload regardless of nesting.
#
# Cases, each run on BOTH the jq path and the jq-absent (sed fallback) path
# using the same real-shape fixture:
#   A. Denied command (git push --force) nested under tool_input.command
#      -> permissionDecision: deny
#   B. Benign command nested under tool_input.command -> no decision (allow)

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$REPO_ROOT/.claude/hooks/pre_bash_guard.sh"

if [ ! -x "$HOOK" ]; then
  echo "FAIL: hook not found or not executable at $HOOK" >&2
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

# make_payload <command> — real-shape PreToolUse payload: the Bash tool's
# command argument nested under .tool_input.command, matching what Claude
# Code actually sends (not a flat top-level .command).
make_payload() {
  local command="$1"
  printf '{"session_id":"test","tool_name":"Bash","tool_input":{"command":"%s"}}' "$command"
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/pre-bash-guard-test.XXXXXX")"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT

# Minimal PATH without jq, mirroring tests/test-check-mojibake.sh's Case E
# technique: symlink only the tools the hook needs, omitting jq so
# `command -v jq` fails and lib_json.sh falls back to its sed path.
minimal_path="$workdir/no-jq-bin"
mkdir -p "$minimal_path"
for tool in sh bash dash cat grep sed printf dirname env tr command test; do
  resolved="$(command -v "$tool" 2>/dev/null || true)"
  [ -n "$resolved" ] && ln -sf "$resolved" "$minimal_path/$tool" 2>/dev/null || true
done

run_case() {
  local label="$1" command="$2" expect_deny="$3" use_path="$4"
  local out
  out="$(make_payload "$command" | PATH="$use_path" "$HOOK" 2>/dev/null)"
  local saw_deny=no
  case "$out" in
    *'"permissionDecision":"deny"'*) saw_deny=yes ;;
  esac
  if [ "$saw_deny" = "$expect_deny" ]; then
    record_pass "$label"
  else
    record_fail "$label (expected deny=$expect_deny, got deny=$saw_deny, output: $out)"
  fi
}

real_path="$PATH"

# ── A. jq path: denied command, real nested payload -> deny ────────────
run_case "A. jq path: git push --force (nested tool_input.command) denies" \
  "git push --force origin main" yes "$real_path"

# ── B. jq path: benign command, real nested payload -> allow ───────────
run_case "B. jq path: benign command (nested tool_input.command) allows" \
  "echo hello" no "$real_path"

# ── C. sed fallback path (jq absent): denied command -> deny ───────────
run_case "C. no-jq fallback: git push --force (nested tool_input.command) denies" \
  "git push --force origin main" yes "$minimal_path"

# ── D. sed fallback path (jq absent): benign command -> allow ──────────
run_case "D. no-jq fallback: benign command (nested tool_input.command) allows" \
  "echo hello" no "$minimal_path"

echo ""
echo "=== test-pre-bash-guard.sh results ==="
for line in "${results[@]}"; do
  echo "  $line"
done
echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"

if [ "$fail" -gt 0 ]; then
  exit 1
fi
exit 0
