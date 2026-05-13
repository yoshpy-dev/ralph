#!/usr/bin/env sh
# test-terraform-rule-frontmatter.sh —
# pins the .claude/rules/terraform.md frontmatter contract.
#
# The plan's acceptance criterion requires the `paths:` array to scope
# **/*.tf, **/*.tofu, **/*.tftest.hcl, and **/.terraform.lock.hcl. The
# verifier subagent recommended a deterministic check on these entries
# because a silent drift here would un-scope the rule for Claude Code's
# editor path matcher with no other gate catching it.
#
# Also asserts that the templates/base mirror is byte-identical with the
# root rule (covered by check-sync.sh too, but cheap to belt-and-brace).

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RULE="$PROJECT_ROOT/.claude/rules/terraform.md"
RULE_MIRROR="$PROJECT_ROOT/templates/base/.claude/rules/terraform.md"

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

assert_file() {
  _desc="$1"; _path="$2"
  if [ -f "$_path" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (file not found: $_path)"
  fi
}

assert_grep() {
  _desc="$1"; _pat="$2"; _path="$3"
  if grep -q -- "$_pat" "$_path" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (pattern not found: $_pat in $_path)"
  fi
}

assert_file "rule file exists at .claude/rules/terraform.md" "$RULE"
assert_file "mirror exists at templates/base/.claude/rules/terraform.md" "$RULE_MIRROR"

# Frontmatter sanity: the first non-empty line must be '---', and a
# closing '---' must appear before any markdown heading.
first_line="$(head -n1 "$RULE")"
_total=$((_total + 1))
if [ "$first_line" = "---" ]; then
  _pass=$((_pass + 1))
  printf '  PASS: rule starts with --- frontmatter fence\n'
else
  _fail=$((_fail + 1))
  printf '  FAIL: rule must start with --- (got %s)\n' "$first_line"
fi

# Extract frontmatter block (lines between first two `---`).
fm="$(awk '
  /^---$/ {
    fences++
    next
  }
  fences == 1 { print }
  fences >= 2 { exit }
' "$RULE")"

# Required path globs.
for glob in "**/*.tf" "**/*.tofu" "**/*.tftest.hcl" "**/.terraform.lock.hcl"; do
  _total=$((_total + 1))
  if printf '%s\n' "$fm" | grep -qF -- "$glob"; then
    _pass=$((_pass + 1))
    printf '  PASS: paths: includes %s\n' "$glob"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: paths: missing %s\n' "$glob"
  fi
done

# `paths:` key must actually exist in the frontmatter.
_total=$((_total + 1))
if printf '%s\n' "$fm" | grep -q '^paths:'; then
  _pass=$((_pass + 1))
  printf '  PASS: frontmatter declares a paths: key\n'
else
  _fail=$((_fail + 1))
  printf '  FAIL: frontmatter missing paths: key\n'
fi

# Mirror byte-identical.
_total=$((_total + 1))
if cmp -s "$RULE" "$RULE_MIRROR"; then
  _pass=$((_pass + 1))
  printf '  PASS: rule and templates/base mirror are byte-identical\n'
else
  _fail=$((_fail + 1))
  printf '  FAIL: rule and templates/base mirror differ\n'
  diff -u "$RULE" "$RULE_MIRROR" || true
fi

printf '\n── Summary ──\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
