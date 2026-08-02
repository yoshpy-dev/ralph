#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BRANCH_NAME="${PROJECT_ROOT}/scripts/branch-name.sh"

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

assert_exit() {
  _desc="$1"
  _expected="$2"
  shift 2
  _total=$((_total + 1))
  set +e
  "$@" >/dev/null 2>&1
  _actual="$?"
  set -e
  if [ "$_expected" -eq "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s (expected exit %s, got %s)\n' "$_desc" "$_expected" "$_actual"
  fi
}

_tmp="$(mktemp -d)"

mkdir -p "$_tmp/docs/plans/active"
cat > "$_tmp/docs/plans/active/2026-05-14-docs-branch-policy.md" <<'MD'
# docs-branch-policy

- Status: Draft
- Related issue: #123
- Type: docs
- Branch: TBD
MD

mkdir -p "$_tmp/docs/plans/active/2026-05-14-fix-loop-policy"
cat > "$_tmp/docs/plans/active/2026-05-14-fix-loop-policy/_manifest.md" <<'MD'
# fix-loop-policy

- Status: Draft
- Related issue: N/A
- Type: fix
- Branch: TBD
MD

cat > "$_tmp/docs/plans/active/2026-05-14-missing-type.md" <<'MD'
# missing-type

- Status: Draft
- Related issue: N/A
- Branch: TBD
MD

printf '==> branch-name.sh generation\n'
_out="$("$BRANCH_NAME" from-plan "$_tmp/docs/plans/active/2026-05-14-docs-branch-policy.md")"
assert_eq "file plan with issue" "docs/123/docs-branch-policy" "$_out"

_out="$("$BRANCH_NAME" from-plan "$_tmp/docs/plans/active/2026-05-14-fix-loop-policy")"
assert_eq "directory plan without issue" "fix/fix-loop-policy" "$_out"

assert_exit "missing Type fails closed" 1 "$BRANCH_NAME" from-plan "$_tmp/docs/plans/active/2026-05-14-missing-type.md"
assert_eq "branch type from branch name" "fix" "$("$BRANCH_NAME" type "fix/50/pr-ready-check")"
assert_eq "title prefix from branch name" "docs:" "$("$BRANCH_NAME" title-prefix "docs/update-readme")"
assert_exit "title prefix rejects invalid branch" 1 "$BRANCH_NAME" title-prefix "codex-update-readme"

printf '==> branch-name.sh validation\n'
for branch in \
  feat/add-login \
  fix/50/pr-ready-check \
  docs/update-readme \
  chore/rotate-hooks \
  refactor/split-runner \
  test/branch-name-script \
  ci/update-actions \
  build/package-cli \
  perf/speed-status \
  release/v1.2.3 \
  security/secret-scan
do
  assert_exit "valid branch: ${branch}" 0 "$BRANCH_NAME" validate "$branch"
done

for branch in \
  issue-58-pr-linked-issue-keywords \
  codex-scoped-verify-test \
  integration/ralph-loop \
  slice/plan/1-task \
  feat \
  feat/
do
  assert_exit "invalid branch: ${branch}" 1 "$BRANCH_NAME" validate "$branch"
done

printf '==> plan creation type metadata\n'
_project="$_tmp/project"
mkdir -p "$_project/scripts" "$_project/docs/plans/templates" "$_project/docs/plans/active"
cp "$PROJECT_ROOT/scripts/branch-name.sh" "$_project/scripts/branch-name.sh"
cp "$PROJECT_ROOT/scripts/new-feature-plan.sh" "$_project/scripts/new-feature-plan.sh"
cp "$PROJECT_ROOT/docs/plans/templates/feature-plan.md" "$_project/docs/plans/templates/feature-plan.md"
chmod +x "$_project/scripts/"*.sh

_today="$(date '+%Y-%m-%d')"
(
  cd "$_project"
  ./scripts/new-feature-plan.sh --type docs typed-feature 77 >/dev/null
)
_feature_plan="$_project/docs/plans/active/${_today}-typed-feature.md"
assert_eq "new-feature-plan writes Type" "docs" "$(awk '/^- Type:/ { print $3; exit }' "$_feature_plan")"
assert_eq "new-feature-plan branch output" "docs/77/typed-feature" "$("$BRANCH_NAME" from-plan "$_feature_plan")"
assert_exit "new-feature-plan missing --type value fails" 1 sh -c 'cd "$1" && ./scripts/new-feature-plan.sh --type' sh "$_project"

printf '\nbranch-name tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
