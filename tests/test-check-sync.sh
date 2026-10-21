#!/usr/bin/env sh
# Regression tests for the docs/reports/ prefixes in check-sync.sh's
# ROOT_ONLY_EXCLUSIONS: a copy of the script runs over a scratch tree.
#
# CHECK_SYNC chooses the script to copy (default: scripts/check-sync.sh), so a
# scratch copy with an exclusion removed runs through the same cases.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CHECK_SYNC="${CHECK_SYNC:-$PROJECT_ROOT/scripts/check-sync.sh}"

_pass=0
_fail=0
_total=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_exit() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc (exit $_actual)"
  else
    record_fail "$_desc (expected exit $_expected, got $_actual)"
  fi
}

# has_root_only_line <path> <output> — check-sync.sh prints
# "  ROOT_ONLY      <path>" for a root file with no template counterpart.
has_root_only_line() {
  awk -v p="$1" '$1 == "ROOT_ONLY" && $2 == p { found = 1 } END { exit !found }' "$2"
}

assert_root_only() {
  _desc="$1"
  _path="$2"
  _out="$3"
  if has_root_only_line "$_path" "$_out"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (no ROOT_ONLY line for $_path)"
    sed 's/^/    out: /' "$_out"
  fi
}

assert_not_root_only() {
  _desc="$1"
  _path="$2"
  _out="$3"
  if has_root_only_line "$_path" "$_out"; then
    record_fail "$_desc (unexpected ROOT_ONLY line for $_path)"
    sed 's/^/    out: /' "$_out"
  else
    record_pass "$_desc"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/check-sync.XXXXXX")"
workdir="$(cd "$workdir" && pwd -P)"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT HUP INT TERM

# A small tree the script passes: AGENTS.md and templates/base/AGENTS.md carry
# the managed block that templates/base/.ralph/core/AGENTS.core.md holds, and
# the pack directories exist (without templates/packs/ the script still passes
# but find prints an error). The copy sits in scripts/, which the script scans,
# but scripts/check-sync.sh is in ROOT_ONLY_EXCLUSIONS, so it needs no template
# copy.
tree="$workdir/tree"
mkdir -p "$tree/scripts" "$tree/templates/base/.ralph/core" \
  "$tree/templates/packs" "$tree/packs/languages" "$tree/docs/reports"
cp "$CHECK_SYNC" "$tree/scripts/check-sync.sh"
printf 'managed line\n' >"$tree/templates/base/.ralph/core/AGENTS.core.md"
for _agents in "$tree/AGENTS.md" "$tree/templates/base/AGENTS.md"; do
  printf '# AGENTS.md\n\n<!-- BEGIN RALPH MANAGED (ralph:agents-md) -->\nmanaged line\n<!-- END RALPH MANAGED -->\n' >"$_agents"
done

# run_check_sync <label> [<path>...] — adds the files to the tree, runs the
# copy, and removes them again; sets RC and OUT.
run_check_sync() {
  _label="$1"
  shift
  for _f in "$@"; do
    mkdir -p "$(dirname "$tree/$_f")"
    printf 'report\n' >"$tree/$_f"
  done
  OUT="$workdir/$_label.out"
  set +e
  bash "$tree/scripts/check-sync.sh" >"$OUT" 2>&1
  RC=$?
  set -e
  for _f in "$@"; do
    rm -f "$tree/$_f"
  done
}

printf '== check-sync.sh over a scratch tree (%s) ==\n' "$CHECK_SYNC"

run_check_sync baseline
assert_exit "baseline tree passes" 0 "$RC"

# The reviewer seat's report and the org report (F8).
run_check_sync org-reports docs/reports/review-x.md docs/reports/org-manifest-x.md
assert_exit "review- and org-manifest- reports pass" 0 "$RC"
assert_not_root_only "review-x.md is excluded" docs/reports/review-x.md "$OUT"
assert_not_root_only "org-manifest-x.md is excluded" docs/reports/org-manifest-x.md "$OUT"

# The exclusions are prefixes, not the whole of docs/reports/: adding a report
# under another name fails, and the two reports stay excluded in that run.
run_check_sync other docs/reports/review-x.md docs/reports/org-manifest-x.md docs/reports/other-x.md
assert_exit "a report under another name fails" 1 "$RC"
assert_root_only "other-x.md is ROOT_ONLY" docs/reports/other-x.md "$OUT"
assert_not_root_only "review-x.md is still excluded" docs/reports/review-x.md "$OUT"
assert_not_root_only "org-manifest-x.md is still excluded" docs/reports/org-manifest-x.md "$OUT"

run_check_sync reviewer docs/reports/reviewer.md
assert_exit "reviewer.md fails" 1 "$RC"
assert_root_only "reviewer.md is ROOT_ONLY (review- matches only part of the name)" docs/reports/reviewer.md "$OUT"

run_check_sync mid-name docs/reports/x-review-y.md
assert_exit "x-review-y.md fails" 1 "$RC"
assert_root_only "x-review-y.md is ROOT_ONLY (review- is not at the start of the name)" docs/reports/x-review-y.md "$OUT"

# A pattern matches at the start of the path only, not after another directory.
run_check_sync nested docs/old/docs/reports/review-x.md
assert_exit "an excluded prefix after another directory fails" 1 "$RC"
assert_root_only "docs/old/docs/reports/review-x.md is ROOT_ONLY" docs/old/docs/reports/review-x.md "$OUT"

# An exclusion that predates F8 still holds.
run_check_sync self-review docs/reports/self-review-x.md
assert_exit "self-review- report passes" 0 "$RC"
assert_not_root_only "self-review-x.md is excluded" docs/reports/self-review-x.md "$OUT"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
