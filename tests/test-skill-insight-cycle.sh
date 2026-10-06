#!/usr/bin/env sh
# tests/test-skill-insight-cycle.sh — the insight-event command of /self-review,
# /verify, /test, /sync-docs, and /cross-review passes --cycle auto on all four
# faces (.claude/skills, .agents/skills, and their templates/base copies), so a
# cycle-2 pipeline run is not recorded as cycle 1.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

_pass=0
_fail=0
_total=0

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

# insight_block FILE — print the first fenced block inside the
# "## Insight event" section of FILE (nothing when there is none).
insight_block() {
  awk '
    /^## Insight event/ { seen = 1; next }
    seen && !inside && /^## / { exit }
    seen && /^```/ { if (inside) exit; inside = 1; next }
    inside { print }
  ' "$1"
}

printf '==> insight-event command passes --cycle auto\n'
for skill in self-review verify test sync-docs cross-review; do
  for face in .claude/skills .agents/skills templates/base/.claude/skills templates/base/.agents/skills; do
    _file="${PROJECT_ROOT}/${face}/${skill}/SKILL.md"
    _desc="${face}/${skill}/SKILL.md"
    if [ ! -f "$_file" ]; then
      fail "${_desc} (file missing)"
      continue
    fi
    _block="$(insight_block "$_file")"
    if ! printf '%s\n' "$_block" | grep -q 'insights-append\.sh'; then
      fail "${_desc} (no insights-append.sh command in the Insight event section)"
      continue
    fi
    if printf '%s\n' "$_block" | grep -Eq -- '--cycle auto([[:space:]]|$)'; then
      pass "$_desc"
    else
      fail "${_desc} (insight-event command lacks --cycle auto)"
    fi
  done
done

printf '\nskill-insight-cycle tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
