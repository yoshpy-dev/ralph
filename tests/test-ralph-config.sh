#!/usr/bin/env sh
# shellcheck disable=SC1090  # the whole point is sourcing $CONFIG dynamically
set -eu

# test-ralph-config.sh — tests for scripts/ralph-config.sh, whose
# ralph_setting <key> reads one standard-flow setting with
# `ralph config get <key>` and falls back to its table of defaults
# (ralph_setting_default) by the exit code.
#
# A stand-in ralph in a temporary directory exits with FAKE_RALPH_EXIT,
# prints FAKE_RALPH_STDOUT and a newline on stdout and FAKE_RALPH_STDERR on
# stderr when they are set, and logs its directory and arguments. Each case
# sources the helper in a fresh shell, under /bin/sh and each of bash, dash,
# and zsh (with -f) found on PATH, and runs ralph_setting from a work
# directory. The PATH of a case is the stand-in's directory (left out for
# ralph not on PATH), a stand-in mktemp's, and /usr/bin:/bin, so the ralph a
# developer installed (such as /opt/homebrew/bin/ralph) is never reached.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG="${PROJECT_ROOT}/scripts/ralph-config.sh"

# The keys of the helper's table; their values are compared with
# config.Default() in internal/config/defaults_sync_test.go.
KEYS="standard.max_pipeline_cycles reviewer.claude_model reviewer.claude_effort reviewer.codex_model reviewer.codex_effort"

# The environment variables earlier versions of the helper read, each set to
# a value no default uses.
RETIRED_ENV="RALPH_STANDARD_MAX_PIPELINE_CYCLES=7 RALPH_CLAUDE_REVIEWER_MODEL=env-claude RALPH_CODEX_REVIEWER_MODEL=env-codex RALPH_CODEX_REASONING_EFFORT=env-effort"

_pass=0
_fail=0
_total=0

pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_eq() {
  if [ "$2" = "$3" ]; then
    pass "$1"
  else
    fail "$1"
    printf '    expected: %s\n' "$2"
    printf '    actual:   %s\n' "$3"
  fi
}

assert_contains() {
  case "$3" in
    *"$2"*) pass "$1" ;;
    *)
      fail "$1"
      printf '    want it to contain: %s\n' "$2"
      printf '    actual:             %s\n' "$3"
      ;;
  esac
}

# assert_one_line <desc> <text> — <text> is one non-empty line.
assert_one_line() {
  if [ -n "$2" ] && [ "$(printf '%s\n' "$2" | wc -l | tr -d ' ')" = 1 ]; then
    pass "$1"
  else
    fail "$1"
    printf '    actual: %s\n' "$2"
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/bin" "$tmp/tools" "$tmp/work" "$tmp/made"
: > "$tmp/made.log"
cat > "$tmp/bin/ralph" <<'EOF'
#!/bin/sh
printf '%s|%s\n' "$(pwd -P)" "$*" >> "$FAKE_RALPH_LOG"
if [ -n "${FAKE_RALPH_STDOUT+set}" ]; then printf '%s\n' "$FAKE_RALPH_STDOUT"; fi
if [ -n "${FAKE_RALPH_STDERR+set}" ]; then printf '%s\n' "$FAKE_RALPH_STDERR" >&2; fi
exit "${FAKE_RALPH_EXIT:-0}"
EOF
# A stand-in mktemp makes its directory in $tmp/made and logs it, so the
# test sees whether ralph_setting removes what it makes (macOS's mktemp -d
# ignores TMPDIR).
cat > "$tmp/tools/mktemp" <<'EOF'
#!/bin/sh
d="$(PATH=/usr/bin:/bin mktemp -d "$FAKE_TMP_MADE/ralph-config.XXXXXX")" || exit 1
printf '%s\n' "$d" >> "$FAKE_TMP_LOG"
printf '%s\n' "$d"
EOF
chmod +x "$tmp/bin/ralph" "$tmp/tools/mktemp"
WORK="$(cd "$tmp/work" && pwd -P)"
FAKE_PATH="$tmp/bin:$tmp/tools:/usr/bin:/bin"
NO_RALPH_PATH="$tmp/tools:/usr/bin:/bin"

. "$CONFIG"

# run_script <shell> <path> <script> <key> [NAME=value ...]
# Runs <script> with <shell> -c from the work directory, with $1 the helper
# and $2 <key>, PATH=<path>, and the assignments. Sets _rc, _out, and _err.
run_script() {
  _shell=$1 _path=$2 _script=$3 _key=$4
  shift 4
  if [ "${_shell##*/}" = zsh ]; then
    set -- "$@" "$_shell" -f
  else
    set -- "$@" "$_shell"
  fi
  : > "$tmp/log"
  _rc=0
  (cd "$tmp/work" && env PATH="$_path" FAKE_RALPH_LOG="$tmp/log" FAKE_TMP_MADE="$tmp/made" FAKE_TMP_LOG="$tmp/made.log" "$@" -c "$_script" sh "$CONFIG" "$_key") \
    > "$tmp/out" 2> "$tmp/err" || _rc=$?
  _out="$(cat "$tmp/out")"
  _err="$(cat "$tmp/err")"
}

# run_setting <shell> <path> <key> [NAME=value ...] — run_script with
# `. <helper>; ralph_setting <key>`.
run_setting() {
  _set_shell=$1 _set_path=$2 _set_key=$3
  shift 3
  # shellcheck disable=SC2016  # $1 and $2 expand in the shell under test
  run_script "$_set_shell" "$_set_path" '. "$1"; ralph_setting "$2"' "$_set_key" "$@"
}

test_table() {
  echo ""
  echo "=== the table of defaults ==="
  for key in $KEYS; do
    _rc=0
    _def="$(ralph_setting_default "$key")" || _rc=$?
    assert_eq "ralph_setting_default $key returns 0" 0 "$_rc"
    if [ -n "$_def" ]; then pass "ralph_setting_default $key prints a value ($_def)"; else fail "ralph_setting_default $key prints nothing"; fi
  done
  _rc=0
  _def="$(ralph_setting_default reviewer.nope)" || _rc=$?
  assert_eq "ralph_setting_default reviewer.nope returns 1" 1 "$_rc"
  assert_eq "ralph_setting_default reviewer.nope prints nothing" "" "$_def"
}

# test_shell <shell> runs every case of ralph_setting under <shell>.
test_shell() {
  sh_=$1
  echo ""
  echo "=== ralph_setting under $sh_ ==="

  # exit 0: the value ralph prints, nothing on stderr, ralph run once from
  # the current directory with `config get <key>`.
  for key in $KEYS; do
    run_setting "$sh_" "$FAKE_PATH" "$key" FAKE_RALPH_EXIT=0 "FAKE_RALPH_STDOUT=set-$key"
    assert_eq "$sh_: exit 0 prints ralph's value of $key" "set-$key" "$_out"
    assert_eq "$sh_: exit 0 of $key writes nothing on stderr" "" "$_err"
    assert_eq "$sh_: exit 0 of $key returns 0" 0 "$_rc"
    assert_eq "$sh_: $key runs ralph config get once from the current directory" "$WORK|config get $key" "$(cat "$tmp/log")"
  done

  # 127: ralph is not on PATH, so the default, silently.
  for key in $KEYS; do
    run_setting "$sh_" "$NO_RALPH_PATH" "$key"
    assert_eq "$sh_: no ralph on PATH prints the default of $key" "$(ralph_setting_default "$key")" "$_out"
    assert_eq "$sh_: no ralph on PATH for $key writes nothing on stderr" "" "$_err"
    assert_eq "$sh_: no ralph on PATH for $key returns 0" 0 "$_rc"
  done

  key=reviewer.codex_model
  def="$(ralph_setting_default "$key")"

  # exit 1 with unknown command "config": an older ralph, so the default,
  # silently. Its stderr is two lines, as cobra writes them.
  run_setting "$sh_" "$FAKE_PATH" "$key" FAKE_RALPH_EXIT=1 "FAKE_RALPH_STDERR=Error: unknown command \"config\" for \"ralph\"
Run 'ralph --help' for usage."
  assert_eq "$sh_: unknown command \"config\" prints the default" "$def" "$_out"
  assert_eq "$sh_: unknown command \"config\" writes nothing on stderr" "" "$_err"
  assert_eq "$sh_: unknown command \"config\" returns 0" 0 "$_rc"

  # Any other failure: the default and one line on stderr with the key, the
  # exit code, the first line of ralph's stderr, and the default.
  for c in '2|Error: config get: unknown key "reviewer.codex_model"' \
    '3|Error: ralph.local.toml: [reviewer].codex_model: "a b" is not one word' \
    '1|Error: some other failure' \
    '5|'; do
    rc_="${c%%|*}"
    msg_="${c#*|}"
    if [ -n "$msg_" ]; then
      run_setting "$sh_" "$FAKE_PATH" "$key" "FAKE_RALPH_EXIT=$rc_" "FAKE_RALPH_STDERR=$msg_
a second line"
    else
      run_setting "$sh_" "$FAKE_PATH" "$key" "FAKE_RALPH_EXIT=$rc_"
    fi
    assert_eq "$sh_: exit $rc_ prints the default" "$def" "$_out"
    assert_eq "$sh_: exit $rc_ returns 0" 0 "$_rc"
    assert_one_line "$sh_: exit $rc_ writes one line on stderr" "$_err"
    assert_contains "$sh_: exit $rc_'s line names the key and the exit code" "ralph config get $key exited $rc_" "$_err"
    assert_contains "$sh_: exit $rc_'s line names the default" "using the default $def" "$_err"
    if [ -n "$msg_" ]; then
      assert_contains "$sh_: exit $rc_'s line quotes ralph's first stderr line" "($msg_)" "$_err"
    fi
  done

  # exit 0 with no value is a failure too.
  run_setting "$sh_" "$FAKE_PATH" "$key" FAKE_RALPH_EXIT=0 FAKE_RALPH_STDOUT=
  assert_eq "$sh_: exit 0 with no value prints the default" "$def" "$_out"
  assert_eq "$sh_: exit 0 with no value returns 0" 0 "$_rc"
  assert_one_line "$sh_: exit 0 with no value writes one line on stderr" "$_err"
  assert_contains "$sh_: exit 0 with no value says so" "ralph config get $key exited 0 (no value); using the default $def" "$_err"

  # A key not in the table: nothing on stdout, one line on stderr, a
  # non-zero return, and ralph not run.
  run_setting "$sh_" "$FAKE_PATH" reviewer.nope FAKE_RALPH_EXIT=0 FAKE_RALPH_STDOUT=set
  assert_eq "$sh_: a key not in the table prints nothing" "" "$_out"
  if [ "$_rc" != 0 ]; then pass "$sh_: a key not in the table returns non-zero ($_rc)"; else fail "$sh_: a key not in the table returns 0"; fi
  assert_eq "$sh_: a key not in the table writes one line on stderr" "ralph-config.sh: unknown setting reviewer.nope" "$_err"
  assert_eq "$sh_: a key not in the table runs no ralph" "" "$(cat "$tmp/log")"

  # The retired environment variables change nothing: sourcing sets none,
  # and ralph_setting prints ralph's value or the default with them set.
  # shellcheck disable=SC2016  # $RALPH_* expand in the shell under test
  run_script "$sh_" "$FAKE_PATH" 'unset RALPH_STANDARD_MAX_PIPELINE_CYCLES RALPH_CLAUDE_REVIEWER_MODEL RALPH_CODEX_REVIEWER_MODEL RALPH_CODEX_REASONING_EFFORT; . "$1"; printf "%s|" "${RALPH_STANDARD_MAX_PIPELINE_CYCLES-unset}" "${RALPH_CLAUDE_REVIEWER_MODEL-unset}" "${RALPH_CODEX_REVIEWER_MODEL-unset}" "${RALPH_CODEX_REASONING_EFFORT-unset}"' "$key"
  assert_eq "$sh_: sourcing the helper sets none of the retired variables" "unset|unset|unset|unset|" "$_out"
  for key_ in $KEYS; do
    # shellcheck disable=SC2086  # RETIRED_ENV is a list of assignments
    run_setting "$sh_" "$FAKE_PATH" "$key_" $RETIRED_ENV FAKE_RALPH_EXIT=0 "FAKE_RALPH_STDOUT=set-$key_"
    assert_eq "$sh_: with the retired variables set, exit 0 still prints ralph's value of $key_" "set-$key_" "$_out"
    # shellcheck disable=SC2086
    run_setting "$sh_" "$NO_RALPH_PATH" "$key_" $RETIRED_ENV
    assert_eq "$sh_: with the retired variables set, no ralph on PATH still prints the default of $key_" "$(ralph_setting_default "$key_")" "$_out"
    assert_eq "$sh_: with the retired variables set, no ralph on PATH for $key_ writes nothing on stderr" "" "$_err"
  done

  # A caller under set -eu goes on after a fallback to the default.
  # shellcheck disable=SC2016  # $1, $2, and $v expand in the shell under test
  seteu='set -eu; . "$1"; v=$(ralph_setting "$2"); printf "%s|after\n" "$v"'
  run_script "$sh_" "$FAKE_PATH" "$seteu" "$key" FAKE_RALPH_EXIT=2 "FAKE_RALPH_STDERR=Error: unknown key"
  assert_eq "$sh_: under set -eu, exit 2 goes on with the default" "$def|after" "$_out"
  assert_eq "$sh_: under set -eu, exit 2 returns 0" 0 "$_rc"
  run_script "$sh_" "$NO_RALPH_PATH" "$seteu" "$key"
  assert_eq "$sh_: under set -eu, no ralph on PATH goes on with the default" "$def|after" "$_out"
  assert_eq "$sh_: under set -eu, no ralph on PATH returns 0" 0 "$_rc"
}

main() {
  echo "=== ralph-config.sh tests ==="

  if PATH=/usr/bin:/bin command -v ralph >/dev/null 2>&1; then
    fail "a ralph in /usr/bin or /bin would answer the no-ralph cases: $(PATH=/usr/bin:/bin command -v ralph)"
  fi

  test_table

  shells=/bin/sh
  for name in bash dash zsh; do
    if path_="$(command -v "$name")"; then
      shells="$shells $path_"
    fi
  done
  for s in $shells; do
    test_shell "$s"
  done

  echo ""
  echo "=== temporary directories ==="
  made="$(wc -l < "$tmp/made.log" | tr -d ' ')"
  if [ "$made" -gt 0 ]; then pass "ralph_setting made $made temporary directories"; else fail "ralph_setting made no temporary directory with the stand-in mktemp"; fi
  assert_eq "ralph_setting removes every temporary directory it makes" "" "$(ls -A "$tmp/made")"

  echo ""
  echo "========================================="
  printf 'Results: %d/%d passed' "$_pass" "$_total"
  if [ "$_fail" -gt 0 ]; then
    printf ', %d FAILED' "$_fail"
  fi
  echo ""
  echo "========================================="

  [ "$_fail" -eq 0 ]
}

main
