#!/usr/bin/env sh
# ralph-common.sh — shared shell helpers sourced by ralph scripts
# sourced, not executed
#
# Must NOT set shell options (set -e etc.) itself; safe when sourced under
# both 'set -eu' and 'set -euo pipefail'.

[ -n "${RALPH_COMMON_SOURCED:-}" ] && return
RALPH_COMMON_SOURCED=1

# ts — print current UTC timestamp in ISO-8601 format
ts() { date -u '+%Y-%m-%dT%H:%M:%SZ'; }

# ts_file — print UTC timestamp suitable for file names
ts_file() { date -u '+%Y-%m-%d-%H%M%S'; }

# log — print a timestamped info line to stdout
log() { printf '[%s] %s\n' "$(ts)" "$*"; }

# log_error — print a timestamped error line to stderr
log_error() { printf '[%s] ERROR: %s\n' "$(ts)" "$*" >&2; }

# default_branch — print the repository's default branch name.
#
# Resolution order:
#   1. git symbolic-ref --quiet --short refs/remotes/origin/HEAD (strip "origin/")
#   2. refs/heads/main if it exists
#   3. refs/heads/master if it exists
#   4. Error: write to stderr and return 1
#
# Pure function: no side effects beyond reading git refs.
# sh-compatible: no 'local' keyword, uses _db_ prefix to avoid variable collisions.
default_branch() {
  _db_remote_head="$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  if [ -n "$_db_remote_head" ]; then
    printf '%s\n' "${_db_remote_head#origin/}"
    return 0
  fi
  if git show-ref --verify --quiet refs/heads/main; then
    printf 'main\n'
    return 0
  fi
  if git show-ref --verify --quiet refs/heads/master; then
    printf 'master\n'
    return 0
  fi
  printf 'ralph-common: could not resolve default branch\n' >&2
  return 1
}

# detect_active_plan_dir — print the newest directory-based plan path.
#
# Scans docs/plans/active/ for directories containing _manifest.md, sorts
# descending (date-prefixed names sort chronologically) and prints the first.
# Prints nothing and returns 0 if no directory-based plan exists.
#
# Pure function: no side effects.
detect_active_plan_dir() {
  find docs/plans/active -maxdepth 1 -type d ! -name active 2>/dev/null | while read -r _dap_d; do
    if [ -f "${_dap_d}/_manifest.md" ]; then printf '%s\n' "$_dap_d"; fi
  done | sort -r | head -1 || true
}
