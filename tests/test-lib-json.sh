#!/usr/bin/env bash
# test-lib-json.sh — tests for extract_json_field in .claude/hooks/lib_json.sh
# on the jq-absent path, where sed reads the raw string value and an awk
# program decodes its JSON escapes.
#
# Each case gives a payload, a field path, and the expected value. The
# jq-absent result is compared byte for byte with the expected value plus
# one newline (what jq -r prints), so a gained or lost trailing newline
# fails too. When jq decodes the payload the same way, the jq path is held
# to the same bytes; without jq on PATH those checks are reported as SKIP.
# The jq-absent cases run under sh and, when it is installed, dash.
#
# Payloads spell the JSON escape backslash-u as %u (filled in by uesc):
# editing tools that pass file text through JSON decode a backslash-u
# escape of a printable character in transit, which would silently turn
# these cases into plain text.
#
#   A. A real newline (\n in the JSON) and a backslash followed by n (\\n in
#      the JSON) come back different, each equal to what jq returns
#   B. Simple escapes: tab, \" \\ \/ \r \b \f, a value ending in an escaped
#      backslash, an escaped backslash before an escaped quote
#   C. %u escapes in the ASCII range: upper- and lower-case hex, %u007f, a
#      %u escape that decodes to a backslash followed by n (decoded once,
#      not twice), NUL (compared through od)
#   D. Kept as written: e acute (%u00e9), a surrogate pair, and, on invalid
#      JSON (no jq comparison), an unknown escape and an incomplete or
#      non-hex %u
#   E. Field paths: nested tool_input.command, top-level permission_mode,
#      tool_input.file_path with an ordinary path, a missing key (no
#      output), an empty string (a newline only)
#   F. Escapes that straddle the awk program's 512-character window, and a
#      JSON string of about 260 KB (180 KB once decoded)

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LIB="$REPO_ROOT/.claude/hooks/lib_json.sh"

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

workdir="$(mktemp -d "${TMPDIR:-/tmp}/lib-json-test.XXXXXX")"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT

# Minimal PATH without jq (the technique of tests/test-pre-bash-guard.sh,
# plus awk): symlink only the tools lib_json.sh needs, omitting jq so
# `command -v jq` fails and extract_json_field takes its sed + awk path.
minimal_path="$workdir/no-jq-bin"
mkdir -p "$minimal_path"
for tool in sh bash dash cat grep sed printf dirname env tr command test awk; do
  resolved="$(command -v "$tool" 2>/dev/null || true)"
  [ -n "$resolved" ] && ln -sf "$resolved" "$minimal_path/$tool" 2>/dev/null || true
done
if PATH="$minimal_path" "$minimal_path/sh" -c 'command -v jq' >/dev/null 2>&1; then
  echo "FAIL: jq is still reachable from the minimal PATH $minimal_path" >&2
  exit 1
fi
if ! PATH="$minimal_path" "$minimal_path/sh" -c 'command -v awk' >/dev/null 2>&1; then
  echo "FAIL: awk is not reachable from the minimal PATH $minimal_path" >&2
  exit 1
fi

real_path="$PATH"
have_jq=no
if command -v jq >/dev/null 2>&1; then
  have_jq=yes
fi
nojq_shells=(sh)
if [ -x "$minimal_path/dash" ]; then
  nojq_shells+=(dash)
fi

# uesc <text> — turn each %u into a JSON backslash-u.
uesc() {
  printf '%s' "$1" | sed 's/%u/\\u/g'
}

# show <text> — the text with control characters made visible, for
# failure messages; text over 120 bytes is cut and its length given.
show() {
  if [ "${#1}" -gt 120 ]; then
    printf '%q... (%d bytes)' "${1:0:120}" "${#1}"
  else
    printf '%q' "$1"
  fi
}

# extract_raw <use_path> <shell> <payload> <field> — sets got_raw to the
# function's stdout byte for byte (the trailing "." keeps trailing
# newlines through the command substitution) and got_rc to its exit
# status. The payload goes in on stdin, as the hooks receive it: Linux
# caps one argument at 128 KB, so the long case in F would not fit in
# argv.
extract_raw() {
  got_raw="$(printf '%s' "$3" | PATH="$1" "$2" -c '. "$1"; payload="$(cat)"; extract_json_field "$payload" "$2"; rc=$?; printf .; exit "$rc"' sh "$LIB" "$4" 2>/dev/null)"
  got_rc=$?
  got_raw="${got_raw%.}"
}

# check_raw <label> <kind> <payload> <field> <want_raw>
#   kind same    — valid JSON that jq decodes to the same bytes
#   kind kept    — valid JSON; jq decodes what the fallback keeps as written,
#                  so only the payload's validity is checked with jq
#   kind invalid — not JSON (jq is not consulted)
# The payload and want_raw go through uesc first.
check_raw() {
  local label="$1" kind="$2" field="$4" payload want_raw sh_name
  payload="$(uesc "$3")"
  want_raw="$(uesc "$5"; printf .)"
  want_raw="${want_raw%.}"
  for sh_name in "${nojq_shells[@]}"; do
    extract_raw "$minimal_path" "$minimal_path/$sh_name" "$payload" "$field"
    if [ "$got_rc" -eq 0 ] && [ "$got_raw" = "$want_raw" ]; then
      record_pass "$label [no-jq $sh_name]"
    else
      record_fail "$label [no-jq $sh_name] (expected $(show "$want_raw"), got $(show "$got_raw"), exit $got_rc)"
    fi
  done
  case "$kind" in
    invalid) return 0 ;;
  esac
  if [ "$have_jq" != yes ]; then
    record_skip "$label [jq] (jq not on PATH)"
    return 0
  fi
  if [ "$kind" = kept ]; then
    if printf '%s' "$payload" | jq -e . >/dev/null 2>&1; then
      record_pass "$label [jq: payload is valid JSON]"
    else
      record_fail "$label [jq: payload is not valid JSON]"
    fi
    return 0
  fi
  extract_raw "$real_path" sh "$payload" "$field"
  if [ "$got_rc" -eq 0 ] && [ "$got_raw" = "$want_raw" ]; then
    record_pass "$label [jq]"
  else
    record_fail "$label [jq] (expected $(show "$want_raw"), got $(show "$got_raw"), exit $got_rc)"
  fi
}

# check <label> <kind> <payload> <field> <value> — check_raw with the value
# plus the newline jq -r prints after it.
check() {
  check_raw "$1" "$2" "$3" "$4" "$5"$'\n'
}

# cmd <json-escaped command> — a PreToolUse-shaped payload.
cmd() {
  printf '{"session_id":"s","permission_mode":"default","tool_name":"Bash","tool_input":{"command":"%s","description":"d"}}' "$1"
}

# ── A. Real newline vs backslash + n ────────────────────────────────────
newline_payload="$(cmd 'git add .\ngit commit -m x')"
backslash_n_payload="$(cmd 'printf a\\ngit commit -m x')"
check "A. real newline" same "$newline_payload" tool_input.command \
  $'git add .\ngit commit -m x'
check "A. backslash + n" same "$backslash_n_payload" tool_input.command \
  'printf a\ngit commit -m x'
same_text_newline="$(cmd 'a\nb')"
same_text_backslash_n="$(cmd 'a\\nb')"
for sh_name in "${nojq_shells[@]}"; do
  extract_raw "$minimal_path" "$minimal_path/$sh_name" "$same_text_newline" tool_input.command
  out_newline="$got_raw"
  extract_raw "$minimal_path" "$minimal_path/$sh_name" "$same_text_backslash_n" tool_input.command
  out_backslash_n="$got_raw"
  if [ "$out_newline" != "$out_backslash_n" ]; then
    record_pass "A. a\\nb and a\\\\nb in the JSON differ [no-jq $sh_name]"
  else
    record_fail "A. a\\nb and a\\\\nb in the JSON differ [no-jq $sh_name] (both $(show "$out_newline"))"
  fi
done
check "A. a\\nb in the JSON" same "$same_text_newline" tool_input.command $'a\nb'
check "A. a\\\\nb in the JSON" same "$same_text_backslash_n" tool_input.command 'a\nb'

# ── B. Simple escapes ───────────────────────────────────────────────────
check "B. tab" same "$(cmd 'a\tb')" tool_input.command $'a\tb'
check "B. escaped quote" same "$(cmd 'say \"hi\"')" tool_input.command 'say "hi"'
check "B. escaped backslash" same "$(cmd 'C:\\dir\\x')" tool_input.command 'C:\dir\x'
check "B. escaped slash" same "$(cmd 'a\/b')" tool_input.command 'a/b'
check "B. carriage return" same "$(cmd 'a\rb')" tool_input.command $'a\rb'
check "B. backspace and form feed" same "$(cmd 'a\bb\fc')" tool_input.command $'a\bb\fc'
check "B. ends in an escaped backslash" same "$(cmd 'echo a\\')" tool_input.command 'echo a\'
check "B. escaped backslash before an escaped quote" same "$(cmd 'x\\\"y')" tool_input.command 'x\"y'
check "B. three escaped backslashes then n" same "$(cmd '\\\\\\n')" tool_input.command '\\\n'

# ── C. ASCII-range %u escapes ───────────────────────────────────────────
check "C. %u0041 (A)" same "$(cmd 'x%u0041y')" tool_input.command 'xAy'
check "C. %u006a and %u006A (j, either case)" same "$(cmd '%u006a%u006A')" tool_input.command 'jj'
check "C. %u007f (DEL)" same "$(cmd 'a%u007fb')" tool_input.command $'a\x7fb'
check "C. %u007F (DEL, upper case)" same "$(cmd 'a%u007Fb')" tool_input.command $'a\x7fb'
check "C. %u0022 and %u0027 (quotes)" same "$(cmd '%u0022%u0027')" tool_input.command "\"'"
check "C. %u000a (newline)" same "$(cmd 'a%u000ab')" tool_input.command $'a\nb'
check "C. %u005cn decodes once (backslash, n)" same "$(cmd '%u005cn')" tool_input.command '\n'
nul_payload="$(uesc "$(cmd 'a%u0000b')")"
for sh_name in "${nojq_shells[@]}"; do
  got_hex="$(printf '%s' "$nul_payload" | PATH="$minimal_path" "$minimal_path/$sh_name" -c '. "$1"; payload="$(cat)"; extract_json_field "$payload" "$2"' sh "$LIB" tool_input.command 2>/dev/null | od -An -tx1 | tr -d ' \n')"
  if [ "$got_hex" = "6100620a" ]; then
    record_pass "C. %u0000 (NUL byte) [no-jq $sh_name]"
  else
    record_fail "C. %u0000 (NUL byte) [no-jq $sh_name] (expected bytes 6100620a, got $got_hex)"
  fi
done
if [ "$have_jq" = yes ]; then
  got_hex="$(printf '%s' "$nul_payload" | jq -r '.tool_input.command // empty' 2>/dev/null | od -An -tx1 | tr -d ' \n')"
  if [ "$got_hex" = "6100620a" ]; then
    record_pass "C. %u0000 (NUL byte) [jq]"
  else
    record_fail "C. %u0000 (NUL byte) [jq] (expected bytes 6100620a, got $got_hex)"
  fi
else
  record_skip "C. %u0000 (NUL byte) [jq] (jq not on PATH)"
fi

# ── D. Kept as written ──────────────────────────────────────────────────
check "D. %u00e9 (e acute) kept" kept "$(cmd 'caf%u00e9')" tool_input.command 'caf%u00e9'
check "D. surrogate pair kept" kept "$(cmd 'x%ud83d%ude00y')" tool_input.command 'x%ud83d%ude00y'
check "D. %u0080 kept" kept "$(cmd 'a%u0080b')" tool_input.command 'a%u0080b'
check "D. unknown escape kept" invalid "$(cmd 'a\xb\qc')" tool_input.command 'a\xb\qc'
check "D. non-hex %u kept" invalid "$(cmd 'a%u12G4b')" tool_input.command 'a%u12G4b'
check "D. incomplete %u at the end kept" invalid "$(cmd 'a%u00')" tool_input.command 'a%u00'

# ── E. Field paths ──────────────────────────────────────────────────────
mixed='{"permission_mode":"bypassPermissions","tool_name":"Bash","tool_input":{"command":"git status","description":"d"}}'
check "E. nested tool_input.command" same "$mixed" tool_input.command 'git status'
check "E. top-level permission_mode" same "$mixed" permission_mode 'bypassPermissions'
check "E. tool_input.file_path, ordinary path" same \
  '{"session_id":"s","tool_name":"Write","tool_input":{"file_path":"/repo/docs/a b.md","content":"x\ny"},"tool_response":{"filePath":"/repo/docs/a b.md"}}' \
  tool_input.file_path '/repo/docs/a b.md'
check_raw "E. missing key prints nothing" same "$mixed" cwd ''
check "E. empty string prints a newline only" same '{"tool_input":{"command":""}}' tool_input.command ''

# ── F. Window boundary and a long value ─────────────────────────────────
bs='\'
dq='"'
for n in 505 506 507 508 509 510 511 512 513; do
  pad="$(printf '%*s' "$n" '' | tr ' ' x)"
  check "F. %u0041 then an escaped backslash and quote after $n characters" same \
    "$(cmd "${pad}%u0041${bs}${bs}${bs}${dq}y${bs}n")" tool_input.command \
    "${pad}A${bs}${dq}y"$'\n'
done
unit_json='ab\"c\\d\ne\tf%u0041g '
unit_value=$'ab"c\\d\ne\tfAg '
long_json=""
long_value=""
for ((k = 0; k < 12000; k++)); do
  long_json+="$unit_json"
  long_value+="$unit_value"
done
check "F. a ${#long_json}-byte JSON string with escapes throughout" same \
  "$(cmd "$long_json")" tool_input.command "$long_value"

# G. Neither jq nor awk: the sed-only fallback still returns the command,
# with \" and \\ decoded and \n kept as two characters (so a caller such as
# the Bash guard gets something to judge instead of nothing).
noawk_path="$workdir/no-jq-no-awk-bin"
mkdir -p "$noawk_path"
for tool in sh bash dash cat grep sed printf dirname env tr command test; do
  resolved="$(command -v "$tool" 2>/dev/null || true)"
  [ -n "$resolved" ] && ln -sf "$resolved" "$noawk_path/$tool" 2>/dev/null || true
done
if PATH="$noawk_path" "$noawk_path/sh" -c 'command -v awk || command -v jq' >/dev/null 2>&1; then
  record_fail "G. awk or jq is still reachable from $noawk_path"
else
  noawk_payload='{"tool_input":{"command":"git commit -m \"x\" && echo a\nb \\\\ c"}}'
  extract_raw "$noawk_path" "$noawk_path/sh" "$noawk_payload" tool_input.command
  noawk_want="$(printf '%s\n.' 'git commit -m "x" && echo a\nb \\ c')"
  noawk_want="${noawk_want%.}"
  if [ "$got_rc" -eq 0 ] && [ "$got_raw" = "$noawk_want" ]; then
    record_pass "G. no jq and no awk: sed-only fallback decodes \\\" and \\\\, keeps \\n"
  else
    record_fail "G. no jq and no awk: sed-only fallback (expected $(show "$noawk_want"), got $(show "$got_raw"), exit $got_rc)"
  fi
fi

echo ""
echo "=== test-lib-json.sh results ==="
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
