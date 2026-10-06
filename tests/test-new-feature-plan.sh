#!/usr/bin/env sh
set -eu

# test-new-feature-plan.sh — the plan that scripts/new-feature-plan.sh writes
# from docs/plans/templates/feature-plan.md carries the /plan approval fields
# (`- Approved:`, `## Visual review`, `- [ ] Plan approved`) and works with
# scripts/plan-visual.sh digest.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLAN_VISUAL="${PROJECT_ROOT}/scripts/plan-visual.sh"

_pass=0
_fail=0
_total=0
_tmp=""

cleanup() {
  [ -n "$_tmp" ] && rm -rf "$_tmp"
}
trap cleanup EXIT HUP INT TERM

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  _total=$((_total + 1))
  if [ "$_expected" = "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    expected: %s\n    actual:   %s\n' "$_desc" "$_expected" "$_actual"
  fi
}

assert_true() {
  _desc="$1"
  shift
  _total=$((_total + 1))
  if "$@" >/dev/null 2>&1; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n' "$_desc"
  fi
}

assert_false() {
  _desc="$1"
  shift
  _total=$((_total + 1))
  if "$@" >/dev/null 2>&1; then
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n' "$_desc"
  else
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  fi
}

# line_no <file> <exact line> — number of the first line equal to <exact line>.
line_no() {
  awk -v want="$2" '$0 == want { print NR; exit }' "$1"
}

_tmp="$(mktemp -d)"

printf '==> plan templates stay identical\n'
assert_true "feature-plan.md and .claude/skills/plan/template.md are byte-identical" \
  cmp -s "$PROJECT_ROOT/docs/plans/templates/feature-plan.md" "$PROJECT_ROOT/.claude/skills/plan/template.md"

printf '==> new-feature-plan.sh output\n'
_project="$_tmp/project"
mkdir -p "$_project/scripts" "$_project/docs/plans/templates"
cp "$PROJECT_ROOT/scripts/new-feature-plan.sh" "$_project/scripts/new-feature-plan.sh"
cp "$PROJECT_ROOT/scripts/branch-name.sh" "$_project/scripts/branch-name.sh"
cp "$PROJECT_ROOT/docs/plans/templates/feature-plan.md" "$_project/docs/plans/templates/feature-plan.md"
chmod +x "$_project/scripts/"*.sh

_out="$(cd "$_project" && ./scripts/new-feature-plan.sh --type feat demo-slug 42)"
_rel="${_out#Created }"
_plan="$_project/$_rel"
assert_true "new-feature-plan.sh reports the created plan" test -f "$_plan"
case "$_rel" in
  docs/plans/active/*-demo-slug.md) _name_ok=yes ;;
  *) _name_ok=no ;;
esac
assert_eq "plan path is docs/plans/active/<date>-demo-slug.md" "yes" "$_name_ok"

assert_true "has '- Status: Draft'" grep -qxF -- '- Status: Draft' "$_plan"
assert_true "has '- Approved: TBD'" grep -qxF -- '- Approved: TBD' "$_plan"
assert_true "has '## Visual review'" grep -qxF -- '## Visual review' "$_plan"
assert_true "has '- [ ] Plan approved'" grep -qxF -- '- [ ] Plan approved' "$_plan"
assert_true "title is the slug" grep -qxF -- '# demo-slug' "$_plan"
assert_true "Related request is the slug" grep -qxF -- '- Related request: demo-slug' "$_plan"
assert_true "Type is substituted" grep -qxF -- '- Type: feat' "$_plan"
assert_true "Related issue is substituted" grep -qxF -- '- Related issue: 42' "$_plan"
assert_false "no __PLACEHOLDER__ left" grep -q '__[A-Z]*__' "$_plan"

_status="$(line_no "$_plan" '- Status: Draft')"
_approved="$(line_no "$_plan" '- Approved: TBD')"
assert_eq "'- Approved:' follows '- Status:'" "$((${_status:-0} + 1))" "$_approved"

_affected="$(line_no "$_plan" '## Affected areas')"
_visual="$(line_no "$_plan" '## Visual review')"
_design="$(line_no "$_plan" '## Design decisions')"
assert_true "'## Visual review' comes after '## Affected areas'" test "${_affected:-0}" -lt "${_visual:-0}"
assert_true "'## Visual review' comes before '## Design decisions'" test "${_visual:-0}" -lt "${_design:-0}"
_next_heading="$(awk -v from="$_affected" 'NR > from && /^## / { print NR; exit }' "$_plan")"
assert_eq "'## Visual review' is the heading right after '## Affected areas'" "$_visual" "$_next_heading"

_reviewed="$(line_no "$_plan" '- [ ] Plan reviewed')"
_plan_approved="$(line_no "$_plan" '- [ ] Plan approved')"
assert_eq "'Plan approved' follows 'Plan reviewed'" "$((${_reviewed:-0} + 1))" "$_plan_approved"

printf '==> plan-visual.sh digest on the created plan\n'
_digest="$("$PLAN_VISUAL" digest "$_plan")"
_hex_ok=no
if printf '%s\n' "$_digest" | grep -Eqx '[0-9a-f]{12}'; then
  _hex_ok=yes
fi
assert_eq "digest prints 12 hex chars ($_digest)" "yes" "$_hex_ok"

# What /plan step 12.e writes on approval must not change the digest.
_approved_plan="$_tmp/approved.md"
sed \
  -e 's/^- Status: Draft$/- Status: Approved/' \
  -e "s/^- Approved: TBD\$/- Approved: 2026-10-05 sha256:${_digest}/" \
  -e 's/^- Branch: TBD$/- Branch: feat\/42\/demo-slug/' \
  -e 's/^- \[ \] Plan approved$/- [x] Plan approved/' \
  "$_plan" > "$_approved_plan"
assert_false "approval edits change the file" cmp -s "$_plan" "$_approved_plan"
assert_eq "approval edits keep the digest" "$_digest" "$("$PLAN_VISUAL" digest "$_approved_plan")"

# An edit to the reviewed body (here the Visual review section) must change it.
_edited_plan="$_tmp/edited.md"
awk '{ print } $0 == "## Visual review" { print ""; print "None (docs-only change)" }' "$_approved_plan" > "$_edited_plan"
_edited_digest="$("$PLAN_VISUAL" digest "$_edited_plan")"
assert_false "a Visual review edit changes the digest" test "$_digest" = "$_edited_digest"

printf '\nnew-feature-plan tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
