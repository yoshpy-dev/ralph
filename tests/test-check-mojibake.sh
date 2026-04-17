#!/usr/bin/env bash
# test-check-mojibake.sh — smoke tests for .claude/hooks/check_mojibake.sh.
#
# Cases (see plan 2026-04-17-mojibake-postedit-guard.md):
#   A. U+FFFD-containing file → exit 2
#   B. Clean UTF-8 file → exit 0
#   C. Non-existent file_path → exit 0
#   D. Allowlisted path with U+FFFD → exit 0
#   E. jq missing → exit 0 and marker file created
#   F. Edit/Write/MultiEdit payload fixtures extract file_path correctly

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$REPO_ROOT/.claude/hooks/check_mojibake.sh"
FIXTURES="$REPO_ROOT/tests/fixtures/payloads"

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

assert_exit() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    record_pass "$label (exit $actual)"
  else
    record_fail "$label (expected $expected, got $actual)"
  fi
}

make_payload() {
  local file_path="$1"
  printf '{"session_id":"test","tool_name":"Edit","tool_input":{"file_path":"%s"}}' "$file_path"
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/mojibake-test.XXXXXX")"
# Scope all test state under $workdir so we do not stomp on the real
# session's .harness/state/ markers. The Case E marker lives under
# $alt_root (which is inside $workdir) and is removed with it.
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT

# ── Case A: U+FFFD present, not allowlisted ─────────────────────────
dirty="$workdir/dirty.txt"
printf 'hello \357\277\275 world\n' > "$dirty"
actual=0
make_payload "$dirty" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "A. U+FFFD triggers exit 2" 2 "$actual"

# ── Case B: clean UTF-8 Japanese ────────────────────────────────────
clean="$workdir/clean.txt"
printf 'こんにちは、世界\n' > "$clean"
actual=0
make_payload "$clean" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "B. Clean UTF-8 exits 0" 0 "$actual"

# ── Case C: non-existent file_path ──────────────────────────────────
actual=0
make_payload "$workdir/does-not-exist.txt" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "C. Missing path exits 0" 0 "$actual"

# ── Case D: allowlist match bypasses detection ──────────────────────
alt_root="$workdir/alt-root"
mkdir -p "$alt_root/.claude/hooks" "$alt_root/tests/fixtures"
printf 'tests/fixtures/**\n' > "$alt_root/.claude/hooks/mojibake-allowlist"
allowed="$alt_root/tests/fixtures/dirty.txt"
printf 'dirty \357\277\275 fixture\n' > "$allowed"
actual=0
make_payload "$allowed" | HOOK_REPO_ROOT="$alt_root" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "D. Allowlisted U+FFFD exits 0" 0 "$actual"

# ── Case E: jq missing → exit 0 + marker ────────────────────────────
mkdir -p "$alt_root/.harness/state"
rm -f "$alt_root/.harness/state/mojibake-jq-missing"
# Restrict PATH so jq cannot be resolved. Link only the essentials the
# hook needs during startup (dirname/pwd/cd) plus a few helpers used
# inside the jq-missing branch (mkdir for marker). We do NOT link jq.
minimal_path="$workdir/no-jq-bin"
mkdir -p "$minimal_path"
for tool in sh bash dash cat grep sed mkdir rm cd command pwd printf dirname env ln test; do
  resolved="$(command -v "$tool" 2>/dev/null || true)"
  [ -n "$resolved" ] && ln -sf "$resolved" "$minimal_path/$tool" 2>/dev/null || true
done
actual=0
PATH="$minimal_path" make_payload "$dirty" | PATH="$minimal_path" HOOK_REPO_ROOT="$alt_root" "$HOOK" >/dev/null 2>/dev/null || actual=$?
if [ "$actual" -eq 0 ] && [ -f "$alt_root/.harness/state/mojibake-jq-missing" ]; then
  record_pass "E. jq missing → exit 0 + marker"
else
  marker_present=no
  [ -f "$alt_root/.harness/state/mojibake-jq-missing" ] && marker_present=yes
  record_fail "E. jq missing (exit=$actual, marker=$marker_present)"
fi

# ── Case F: Edit/Write/MultiEdit payload fixtures ───────────────────
for tool in edit write multiedit; do
  fixture="$FIXTURES/$tool.json"
  if [ ! -f "$fixture" ]; then
    record_fail "F.$tool fixture missing at $fixture"
    continue
  fi
  clean_fx="$workdir/$tool-clean.txt"
  printf 'fixture %s ok\n' "$tool" > "$clean_fx"
  payload="$(sed "s#__FILE_PATH__#$clean_fx#" "$fixture")"
  actual=0
  printf '%s' "$payload" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
  assert_exit "F.$tool clean payload → exit 0" 0 "$actual"

  dirty_fx="$workdir/$tool-dirty.txt"
  printf 'fixture \357\277\275 %s\n' "$tool" > "$dirty_fx"
  payload="$(sed "s#__FILE_PATH__#$dirty_fx#" "$fixture")"
  actual=0
  printf '%s' "$payload" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
  assert_exit "F.$tool dirty payload → exit 2" 2 "$actual"
done

# ── Summary ─────────────────────────────────────────────────────────
echo
echo "=== check_mojibake.sh test results ==="
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
