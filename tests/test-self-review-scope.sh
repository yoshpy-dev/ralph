#!/usr/bin/env sh
# Regression guard for the self-review phase boundary.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

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

assert_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -Fiq -- "$_needle" "$_file"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in $_file)"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -Fq -- "$_needle" "$_file"; then
    record_fail "$_desc (unexpected: $_needle in $_file)"
  else
    record_pass "$_desc"
  fi
}

self_review_files="
.agents/skills/self-review/SKILL.md
.claude/skills/self-review/SKILL.md
templates/base/.agents/skills/self-review/SKILL.md
templates/base/.claude/skills/self-review/SKILL.md
.claude/agents/reviewer.md
templates/base/.claude/agents/reviewer.md
.codex/agents/reviewer.toml
templates/base/.codex/agents/reviewer.toml
"

for file in $self_review_files; do
  assert_contains "$file declares diff quality only" "diff quality only" "$file"
  assert_contains "$file forbids running tests" "Do NOT run tests" "$file"
  assert_contains "$file forbids static analysis" "static analysis" "$file"
  assert_contains "$file forbids broad unrelated scope" "broad unrelated" "$file"
  assert_contains "$file forbids repo audits" "repo audits" "$file"

  assert_not_contains "$file must not call run-test.sh" "run-test.sh" "$file"
  assert_not_contains "$file must not call run-static-verify.sh" "run-static-verify.sh" "$file"
  assert_not_contains "$file must not call run-verify.sh" "run-verify.sh" "$file"
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
