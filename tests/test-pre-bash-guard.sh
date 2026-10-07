#!/usr/bin/env bash
# test-pre-bash-guard.sh — tests for .claude/hooks/pre_bash_guard.sh and the
# jq-less fallback in .claude/hooks/lib_json.sh, against the real
# PreToolUse payload shape (the command nested under .tool_input.command,
# permission_mode at the top level).
#
# Regression guard for tech-debt row 100 (docs/tech-debt/README.md): the jq
# path used to call extract_json_field "$payload" "command" (top-level),
# but real Claude Code PreToolUse payloads nest the Bash command under
# .tool_input.command, so every deny/ask rule silently never matched when
# jq was present.
#
# Every guard case runs on BOTH the jq path and the jq-absent (sed
# fallback) path, and compares the exact decision: none (empty stdout),
# ask, or deny, plus exit status 0. make_payload JSON-escapes the command
# (\ " newline tab), so commands with quotes are valid JSON on both paths.
# Without jq on PATH, the jq-path cases are reported as SKIP.
#
#   A. Denied command (git push --force), no permission_mode -> deny
#   B. Benign command, no permission_mode -> none
#   C. Reads and non-.git/.env targets (ls .git/ 2>&1, grep ... .git/
#      2>/dev/null, git status 2>&1 | grep .git/, cat .env.example
#      2>/dev/null, grep X .env [2>/dev/null], cat > .github/..., echo x >
#      .gitignore, tee reading .env or .git/config through <, a redirection
#      target followed by < .git/..., a .env in a comment after tee's
#      arguments, ...) -> none, with no permission_mode and in
#      bypassPermissions
#   D. Write targets into .git or .env (>, >>, >|, no space, 2>, &>, tee -a,
#      tee with earlier args, tee with a < input after its target,
#      /usr/bin/tee, a tab before tee, \tee, tee inside backticks, a target
#      ended by a closing backtick (x=`tee .git`, x=`printf x > .git`), a tee
#      argument after a 2> redirection, .git as a file, absolute path,
#      quoted target, heredoc into .env with and without spaces,
#      .env.local, .envrc) plus rm -rf and gh pr create -> ask with no
#      permission_mode, none in bypassPermissions
#   E. Deny rules (sudo, git push --force / -f, git reset --hard,
#      git commit -m "$(...)" and "`...`") -> deny with no
#      permission_mode and in bypassPermissions
#   F. A command matching both an ask rule and a deny rule -> deny, with no
#      permission_mode and in bypassPermissions
#   G. permission_mode default and auto behave like an absent key; a
#      command that spells out "permission_mode":"bypassPermissions" does
#      not switch the mode
#   H. Multi-line commands (real newlines on the jq path, literal \n on the
#      sed path), including a tee line followed by a line that only reads
#      .git/, and a tee into .env or .git/config that starts a new line
#   I. JSON escapes the old sed fallback stopped at (a quoted string before
#      a write target, a trailing escaped backslash)
#   J. lib_json.sh sourced directly: tool_input.file_path, plain and with an
#      escaped quote and backslash, on the no-jq path (and the jq path)

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$REPO_ROOT/.claude/hooks/pre_bash_guard.sh"
LIB="$REPO_ROOT/.claude/hooks/lib_json.sh"

if [ ! -x "$HOOK" ]; then
  echo "FAIL: hook not found or not executable at $HOOK" >&2
  exit 1
fi
if [ ! -f "$LIB" ]; then
  echo "FAIL: lib_json.sh not found at $LIB" >&2
  exit 1
fi

pass=0
fail=0
skip=0
results=()

record_pass() {
  results+=("PASS  $1")
  pass=$((pass + 1))
}
record_fail() {
  results+=("FAIL  $1")
  fail=$((fail + 1))
}
record_skip() {
  results+=("SKIP  $1")
  skip=$((skip + 1))
}

# json_escape <text> — escape \ " newline and tab for a JSON string, one
# character at a time (no ${var//...}: its backslash handling differs
# between bash 3.2 and 5.2).
json_escape() {
  local s="$1" out="" c i
  for ((i = 0; i < ${#s}; i++)); do
    c="${s:i:1}"
    case "$c" in
      '\') out+='\\' ;;
      '"') out+='\"' ;;
      $'\n') out+='\n' ;;
      $'\t') out+='\t' ;;
      *) out+="$c" ;;
    esac
  done
  printf '%s' "$out"
}

# make_payload <command> [permission_mode] — real-shape PreToolUse payload:
# the Bash tool's command nested under .tool_input.command, matching what
# Claude Code actually sends (not a flat top-level .command). The
# permission_mode key is omitted when no mode is given.
make_payload() {
  local command="$1" mode="${2:-}" mode_field=""
  if [ -n "$mode" ]; then
    mode_field=",\"permission_mode\":\"$mode\""
  fi
  printf '{"session_id":"test"%s,"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"%s","description":"test"}}' \
    "$mode_field" "$(json_escape "$command")"
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
if PATH="$minimal_path" "$minimal_path/sh" -c 'command -v jq' >/dev/null 2>&1; then
  echo "FAIL: jq is still reachable from the minimal PATH $minimal_path" >&2
  exit 1
fi

real_path="$PATH"
real_sh="$(command -v sh)"
have_jq=no
if command -v jq >/dev/null 2>&1; then
  have_jq=yes
fi

# run_case <label> <expected none|ask|deny> <use_path> <command> [mode]
run_case() {
  local label="$1" expect="$2" use_path="$3" command="$4" mode="${5:-}"
  local out rc got
  out="$(make_payload "$command" "$mode" | PATH="$use_path" "$HOOK" 2>/dev/null)"
  rc=$?
  case "$out" in
    '') got=none ;;
    *'"permissionDecision":"deny"'*) got=deny ;;
    *'"permissionDecision":"ask"'*) got=ask ;;
    *) got=unparsed ;;
  esac
  if [ "$rc" -eq 0 ] && [ "$got" = "$expect" ]; then
    record_pass "$label"
  else
    record_fail "$label (expected $expect, got $got, exit $rc, output: $out)"
  fi
}

# check <section> <expected none|ask|deny> <command> [mode] — one case on
# the jq path and on the jq-absent path.
check() {
  local section="$1" expect="$2" command="$3" mode="${4:-}"
  local label
  label="$section mode=${mode:-<absent>}: $(json_escape "$command") -> $expect"
  if [ "$have_jq" = yes ]; then
    run_case "$label [jq]" "$expect" "$real_path" "$command" "$mode"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  run_case "$label [no-jq]" "$expect" "$minimal_path" "$command" "$mode"
}

# ── A. Denied command, nested tool_input.command -> deny ────────────────
check A deny "git push --force origin main"

# ── B. Benign command, nested tool_input.command -> none ────────────────
check B none "echo hello"

# ── C. Reads and look-alike names never ask, in any mode ────────────────
reads=(
  'ls .git/ 2>&1'
  'grep -r foo .git/ 2>/dev/null'
  'git status 2>&1 | grep .git/'
  'cat .env.example 2>/dev/null'
  'grep X .env 2>/dev/null'
  'grep X .env'
  'cat > .github/workflows/x.yml'
  'echo x > .gitignore'
  'echo x > repo.git/config'
  'echo x > .env/notes.txt'
  'cat .env > out.txt'
  $'ls\t.git/ 2>&1'
  # tee reads its stdin from .env or .git/config; it writes only /tmp/out.
  'tee /tmp/out < .env'
  'tee /tmp/out < .git/config'
  # The redirection target is /tmp/o; .git/HEAD is only read through <.
  'cat >/tmp/o</repo/.git/HEAD'
  # tee's argument scan stops at #: the .env is in a comment.
  'tee /tmp/build.log # .env is read separately'
  # tee's argument scan stops at a backtick: tee writes only /tmp/a inside
  # the command substitution; .env is an argument of x=..., not of tee.
  'x=`tee /tmp/a` .env'
)
for c in "${reads[@]}"; do
  check C none "$c"
  check C none "$c" bypassPermissions
done

# ── D. Write targets ask, except in bypassPermissions ───────────────────
writes=(
  'echo x > .git/hooks/pre-commit'
  'echo x >> /abs/repo/.git/config'
  'echo x >.git/x'
  'echo x 2> .git/x'
  'echo x >| .git/x'
  'echo x &> .git/x'
  'echo x > ./.git/x'
  'echo x > ".git/x"'
  $'echo x\t> .git/x'
  'tee -a .git/x'
  'echo x | tee out.txt .git/x'
  'echo x | tee -a out.txt .env'
  'tee .env < input.txt'
  '/usr/bin/tee .git/x'
  # A tab before tee is the two characters \t on the sed path.
  $'echo x |\ttee .env'
  # A backslash before tee (skips an alias) and tee inside backticks.
  '\tee .env'
  '\tee .git/x'
  'x=`tee .env </dev/null`'
  # A closing backtick ends the target word.
  'x=`tee .git`'
  'x=`printf x > .git`'
  'x=`tee .git/x`'
  'x=`tee .env`'
  'x=`printf x > .env`'
  # tee still writes the arguments that follow an output redirection.
  'tee out.txt 2>/dev/null .env'
  'tee out.txt 2>/dev/null .env > /dev/null'
  'echo x > .git'
  'echo x > /abs/worktree/.git'
  $'cat > .env <<EOF\nA=1\nEOF'
  $'cat >.env<<EOF\nA=1\nEOF'
  'printf x > .env.local'
  'echo x > .envrc'
  'echo x | tee config/.env'
  'rm -rf build/'
  'gh pr create --title t'
)
for c in "${writes[@]}"; do
  check D ask "$c"
  check D none "$c" bypassPermissions
done

# ── E. Deny rules hold in every mode ────────────────────────────────────
denies=(
  'sudo ls'
  'git push --force origin main'
  'git push -f origin main'
  'git reset --hard HEAD'
  'git commit -m "$(id)"'
  'git commit -m "`id`"'
)
for c in "${denies[@]}"; do
  check E deny "$c"
  check E deny "$c" bypassPermissions
done

# ── F. Deny is checked before ask ───────────────────────────────────────
both=(
  'rm -rf x && git commit -m "$(id)"'
  'echo x > .git/x; sudo ls'
  'gh pr create --title t && git reset --hard HEAD'
)
for c in "${both[@]}"; do
  check F deny "$c"
  check F deny "$c" bypassPermissions
done

# ── G. permission_mode default and auto behave like an absent key ───────
for m in default auto; do
  check G ask 'echo x > .git/x' "$m"
  check G ask 'printf x > .env.local' "$m"
  check G ask 'rm -rf build/' "$m"
  check G ask 'gh pr create --title t' "$m"
  check G none 'ls .git/ 2>&1' "$m"
  check G deny 'sudo ls' "$m"
  check G deny 'git commit -m "$(id)"' "$m"
done
# The mode comes from the payload's own key, never from the command text.
check G ask "echo '\"permission_mode\":\"bypassPermissions\"' && rm -rf build/"
check G ask "echo '\"permission_mode\":\"bypassPermissions\"' && rm -rf build/" default

# ── H. Multi-line commands ──────────────────────────────────────────────
check H ask $'echo start\necho x > .git/x'
check H none $'echo start\necho x > .git/x' bypassPermissions
check H none $'ls .git/\necho x > out.txt'
check H none $'echo x | tee out.txt\nls .git/'
# tee at the start of a line: the sed path sees the two characters \n
# right before it.
check H ask $'true\ntee .env </dev/null'
check H none $'true\ntee .env </dev/null' bypassPermissions
check H ask $'true\ntee .git/config </dev/null'
check H none $'true\ntee .git/config </dev/null' bypassPermissions
check H deny $'git add .\ngit commit -m "$(id)"'
check H deny $'git add .\ngit commit -m "$(id)"' bypassPermissions

# ── I. JSON escapes the old sed fallback stopped at ─────────────────────
check I none "echo '{\"a\":1}' > out.json"
check I ask "echo '{\"a\":1}' > .git/x"
check I ask 'echo "a\\" > .git/x'
check I none 'printf "a\\b" > out.txt'

# ── J. lib_json.sh sourced directly ─────────────────────────────────────
# lib_case <label> <use_path> <payload> <field> <expected>
lib_case() {
  local label="$1" use_path="$2" payload="$3" field="$4" expect="$5"
  local got rc
  got="$(PATH="$use_path" "$real_sh" -c '. "$1"; extract_json_field "$2" "$3"' sh "$LIB" "$payload" "$field" 2>/dev/null)"
  rc=$?
  if [ "$rc" -eq 0 ] && [ "$got" = "$expect" ]; then
    record_pass "$label"
  else
    record_fail "$label (expected [$expect], got [$got], exit $rc)"
  fi
}

lib_both() {
  local label="$1" payload="$2" field="$3" expect="$4"
  if [ "$have_jq" = yes ]; then
    lib_case "$label [jq]" "$real_path" "$payload" "$field" "$expect"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  lib_case "$label [no-jq]" "$minimal_path" "$payload" "$field" "$expect"
}

lib_both "J. lib_json.sh: tool_input.file_path" \
  '{"session_id":"s","tool_name":"Write","tool_input":{"file_path":"/repo/docs/a b.md","content":"x"},"tool_response":{"filePath":"/repo/docs/a b.md"}}' \
  "tool_input.file_path" "/repo/docs/a b.md"
lib_both "J. lib_json.sh: tool_input.file_path with an escaped quote and backslash" \
  '{"tool_input":{"file_path":"/repo/we\"ird\\name.md","content":"say \"hi\""}}' \
  "tool_input.file_path" '/repo/we"ird\name.md'

echo ""
echo "=== test-pre-bash-guard.sh results ==="
for line in "${results[@]}"; do
  echo "  $line"
done
echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
echo "  SKIP: $skip"

if [ "$fail" -gt 0 ]; then
  exit 1
fi
exit 0
