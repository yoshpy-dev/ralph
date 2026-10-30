#!/usr/bin/env sh
# ralph-config.sh — reads the standard flow's settings from ralph.toml for
# the skills (/plan and /cross-review). Sourcing it defines two functions
# and sets no variable; it holds no value a person chooses. The values live
# in ralph.toml and the git-ignored ralph.local.toml, which
# `ralph config get` reads (the keys [standard].max_pipeline_cycles and the
# four of [reviewer]).
#
# It reads no environment variable. RALPH_STANDARD_MAX_PIPELINE_CYCLES,
# RALPH_CLAUDE_REVIEWER_MODEL, RALPH_CODEX_REVIEWER_MODEL, and
# RALPH_CODEX_REASONING_EFFORT, which earlier versions read, change nothing.
#
# Usage, from the repository root (`ralph config get` reads ./ralph.toml):
#   . ./scripts/ralph-config.sh
#   model=$(ralph_setting reviewer.codex_model)
#
# ralph_setting <key> runs `command ralph config get <key>` in the current
# directory and prints one value, chosen by ralph's exit code:
#   0         the value ralph printed. Printing nothing counts as a failure
#             below.
#   127       ralph is not on PATH: the default from the table, silently.
#   1 with `unknown command "config"` on stderr
#             a ralph older than `config get`: the default, silently.
#   any other (2 for a key that ralph does not know, 3 for a ralph.toml or
#             ralph.local.toml that cannot be read or fails validation, any
#             other 1, and the rest)
#             one line on stderr with the key, the exit code, the first line
#             ralph wrote on stderr, if any, and the default; then the
#             default.
# It returns 0 in all of these, so a caller under `set -e` goes on with the
# default. A key that is not in the table prints nothing, writes one line on
# stderr, and returns 2.

# ralph_setting_default <key>
# The table of defaults, the only one in this file: one line per key, of the
# form
#   <key>) printf '%s\n' '<value>' ;;
# with the value in single quotes. It prints the key's default, or nothing
# and returns 1 for a key not in the table. The defaults equal
# config.Default() in the ralph CLI. In the ralph CLI's own repository,
# internal/config/defaults_sync_test.go reads these lines with a regular
# expression, and tests/test-ralph-config.sh calls this function: change
# those tests when you change the form of a line.
ralph_setting_default() {
  case "${1-}" in
    standard.max_pipeline_cycles) printf '%s\n' '2' ;;
    reviewer.claude_model) printf '%s\n' 'fable' ;;
    reviewer.claude_effort) printf '%s\n' 'xhigh' ;;
    reviewer.codex_model) printf '%s\n' 'gpt-6-astra' ;;
    reviewer.codex_effort) printf '%s\n' 'xhigh' ;;
    *) return 1 ;;
  esac
}

# ralph_setting <key>
# Prints the value of <key> as the header of this file describes. Its
# variables start with _rs_, since POSIX sh has no local variables.
ralph_setting() {
  if ! _rs_default=$(ralph_setting_default "${1-}"); then
    printf 'ralph-config.sh: unknown setting %s\n' "${1-}" >&2
    return 2
  fi
  if ! _rs_dir=$(mktemp -d); then
    printf 'ralph-config.sh: cannot make a temporary directory to run ralph config get %s; using the default %s\n' "$1" "$_rs_default" >&2
    printf '%s\n' "$_rs_default"
    return 0
  fi
  _rs_rc=0
  command ralph config get "$1" > "$_rs_dir/out" 2> "$_rs_dir/err" || _rs_rc=$?
  _rs_value=$(cat "$_rs_dir/out")
  _rs_first=$(sed -n 1p "$_rs_dir/err")
  _rs_old=0
  if [ "$_rs_rc" = 1 ] && grep -q 'unknown command "config"' "$_rs_dir/err"; then
    _rs_old=1
  fi
  rm -rf "$_rs_dir"
  if [ "$_rs_rc" = 0 ] && [ -n "$_rs_value" ]; then
    printf '%s\n' "$_rs_value"
    return 0
  fi
  if [ "$_rs_rc" = 127 ] || [ "$_rs_old" = 1 ]; then
    printf '%s\n' "$_rs_default"
    return 0
  fi
  if [ "$_rs_rc" = 0 ]; then
    _rs_first="no value"
  fi
  if [ -n "$_rs_first" ]; then
    printf 'ralph-config.sh: ralph config get %s exited %s (%s); using the default %s\n' "$1" "$_rs_rc" "$_rs_first" "$_rs_default" >&2
  else
    printf 'ralph-config.sh: ralph config get %s exited %s; using the default %s\n' "$1" "$_rs_rc" "$_rs_default" >&2
  fi
  printf '%s\n' "$_rs_default"
}
