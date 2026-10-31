#!/usr/bin/env sh
# ralph-config.sh — reads settings from ralph.toml and ralph.local.toml for
# the skills (/plan and /cross-review) and the scripts (plan-visual.sh,
# xreview-helpers.sh, detect-changed-languages.sh, and run-verify.sh).
# Sourcing it defines ralph_setting_default, ralph_setting,
# ralph_setting_or_fail, and _ralph_setting_get, the part the last two
# share, and sets no variable; it holds no value a person chooses. The
# values live in ralph.toml and the git-ignored ralph.local.toml, which
# `ralph config get` reads: the keys [standard].max_pipeline_cycles and the
# four of [reviewer] for the skills, and [standard].base_branch, the two of
# [plan_visual], and the two of [verify] for the scripts. Only
# ralph.local.toml may set those last five; ralph refuses a ralph.toml that
# sets one.
#
# It reads no environment variable. RALPH_STANDARD_MAX_PIPELINE_CYCLES,
# RALPH_CLAUDE_REVIEWER_MODEL, RALPH_CODEX_REVIEWER_MODEL, and
# RALPH_CODEX_REASONING_EFFORT, which earlier versions read, change nothing,
# and neither do RALPH_PLAN_VISUAL_BROWSER, RALPH_PLAN_VISUAL_OPENER,
# RALPH_XREVIEW_BASE, RALPH_VERIFY_BASE, and
# RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN, which earlier versions of the
# scripts read.
#
# Usage, from the repository root (`ralph config get` reads ./ralph.toml):
#   . ./scripts/ralph-config.sh
#   model=$(ralph_setting reviewer.codex_model)
#   base=$(ralph_setting_or_fail verify.base) || <take the safe side>
#
# ralph_setting <key> and ralph_setting_or_fail <key> run
# `command ralph config get <key>` in the current directory and print one
# value, chosen by ralph's exit code:
#   0         the value ralph printed. An empty value is the value of a key
#             whose default is empty (an unset browser, opener, or base):
#             an empty line, with nothing on stderr. For any other key,
#             printing nothing counts as a failure below.
#   127       ralph is not on PATH: the default from the table, silently.
#   1 with `unknown command "config"` on stderr
#             a ralph older than `config get`: the default, silently.
#   any other exit code, or a temporary directory that cannot be made
#             a failure: one line on stderr with the key and, when ralph
#             ran, its exit code and the first line it wrote on stderr, if
#             any. The exit code is 2 for a key that ralph does not know, 3
#             for a ralph.toml or ralph.local.toml that cannot be read or
#             fails validation (a mistake in any key fails every key), and 1
#             or another code for any other error. The two functions differ
#             only here. ralph_setting ends the line with "using the default
#             <value>" ("using the empty default" for an empty one), prints
#             the default, and returns 0, so a caller under `set -e` goes on
#             with it. ralph_setting_or_fail ends the line with "not using
#             the default", prints nothing, and returns 1.
# Use ralph_setting where the default is a harmless answer to a mistake in
# the settings, and ralph_setting_or_fail where the default could hide it:
# a base that silently falls back to the automatic one changes what a
# verify, a review, or a secret scan covers, so its caller stops or takes
# the safe side instead. Only a ralph with `config get` reads
# ralph.local.toml, so with no ralph or an older one there is no setting to
# get wrong, and both give the default. A key that is not in the table
# prints nothing, writes one line on stderr, and returns 2, in both.

# ralph_setting_default <key>
# The table of defaults, the only one in this file: one line per key, of the
# form
#   <key>) printf '%s\n' '<value>' ;;
# with the value in single quotes as `ralph config get` prints it ('' for an
# empty string, 'false' for a bool). It prints the key's default, or nothing
# and returns 1 for a key not in the table. The defaults equal
# config.Default() in the ralph CLI. In the ralph CLI's own repository, the
# regular expression of internal/config/defaults_sync_settings_test.go reads
# these lines for TestDefaultsLockStep, and tests/test-ralph-config.sh calls
# this function: change those tests when you change the form of a line.
ralph_setting_default() {
  case "${1-}" in
    standard.max_pipeline_cycles) printf '%s\n' '2' ;;
    standard.base_branch) printf '%s\n' '' ;;
    reviewer.claude_model) printf '%s\n' 'fable' ;;
    reviewer.claude_effort) printf '%s\n' 'xhigh' ;;
    reviewer.codex_model) printf '%s\n' 'gpt-6-astra' ;;
    reviewer.codex_effort) printf '%s\n' 'xhigh' ;;
    plan_visual.browser) printf '%s\n' '' ;;
    plan_visual.opener) printf '%s\n' '' ;;
    verify.base) printf '%s\n' '' ;;
    verify.skip_branch_secret_scan) printf '%s\n' 'false' ;;
    *) return 1 ;;
  esac
}

# _ralph_setting_get <key>
# Runs ralph config get <key> and sorts the outcome as the header of this
# file describes, for ralph_setting and ralph_setting_or_fail. It sets
# _rs_default; _rs_why, empty when the outcome is a value and otherwise the
# failure without the line's prefix and ending; and _rs_value, the value to
# print when _rs_why is empty (ralph's value, or the default for no ralph or
# an older one). For a key not in the table it writes one line on stderr and
# returns 2. Its variables start with _rs_, since POSIX sh has no local
# variables.
_ralph_setting_get() {
  if ! _rs_default=$(ralph_setting_default "${1-}"); then
    printf 'ralph-config.sh: unknown setting %s\n' "${1-}" >&2
    return 2
  fi
  _rs_value=''
  _rs_why=''
  if ! _rs_dir=$(mktemp -d); then
    _rs_why="cannot make a temporary directory to run ralph config get $1"
    return 0
  fi
  _rs_rc=0
  command ralph config get "$1" > "$_rs_dir/out" 2> "$_rs_dir/err" || _rs_rc=$?
  _rs_out=$(cat "$_rs_dir/out")
  _rs_first=$(sed -n 1p "$_rs_dir/err")
  _rs_old=0
  if [ "$_rs_rc" = 1 ] && grep -q 'unknown command "config"' "$_rs_dir/err"; then
    _rs_old=1
  fi
  rm -rf "$_rs_dir"
  if [ "$_rs_rc" = 0 ] && { [ -n "$_rs_out" ] || [ -z "$_rs_default" ]; }; then
    _rs_value=$_rs_out
    return 0
  fi
  if [ "$_rs_rc" = 127 ] || [ "$_rs_old" = 1 ]; then
    _rs_value=$_rs_default
    return 0
  fi
  if [ "$_rs_rc" = 0 ]; then
    _rs_first="no value"
  fi
  if [ -n "$_rs_first" ]; then
    _rs_why="ralph config get $1 exited $_rs_rc ($_rs_first)"
  else
    _rs_why="ralph config get $1 exited $_rs_rc"
  fi
}

# ralph_setting <key>
# Prints the value of <key>, or its default after one line on stderr when
# ralph fails, and returns 0, as the header of this file describes.
ralph_setting() {
  _ralph_setting_get "${1-}" || return 2
  if [ -n "$_rs_why" ]; then
    if [ -n "$_rs_default" ]; then
      printf 'ralph-config.sh: %s; using the default %s\n' "$_rs_why" "$_rs_default" >&2
    else
      printf 'ralph-config.sh: %s; using the empty default\n' "$_rs_why" >&2
    fi
    _rs_value=$_rs_default
  fi
  printf '%s\n' "$_rs_value"
}

# ralph_setting_or_fail <key>
# Prints the value of <key> and returns 0 as ralph_setting does, but when
# ralph fails it writes one line on stderr, prints nothing, and returns 1,
# as the header of this file describes.
ralph_setting_or_fail() {
  _ralph_setting_get "${1-}" || return 2
  if [ -n "$_rs_why" ]; then
    printf 'ralph-config.sh: %s; not using the default\n' "$_rs_why" >&2
    return 1
  fi
  printf '%s\n' "$_rs_value"
}
