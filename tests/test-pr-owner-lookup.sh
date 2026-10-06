#!/usr/bin/env sh
# tests/test-pr-owner-lookup.sh — run the expressions that /pr step 5.c
# writes in .claude/skills/pr/SKILL.md: the sed that takes the owner from
# origin's URL, and the two jq filters of the open-PR lookups (into the
# intended base, and into any base). The expressions are read from SKILL.md,
# so editing them there without keeping the behavior fails this test. The
# jq cases are skipped when jq is not on PATH; the sed cases always run.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SKILL="${PROJECT_ROOT}/.claude/skills/pr/SKILL.md"

_pass=0
_fail=0
_total=0
_tmp=""

# An if, not "[ ] && rm": under set -e a false test here makes the EXIT trap
# exit 1 when no temp dir was made (the jq SKIP path).
cleanup() {
  if [ -n "$_tmp" ]; then
    rm -rf "$_tmp"
  fi
}
trap cleanup EXIT HUP INT TERM

pass() {
  _total=$((_total + 1))
  _pass=$((_pass + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _total=$((_total + 1))
  _fail=$((_fail + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    pass "$_desc"
  else
    fail "$_desc"
    printf '    expected: %s\n    actual:   %s\n' "$_expected" "$_actual"
  fi
}

summary() {
  printf '\npr-owner-lookup tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
}

# sed_expr FILE — the single-quoted argument after
# "git remote get-url origin | sed -E " (one per matching line).
sed_expr() {
  grep -F "git remote get-url origin | sed -E '" "$1" |
    sed -n "s/.*git remote get-url origin | sed -E '\([^']*\)'.*/\1/p"
}

# base_jq FILE — the --jq filter of the lookup line that passes --base <base>.
base_jq() {
  grep -F 'gh pr list --head' "$1" | grep -F -- "--jq '" | grep -F -- '--base <base>' |
    sed -n "s/.*--jq '\([^']*\)'.*/\1/p"
}

# any_base_jq FILE — the --jq filter of the lookup line with no --base.
any_base_jq() {
  grep -F 'gh pr list --head' "$1" | grep -F -- "--jq '" | grep -v -F -- '--base' |
    sed -n "s/.*--jq '\([^']*\)'.*/\1/p"
}

# count_lines TEXT — number of lines in TEXT (0 when empty).
count_lines() {
  if [ -z "$1" ]; then
    printf '0'
  else
    printf '%s\n' "$1" | wc -l | tr -d ' '
  fi
}

# expect_one NAME TEXT — TEXT is exactly one extracted expression.
expect_one() {
  _n="$(count_lines "$2")"
  if [ "$_n" = "1" ]; then
    pass "exactly one ${1}"
  else
    fail "exactly one ${1} (found ${_n}; did the 5.c wording change?)"
    _found=0
  fi
}

# expect_owner_placeholder NAME FILTER — the test substitutes "<owner>".
expect_owner_placeholder() {
  case "$2" in
    *'"<owner>"'*) pass "${1} keeps the \"<owner>\" placeholder" ;;
    *) fail "${1} keeps the \"<owner>\" placeholder"; _found=0 ;;
  esac
}

# run_jq DESC FILTER FIXTURE EXPECTED — the filter, with "<owner>" replaced
# by "yoshpy-dev", exits 0 on FIXTURE and prints EXPECTED (gh --jq prints
# strings raw, as jq -r does).
run_jq() {
  _desc="$1"
  _filter="$(printf '%s' "$2" | sed 's/"<owner>"/"yoshpy-dev"/g')"
  set +e
  _out="$(jq -r "$_filter" "$3" 2>&1)"
  _rc="$?"
  set -e
  if [ "$_rc" -ne 0 ]; then
    fail "${_desc} (jq exited ${_rc})"
    printf '    output: %s\n' "$_out"
  else
    assert_eq "$_desc" "$4" "$_out"
  fi
}

printf '==> expressions found in .claude/skills/pr/SKILL.md\n'
_sed="$(sed_expr "$SKILL")"
_jq_base="$(base_jq "$SKILL")"
_jq_any="$(any_base_jq "$SKILL")"
_found=1
expect_one "owner sed" "$_sed"
expect_one "lookup jq with --base" "$_jq_base"
expect_one "lookup jq without --base" "$_jq_any"
if [ "$_found" = 1 ]; then
  expect_owner_placeholder "lookup jq with --base" "$_jq_base"
  expect_owner_placeholder "lookup jq without --base" "$_jq_any"
fi
if [ "$_found" = 0 ]; then
  summary
  exit 1
fi

printf '==> the four faces carry the same expressions\n'
for face in .agents/skills templates/base/.claude/skills templates/base/.agents/skills; do
  _file="${PROJECT_ROOT}/${face}/pr/SKILL.md"
  if [ ! -f "$_file" ]; then
    fail "${face}/pr/SKILL.md (file missing)"
    continue
  fi
  if [ "$(sed_expr "$_file")" = "$_sed" ] &&
     [ "$(base_jq "$_file")" = "$_jq_base" ] &&
     [ "$(any_base_jq "$_file")" = "$_jq_any" ]; then
    pass "${face}/pr/SKILL.md"
  else
    fail "${face}/pr/SKILL.md (5.c expressions differ from .claude/skills/pr/SKILL.md)"
  fi
done

printf '==> owner from origin URL\n'
for _case in \
  'https://github.com/yoshpy-dev/ralph.git yoshpy-dev' \
  'https://github.com/yoshpy-dev/ralph yoshpy-dev' \
  'https://github.com/yoshpy-dev/ralph/ yoshpy-dev' \
  'ssh://git@github.com/yoshpy-dev/ralph.git yoshpy-dev' \
  'ssh://git@github.com:22/yoshpy-dev/ralph.git yoshpy-dev' \
  'git@github.com:yoshpy-dev/ralph.git yoshpy-dev' \
  'git@github.com.emu:in-house-tools-dena/foo.git in-house-tools-dena'
do
  _url="${_case% *}"
  _want="${_case##* }"
  assert_eq "$_url" "$_want" "$(printf '%s\n' "$_url" | sed -E "$_sed")"
done

if command -v jq >/dev/null 2>&1; then
  _tmp="$(mktemp -d)"

  cat > "$_tmp/mixed.json" <<'JSON'
[
  {"url": "https://github.com/yoshpy-dev/ralph/pull/1", "headRepositoryOwner": {"id": "a", "login": "YoshPy-Dev"}},
  {"url": "https://github.com/yoshpy-dev/ralph/pull/2", "headRepositoryOwner": null},
  {"url": "https://github.com/yoshpy-dev/ralph/pull/3", "headRepositoryOwner": {"id": "b", "login": "someone-else"}}
]
JSON

  cat > "$_tmp/null-owner.json" <<'JSON'
[
  {"url": "https://github.com/yoshpy-dev/ralph/pull/2", "headRepositoryOwner": null}
]
JSON

  cat > "$_tmp/other-base.json" <<'JSON'
[
  {"url": "https://github.com/yoshpy-dev/ralph/pull/4", "baseRefName": "release", "headRepositoryOwner": {"id": "a", "login": "Yoshpy-Dev"}},
  {"url": "https://github.com/yoshpy-dev/ralph/pull/5", "baseRefName": "main", "headRepositoryOwner": {"id": "b", "login": "someone-else"}},
  {"url": "https://github.com/yoshpy-dev/ralph/pull/6", "baseRefName": "main", "headRepositoryOwner": null}
]
JSON

  cat > "$_tmp/other-owner-only.json" <<'JSON'
[
  {"url": "https://github.com/yoshpy-dev/ralph/pull/5", "baseRefName": "release", "headRepositoryOwner": {"id": "b", "login": "someone-else"}}
]
JSON

  printf '==> lookup into the intended base (--base <base>)\n'
  run_jq "upper-case login matches; null and other owners are left out" \
    "$_jq_base" "$_tmp/mixed.json" 'https://github.com/yoshpy-dev/ralph/pull/1'
  run_jq "null head owner alone gives empty output" \
    "$_jq_base" "$_tmp/null-owner.json" ''

  printf '==> lookup into any base (no --base)\n'
  run_jq "open PR into another base, mixed-case login, prints its base and URL" \
    "$_jq_any" "$_tmp/other-base.json" 'release https://github.com/yoshpy-dev/ralph/pull/4'
  run_jq "only another owner's PR gives empty output" \
    "$_jq_any" "$_tmp/other-owner-only.json" ''
else
  printf '==> open-PR lookup filters\n'
  printf '  SKIP: jq not on PATH\n'
fi

summary
[ "$_fail" -eq 0 ]
