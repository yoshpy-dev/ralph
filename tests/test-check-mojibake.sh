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
#   G. Codex apply_patch payloads (no tool_input.file_path; patch body in
#      tool_input.command) — derives scan targets from "*** Add/Update
#      File:" lines: dirty target → exit 2, clean patch → exit 0,
#      Delete-only patch → exit 0 (no crash, nothing to scan)
#   G4. apply_patch payload with a top-level "cwd" pointing at a
#      subdirectory and a RELATIVE envelope path (the actual shape
#      production sends) → the relative path must resolve against "cwd",
#      not against this hook's own process cwd (C3-H1: after the AR#1
#      fix, .codex/hooks.json cd's to the git root before running this
#      hook, but apply_patch envelope paths remain relative to the
#      Codex SESSION's cwd, which may be a subdirectory of that root)

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

# ── Case G: Codex apply_patch payloads ──────────────────────────────
make_apply_patch_payload() {
  # make_apply_patch_payload <patch-body-with-real-newlines>
  jq -n --arg cmd "$1" '{"session_id":"fixture-session","tool_name":"apply_patch","tool_input":{"command":$cmd}}'
}

apply_patch_dirty="$workdir/apply-patch-dirty.go"
printf 'hello \357\277\275 world\n' > "$apply_patch_dirty"
apply_patch_clean_add="$workdir/apply-patch-add.md"
apply_patch_clean_update="$workdir/apply-patch-clean.go"
printf 'clean content\n' > "$apply_patch_clean_update"

# G1: patch touching one clean file and one dirty file -> exit 2
patch_body="$(printf '*** Begin Patch\n*** Add File: %s\n+hello\n*** Update File: %s\n@@\n-old\n+new\n*** End Patch\n' \
  "$apply_patch_clean_add" "$apply_patch_dirty")"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
printf '%s' "$payload" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "G1. apply_patch with a dirty Update File target exits 2" 2 "$actual"

# G2: patch touching only clean files -> exit 0
patch_body="$(printf '*** Begin Patch\n*** Add File: %s\n+hello\n*** Update File: %s\n@@\n-old\n+new\n*** End Patch\n' \
  "$apply_patch_clean_add" "$apply_patch_clean_update")"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
printf '%s' "$payload" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "G2. Clean apply_patch payload exits 0" 0 "$actual"

# G3: Delete-only patch -> exit 0, no crash (Delete targets are not
# scanned; the deleted file no longer exists to check anyway)
patch_body="$(printf '*** Begin Patch\n*** Delete File: %s\n*** End Patch\n' "$apply_patch_dirty")"
payload="$(make_apply_patch_payload "$patch_body")"
actual=0
printf '%s' "$payload" | HOOK_REPO_ROOT="$REPO_ROOT" "$HOOK" >/dev/null 2>/dev/null || actual=$?
assert_exit "G3. Delete-only apply_patch payload exits 0 (no crash)" 0 "$actual"

# ── Case G4: relative envelope path resolved against payload "cwd" ──
# Simulates the AR#1/AR#2 interaction: the hook's own process cwd is the
# git root (proj_root, matching the cd-first .codex/hooks.json command),
# but the envelope path "dirty.txt" is relative to the Codex SESSION's
# cwd, a subdirectory of that root (proj_subdir). Pre-fix, [ -f "dirty.txt" ]
# is evaluated against proj_root, misses the file, and exits 0 (bug: the
# guard silently stops guarding). Post-fix, "dirty.txt" resolves against
# the payload's "cwd" field to proj_subdir/dirty.txt, which exists.
proj_root="$workdir/proj-root"
proj_subdir="$proj_root/docs"
mkdir -p "$proj_subdir"
subdir_dirty="$proj_subdir/dirty.txt"
printf 'hello \357\277\275 world\n' > "$subdir_dirty"
patch_body="$(printf '*** Begin Patch\n*** Update File: dirty.txt\n@@\n-old\n+new\n*** End Patch\n')"
payload="$(jq -n --arg cmd "$patch_body" --arg cwd "$proj_subdir" \
  '{"session_id":"fixture-session","tool_name":"apply_patch","cwd":$cwd,"tool_input":{"command":$cmd}}')"
actual=0
( cd "$proj_root" && printf '%s' "$payload" | HOOK_REPO_ROOT="$proj_root" "$HOOK" >/dev/null 2>/dev/null ) || actual=$?
assert_exit "G4. relative envelope path resolves against payload cwd (subdir launch) → exit 2" 2 "$actual"

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
