#!/usr/bin/env sh
# shellcheck disable=SC1090  # the whole point is sourcing $CONFIG and $FAKE_RALPH dynamically
set -eu

# test-ralph-config.sh — tests for scripts/ralph-config.sh, whose
# ralph_setting <key> and ralph_setting_or_fail <key> read one setting with
# `ralph config get <key>` and fall back to its table of defaults
# (ralph_setting_default) by the exit code: ralph_setting on every failure,
# ralph_setting_or_fail only when ralph is not on PATH or is too old.
#
# The stand-in ralph of tests/lib/fake-ralph.sh answers each key with an exit
# code and output the case sets, and logs its directory and arguments. A case
# sources the helper in a fresh shell, under /bin/sh and each of bash, dash,
# and zsh (with -f) found on PATH, and calls one of the two functions from a
# work directory, on one key or on several in turn. The PATH of a case is the
# stand-in's directory (left out for ralph not on PATH), a stand-in mktemp's,
# and /usr/bin:/bin, so the ralph a developer installed (such as
# /opt/homebrew/bin/ralph) is never reached.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG="${PROJECT_ROOT}/scripts/ralph-config.sh"
FAKE_RALPH="${PROJECT_ROOT}/tests/lib/fake-ralph.sh"

# The keys of the helper's table; their values are compared with
# config.Default() by TestDefaultsLockStep (internal/config).
KEYS="standard.max_pipeline_cycles standard.base_branch reviewer.claude_model reviewer.claude_effort reviewer.codex_model reviewer.codex_effort plan_visual.browser plan_visual.opener verify.base verify.skip_branch_secret_scan"
# The keys whose default is empty, so an empty value from ralph is their
# value, and the keys whose default is not.
EMPTY_KEYS="standard.base_branch plan_visual.browser plan_visual.opener verify.base"
SET_KEYS="standard.max_pipeline_cycles reviewer.claude_model reviewer.claude_effort reviewer.codex_model reviewer.codex_effort verify.skip_branch_secret_scan"
# The keys the shells other than /bin/sh run the cases with a value, no
# ralph, and an older ralph on: a string default, an empty default, and a
# bool default. /bin/sh runs them on every key.
SOME_KEYS="reviewer.codex_model verify.base verify.skip_branch_secret_scan"

# The environment variables earlier versions of the helper and of the
# scripts read, and the same names each set to a value no default uses.
RETIRED_NAMES="RALPH_STANDARD_MAX_PIPELINE_CYCLES RALPH_CLAUDE_REVIEWER_MODEL RALPH_CODEX_REVIEWER_MODEL RALPH_CODEX_REASONING_EFFORT RALPH_PLAN_VISUAL_BROWSER RALPH_PLAN_VISUAL_OPENER RALPH_XREVIEW_BASE RALPH_VERIFY_BASE RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN"
RETIRED_ENV="RALPH_STANDARD_MAX_PIPELINE_CYCLES=7 RALPH_CLAUDE_REVIEWER_MODEL=env-claude RALPH_CODEX_REVIEWER_MODEL=env-codex RALPH_CODEX_REASONING_EFFORT=env-effort RALPH_PLAN_VISUAL_BROWSER=/env/browser RALPH_PLAN_VISUAL_OPENER=env-opener RALPH_XREVIEW_BASE=env-base RALPH_VERIFY_BASE=env-verify-base RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1"

# The failures other than no ralph and an older one, one per line as
# `<key>|<exit code>|<the first line ralph writes on stderr>`, for keys with
# a default and keys whose default is empty. The stand-in writes a second
# line after ralph's line, which the helper leaves out, and nothing for an
# empty one.
FAILS='reviewer.codex_model|2|config get: reviewer.codex_model: not a key of one value in ralph.toml: no such key
reviewer.claude_model|3|config get: ralph.toml: cannot be read or is invalid: [plan_visual].browser "a b" must be "", "none", or an absolute path
reviewer.codex_effort|1|some other failure
reviewer.claude_effort|5|
verify.base|3|config get: ralph.toml: cannot be read or is invalid: [verify].base "a b" has white space
standard.base_branch|2|config get: standard.base_branch: not a key of one value in ralph.toml: no such key
plan_visual.opener|5|'
# The keys whose default is not empty, for which exit 0 with no value is a
# failure too.
NO_VALUE_KEYS="reviewer.codex_model verify.skip_branch_secret_scan"

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

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir "$tmp/tools" "$tmp/notmp" "$tmp/work" "$tmp/made"
: > "$tmp/made.log"
. "$FAKE_RALPH"
BIN="$tmp/bin"
fake_ralph_install "$BIN"
# A stand-in mktemp makes its directory in $tmp/made and logs it, so the
# test sees whether the helper removes what it makes (macOS's mktemp -d
# ignores TMPDIR). The one in $tmp/notmp always fails, silently.
cat > "$tmp/tools/mktemp" <<'EOF'
#!/bin/sh
d="$(PATH=/usr/bin:/bin mktemp -d "$FAKE_TMP_MADE/ralph-config.XXXXXX")" || exit 1
printf '%s\n' "$d" >> "$FAKE_TMP_LOG"
printf '%s\n' "$d"
EOF
printf '#!/bin/sh\nexit 1\n' > "$tmp/notmp/mktemp"
chmod +x "$tmp/tools/mktemp" "$tmp/notmp/mktemp"
WORK="$(cd "$tmp/work" && pwd -P)"
FAKE_PATH="$BIN:$tmp/tools:/usr/bin:/bin"
NO_RALPH_PATH="$tmp/tools:/usr/bin:/bin"
NO_TMP_PATH="$BIN:$tmp/notmp:/usr/bin:/bin"

. "$CONFIG"

# run_script <shell> <path> <script> <key> [NAME=value ...]
# Runs <script> with <shell> -c from the work directory, with $1 the helper
# and $2 <key>, PATH=<path>, and the assignments, after emptying the
# stand-in's log. Sets _rc, _out, _err, and _bytes (the bytes on stdout).
run_script() {
  _shell=$1 _path=$2 _script=$3 _key=$4
  shift 4
  if [ "${_shell##*/}" = zsh ]; then
    set -- "$@" "$_shell" -f
  else
    set -- "$@" "$_shell"
  fi
  fake_ralph_clear_calls "$BIN"
  _rc=0
  (cd "$tmp/work" && env PATH="$_path" FAKE_TMP_MADE="$tmp/made" FAKE_TMP_LOG="$tmp/made.log" "$@" -c "$_script" sh "$CONFIG" "$_key") \
    > "$tmp/out" 2> "$tmp/err" || _rc=$?
  _out="$(cat "$tmp/out")"
  _err="$(cat "$tmp/err")"
  _bytes="$(wc -c < "$tmp/out" | tr -d ' ')"
}

# run_get <function> <shell> <path> <key> [NAME=value ...] — run_script
# with `. <helper>; <function> <key>`.
run_get() {
  _get_fn=$1 _get_shell=$2 _get_path=$3 _get_key=$4
  shift 4
  # shellcheck disable=SC2016  # $1 and $2 expand in the shell under test
  run_script "$_get_shell" "$_get_path" '. "$1"; '"$_get_fn"' "$2"' "$_get_key" "$@"
}

# run_batch <function> <shell> <path> <keys> [NAME=value ...] — run_script
# with a script that sources the helper once and calls <function> on each of
# <keys> in turn, printing `<key>=<value>|<status>` for each, so one shell
# covers several keys. Each call's line on stderr, if any, follows the one
# before.
run_batch() {
  _b_fn=$1 _b_shell=$2 _b_path=$3 _b_keys=$4
  shift 4
  # shellcheck disable=SC2016  # $1, $k, $v, and $? expand in the shell under test
  run_script "$_b_shell" "$_b_path" '. "$1"; for k in '"$_b_keys"'; do v=$('"$_b_fn"' "$k"); printf "%s=%s|%s\n" "$k" "$v" "$?"; done' "" "$@"
}

# want_batch <keys> <kind> — what run_batch prints when each of <keys> gets
# ralph's value set-<key> (value), its default (default), or ralph's empty
# value (empty), with status 0.
want_batch() {
  _w=''
  for _k in $1; do
    case $2 in
      value) _v="set-$_k" ;;
      default) _v="$(ralph_setting_default "$_k")" ;;
      *) _v='' ;;
    esac
    _w="$_w$_k=$_v|0
"
  done
  printf '%s' "$_w"
}

# want_calls <keys> — the stand-in's log when it was asked each of <keys> in
# turn from the work directory.
want_calls() {
  _w=''
  for _k in $1; do
    _w="$_w$WORK|config get $_k
"
  done
  printf '%s' "$_w"
}

# answer_values <keys> <value|empty> — answers each of <keys> with
# set-<key>, or with an empty value, and nothing else.
answer_values() {
  fake_ralph_reset "$BIN"
  for _k in $1; do
    if [ "$2" = value ]; then
      fake_ralph_value "$BIN" "$_k" "set-$_k"
    else
      fake_ralph_value "$BIN" "$_k" ""
    fi
  done
}

# answer_fails — answers each key of FAILS with its failure, and nothing
# else. Sets _fail_keys to the keys in order.
answer_fails() {
  fake_ralph_reset "$BIN"
  _fail_keys=''
  while IFS='|' read -r _k _c _m; do
    if [ -n "$_m" ]; then
      fake_ralph_fail "$BIN" "$_k" "$_c" "$_m
a second line"
    else
      fake_ralph_fail "$BIN" "$_k" "$_c"
    fi
    _fail_keys="$_fail_keys $_k"
  done <<EOF
$FAILS
EOF
}

# want_line <function> <key> <why> — the line <function> writes on stderr
# for a failure of <key> described by <why>: ralph_setting ends it with the
# default, ralph_setting_or_fail with "not using the default".
want_line() {
  if [ "$1" = ralph_setting_or_fail ]; then
    printf 'ralph-config.sh: %s; not using the default' "$3"
  elif [ -n "$(ralph_setting_default "$2")" ]; then
    printf 'ralph-config.sh: %s; using the default %s' "$3" "$(ralph_setting_default "$2")"
  else
    printf 'ralph-config.sh: %s; using the empty default' "$3"
  fi
}

# want_fails <function> <out|err> — what run_batch of <function> on the keys
# of FAILS prints on stdout (each key's default and 0 for ralph_setting,
# nothing and 1 for ralph_setting_or_fail) or on stderr (one line per key,
# with ralph's first line in parentheses when it wrote one).
want_fails() {
  _w=''
  while IFS='|' read -r _k _c _m; do
    if [ "$2" = out ] && [ "$1" = ralph_setting ]; then
      _w="$_w$_k=$(ralph_setting_default "$_k")|0
"
    elif [ "$2" = out ]; then
      _w="$_w$_k=|1
"
    elif [ -n "$_m" ]; then
      _w="$_w$(want_line "$1" "$_k" "ralph config get $_k exited $_c ($_m)")
"
    else
      _w="$_w$(want_line "$1" "$_k" "ralph config get $_k exited $_c")
"
    fi
  done <<EOF
$FAILS
EOF
  printf '%s' "$_w"
}

# run_stand_in <arguments> — runs the stand-in ralph with <arguments> from
# the work directory. Sets _rc, _out, _err, and _bytes.
run_stand_in() {
  _rc=0
  (cd "$tmp/work" && PATH="$FAKE_PATH" "$BIN/ralph" "$@") > "$tmp/out" 2> "$tmp/err" || _rc=$?
  _out="$(cat "$tmp/out")"
  _err="$(cat "$tmp/err")"
  _bytes="$(wc -c < "$tmp/out" | tr -d ' ')"
}

# test_stand_in checks the stand-in of tests/lib/fake-ralph.sh, which other
# shell tests share: each kind of answer, a key with no answer, a call other
# than config get, the old mode, the log, and fake_ralph_reset.
test_stand_in() {
  echo ""
  echo "=== the stand-in ralph ==="
  fake_ralph_reset "$BIN"
  fake_ralph_value "$BIN" verify.base HEAD~1
  fake_ralph_value "$BIN" plan_visual.opener ""
  fake_ralph_fail "$BIN" standard.base_branch 3 "first line
second line"
  fake_ralph_fail "$BIN" plan_visual.browser 5
  run_stand_in config get verify.base
  assert_eq "stand-in: a value exits 0" 0 "$_rc"
  assert_eq "stand-in: a value is printed" HEAD~1 "$_out"
  assert_eq "stand-in: a value writes nothing on stderr" "" "$_err"
  run_stand_in config get plan_visual.opener
  assert_eq "stand-in: an empty value prints only a newline" 1 "$_bytes"
  run_stand_in config get standard.base_branch
  assert_eq "stand-in: a failure exits with its code" 3 "$_rc"
  assert_eq "stand-in: a failure prints nothing" 0 "$_bytes"
  assert_eq "stand-in: a failure writes its lines on stderr" "first line
second line" "$_err"
  run_stand_in config get plan_visual.browser
  assert_eq "stand-in: a failure with no text exits with its code" 5 "$_rc"
  assert_eq "stand-in: a failure with no text writes nothing" "" "$_out$_err"
  run_stand_in config get reviewer.codex_model
  assert_eq "stand-in: a key with no answer exits 2" 2 "$_rc"
  assert_eq "stand-in: a key with no answer writes ralph's line" "config get: reviewer.codex_model: not a key of one value in ralph.toml: no such key" "$_err"
  run_stand_in doctor --json
  assert_eq "stand-in: a call other than config get exits 99" 99 "$_rc"
  assert_eq "stand-in: a call other than config get says so" "fake ralph: no answer for: doctor --json" "$_err"
  assert_eq "stand-in: the log holds each call with its directory" "$WORK|config get verify.base
$WORK|config get plan_visual.opener
$WORK|config get standard.base_branch
$WORK|config get plan_visual.browser
$WORK|config get reviewer.codex_model
$WORK|doctor --json" "$(fake_ralph_calls "$BIN")"
  fake_ralph_old "$BIN"
  run_stand_in config get verify.base
  assert_eq "stand-in: the old mode exits 1 for config get" 1 "$_rc"
  assert_eq "stand-in: the old mode writes the line of a ralph without config" 'unknown command "config" for "ralph"' "$_err"
  run_stand_in doctor
  assert_eq "stand-in: the old mode names the first argument" 'unknown command "doctor" for "ralph"' "$_err"
  fake_ralph_reset "$BIN"
  run_stand_in config get verify.base
  assert_eq "stand-in: fake_ralph_reset turns the old mode off and removes the answers" 2 "$_rc"
  assert_eq "stand-in: fake_ralph_reset empties the log" "$WORK|config get verify.base" "$(fake_ralph_calls "$BIN")"
}

test_table() {
  echo ""
  echo "=== the table of defaults ==="
  for key in $KEYS; do
    _rc=0
    _def="$(ralph_setting_default "$key")" || _rc=$?
    assert_eq "ralph_setting_default $key returns 0" 0 "$_rc"
  done
  for key in $SET_KEYS; do
    _def="$(ralph_setting_default "$key")"
    if [ -n "$_def" ]; then pass "ralph_setting_default $key prints a value ($_def)"; else fail "ralph_setting_default $key prints nothing"; fi
  done
  for key in $EMPTY_KEYS; do
    assert_eq "ralph_setting_default $key prints an empty line" 1 "$(ralph_setting_default "$key" | wc -c | tr -d ' ')"
  done
  assert_eq "ralph_setting_default verify.skip_branch_secret_scan prints false" false "$(ralph_setting_default verify.skip_branch_secret_scan)"
  _rc=0
  _def="$(ralph_setting_default reviewer.nope)" || _rc=$?
  assert_eq "ralph_setting_default reviewer.nope returns 1" 1 "$_rc"
  assert_eq "ralph_setting_default reviewer.nope prints nothing" "" "$_def"
}

# test_shell <shell> runs every case of both functions under <shell>, on
# every key of the table under /bin/sh and on SOME_KEYS under the others.
# The cases where ralph answers a value or is not on PATH run with the
# retired environment variables set, which change nothing.
test_shell() {
  sh_=$1
  keys_=$SOME_KEYS
  if [ "$sh_" = /bin/sh ]; then keys_=$KEYS; fi
  for fn in ralph_setting ralph_setting_or_fail; do
    echo ""
    echo "=== $fn under $sh_ ==="

    # exit 0: the value ralph prints, nothing on stderr, ralph run once per
    # key from the current directory with `config get <key>`.
    answer_values "$keys_" value
    # shellcheck disable=SC2086  # RETIRED_ENV is a list of assignments
    run_batch "$fn" "$sh_" "$FAKE_PATH" "$keys_" $RETIRED_ENV
    assert_eq "$sh_ $fn: exit 0 prints ralph's value" "$(want_batch "$keys_" value)" "$_out"
    assert_eq "$sh_ $fn: exit 0 writes nothing on stderr" "" "$_err"
    assert_eq "$sh_ $fn: each key runs ralph config get once from the current directory" "$(want_calls "$keys_")" "$(fake_ralph_calls "$BIN")"

    # exit 0 with an empty value, for a key whose default is empty: the
    # value, an empty line, silently.
    answer_values "$EMPTY_KEYS" empty
    # shellcheck disable=SC2086
    run_batch "$fn" "$sh_" "$FAKE_PATH" "$EMPTY_KEYS" $RETIRED_ENV
    assert_eq "$sh_ $fn: exit 0 with an empty value prints it for a key whose default is empty" "$(want_batch "$EMPTY_KEYS" empty)" "$_out"
    assert_eq "$sh_ $fn: exit 0 with an empty value writes nothing on stderr" "" "$_err"
    run_get "$fn" "$sh_" "$FAKE_PATH" verify.base
    assert_eq "$sh_ $fn: exit 0 with an empty verify.base prints an empty line" 1 "$_bytes"
    assert_eq "$sh_ $fn: exit 0 with an empty verify.base returns 0" 0 "$_rc"

    # 127: ralph is not on PATH, so the default, silently.
    # shellcheck disable=SC2086
    run_batch "$fn" "$sh_" "$NO_RALPH_PATH" "$keys_" $RETIRED_ENV
    assert_eq "$sh_ $fn: no ralph on PATH prints the default" "$(want_batch "$keys_" default)" "$_out"
    assert_eq "$sh_ $fn: no ralph on PATH writes nothing on stderr" "" "$_err"

    # exit 1 with unknown command "config" on stderr: an older ralph, so the
    # default, silently.
    fake_ralph_reset "$BIN"
    fake_ralph_old "$BIN"
    run_batch "$fn" "$sh_" "$FAKE_PATH" "$SOME_KEYS"
    assert_eq "$sh_ $fn: an older ralph prints the default" "$(want_batch "$SOME_KEYS" default)" "$_out"
    assert_eq "$sh_ $fn: an older ralph writes nothing on stderr" "" "$_err"
    assert_eq "$sh_ $fn: an older ralph is asked each key once" "$(want_calls "$SOME_KEYS")" "$(fake_ralph_calls "$BIN")"

    # A key not in the table: nothing on stdout, one line on stderr, 2, and
    # ralph not run.
    fake_ralph_reset "$BIN"
    fake_ralph_value "$BIN" reviewer.nope set
    run_get "$fn" "$sh_" "$FAKE_PATH" reviewer.nope
    assert_eq "$sh_ $fn: a key not in the table prints nothing" 0 "$_bytes"
    assert_eq "$sh_ $fn: a key not in the table returns 2" 2 "$_rc"
    assert_eq "$sh_ $fn: a key not in the table writes one line on stderr" "ralph-config.sh: unknown setting reviewer.nope" "$_err"
    assert_eq "$sh_ $fn: a key not in the table runs no ralph" "" "$(fake_ralph_calls "$BIN")"

    # Any other failure: one line on stderr per key with the key, the exit
    # code, the first line of ralph's stderr, and the default or "not using
    # the default"; then the default and 0 (ralph_setting) or nothing and 1
    # (ralph_setting_or_fail).
    answer_fails
    run_batch "$fn" "$sh_" "$FAKE_PATH" "$_fail_keys"
    assert_eq "$sh_ $fn: each failure prints what the function gives instead of a value" "$(want_fails "$fn" out)" "$_out"
    assert_eq "$sh_ $fn: each failure writes one line on stderr" "$(want_fails "$fn" err)" "$_err"

    # exit 0 with no value is a failure for a key whose default is not empty.
    answer_values "$NO_VALUE_KEYS" empty
    run_batch "$fn" "$sh_" "$FAKE_PATH" "$NO_VALUE_KEYS"
    _want_out='' _want_err=''
    for key in $NO_VALUE_KEYS; do
      if [ "$fn" = ralph_setting ]; then
        _want_out="$_want_out$key=$(ralph_setting_default "$key")|0
"
      else
        _want_out="$_want_out$key=|1
"
      fi
      _want_err="$_want_err$(want_line "$fn" "$key" "ralph config get $key exited 0 (no value)")
"
    done
    assert_eq "$sh_ $fn: exit 0 with no value, for a key with a default, prints what a failure gives" "${_want_out%?}" "$_out"
    assert_eq "$sh_ $fn: exit 0 with no value, for a key with a default, writes one line on stderr" "${_want_err%?}" "$_err"

    # No temporary directory: a failure, with ralph not run.
    key=reviewer.codex_model
    answer_values "$key" value
    run_get "$fn" "$sh_" "$NO_TMP_PATH" "$key"
    if [ "$fn" = ralph_setting ]; then
      assert_eq "$sh_ $fn: no temporary directory prints the default" "$(ralph_setting_default "$key")" "$_out"
      assert_eq "$sh_ $fn: no temporary directory returns 0" 0 "$_rc"
    else
      assert_eq "$sh_ $fn: no temporary directory prints nothing" 0 "$_bytes"
      assert_eq "$sh_ $fn: no temporary directory returns 1" 1 "$_rc"
    fi
    assert_eq "$sh_ $fn: no temporary directory writes one line on stderr" "$(want_line "$fn" "$key" "cannot make a temporary directory to run ralph config get $key")" "$_err"
    assert_eq "$sh_ $fn: no temporary directory runs no ralph" "" "$(fake_ralph_calls "$BIN")"

    # A caller under set -eu goes on after ralph_setting's fallback to the
    # default and stops at a failure of ralph_setting_or_fail; both go on
    # with the default when ralph is not on PATH.
    # shellcheck disable=SC2016  # $1, $2, and $v expand in the shell under test
    seteu='set -eu; . "$1"; v=$('"$fn"' "$2"); printf "%s|after\n" "$v"'
    key=verify.base
    fake_ralph_reset "$BIN"
    fake_ralph_fail "$BIN" "$key" 3 "config get: ralph.toml: cannot be read or is invalid"
    run_script "$sh_" "$FAKE_PATH" "$seteu" "$key"
    if [ "$fn" = ralph_setting ]; then
      assert_eq "$sh_ $fn: under set -eu, exit 3 goes on with the default" "|after" "$_out"
      assert_eq "$sh_ $fn: under set -eu, exit 3 returns 0" 0 "$_rc"
    else
      assert_eq "$sh_ $fn: under set -eu, exit 3 stops the caller" "" "$_out"
      if [ "$_rc" != 0 ]; then pass "$sh_ $fn: under set -eu, exit 3 makes the caller fail ($_rc)"; else fail "$sh_ $fn: under set -eu, exit 3 lets the caller succeed"; fi
    fi
    run_script "$sh_" "$NO_RALPH_PATH" "$seteu" "$key"
    assert_eq "$sh_ $fn: under set -eu, no ralph on PATH goes on with the default" "|after" "$_out"
    assert_eq "$sh_ $fn: under set -eu, no ralph on PATH returns 0" 0 "$_rc"
  done

  # Sourcing the helper sets none of the retired variables.
  _names_script="unset $RETIRED_NAMES; . \"\$1\"; printf \"%s|\""
  _want=""
  for name_ in $RETIRED_NAMES; do
    _names_script="$_names_script \"\${$name_-unset}\""
    _want="${_want}unset|"
  done
  # shellcheck disable=SC2086
  run_script "$sh_" "$FAKE_PATH" "$_names_script" reviewer.codex_model $RETIRED_ENV
  assert_eq "$sh_: sourcing the helper sets none of the retired variables" "$_want" "$_out"
}

main() {
  echo "=== ralph-config.sh tests ==="

  if PATH=/usr/bin:/bin command -v ralph >/dev/null 2>&1; then
    fail "a ralph in /usr/bin or /bin would answer the no-ralph cases: $(PATH=/usr/bin:/bin command -v ralph)"
  fi

  test_stand_in
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
  if [ "$made" -gt 0 ]; then pass "the helper made $made temporary directories"; else fail "the helper made no temporary directory with the stand-in mktemp"; fi
  assert_eq "the helper removes every temporary directory it makes" "" "$(ls -A "$tmp/made")"

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
