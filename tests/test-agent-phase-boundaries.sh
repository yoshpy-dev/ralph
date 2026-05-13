#!/usr/bin/env sh
# Regression guard for verifier/tester subagent phase boundaries.

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
  if grep -Fiq -- "$_needle" "$_file"; then
    record_fail "$_desc (unexpected: $_needle in $_file)"
  else
    record_pass "$_desc"
  fi
}

verifier_files="
.claude/agents/verifier.md
templates/base/.claude/agents/verifier.md
.codex/agents/verifier.toml
templates/base/.codex/agents/verifier.toml
"

tester_files="
.claude/agents/tester.md
templates/base/.claude/agents/tester.md
.codex/agents/tester.toml
templates/base/.codex/agents/tester.toml
"

for file in $verifier_files; do
  assert_contains "$file uses static verifier entrypoint" "./scripts/run-static-verify.sh" "$file"
  assert_contains "$file forbids test entrypoint" "./scripts/run-test.sh" "$file"
  assert_contains "$file forbids behavioral tests" "behavioral test" "$file"
  assert_contains "$file mentions static verification" "static verification" "$file"
  assert_not_contains "$file must not use aggregate verifier" "./scripts/run-verify.sh" "$file"
done

for file in $tester_files; do
  assert_contains "$file uses test entrypoint" "./scripts/run-test.sh" "$file"
  assert_contains "$file forbids static verifier entrypoint" "./scripts/run-static-verify.sh" "$file"
  assert_contains "$file forbids static analyzers" "static analyzers" "$file"
  assert_contains "$file forbids type checks" "type checks" "$file"
  assert_contains "$file forbids drift checks" "drift checks" "$file"
  assert_not_contains "$file must not use aggregate verifier" "./scripts/run-verify.sh" "$file"
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
