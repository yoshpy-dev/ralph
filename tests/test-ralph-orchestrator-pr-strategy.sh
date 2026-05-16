#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ORCHESTRATOR="${PROJECT_ROOT}/scripts/ralph-orchestrator.sh"
RALPH="${PROJECT_ROOT}/scripts/ralph"

PASS=0
FAIL=0

pass() {
  PASS=$((PASS + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_contains() {
  _label="$1"
  _needle="$2"
  _file="$3"
  if grep -Fq "$_needle" "$_file"; then
    pass "$_label"
  else
    fail "$_label"
    printf '    missing: %s\n' "$_needle"
  fi
}

assert_not_exists() {
  _label="$1"
  _path="$2"
  if [ ! -e "$_path" ]; then
    pass "$_label"
  else
    fail "$_label"
    printf '    unexpected path exists: %s\n' "$_path"
  fi
}

setup_repo() {
  _tmp="$(mktemp -d "${TMPDIR:-/tmp}/ralph-pr-strategy.XXXXXX")"
  cd "$_tmp"
  git init -q -b main
  git config user.email test@example.com
  git config user.name "Test User"
  printf 'seed\n' > README.md
  git add README.md
  git commit -q -m 'chore: seed'

  plan_dir="docs/plans/active/2026-05-15-grouped"
  mkdir -p "$plan_dir"
  cat > "${plan_dir}/_manifest.md" <<'MD'
# Grouped

- Type: feat
- Related issue: #90

## PR grouping

```toml
pr_strategy = "grouped"

[pr_strategy_decision]
selected = "grouped"
recommended_by = "ai"
human_approved = false
approval_note = ""
rationale = "Grouped PRs are independently reviewable."

[[pr_groups]]
name = "core"
slices = ["slice-1-api"]

[[pr_groups]]
name = "docs-tests"
slices = ["slice-2-docs"]

[[pr_strategy_decision.group_rationale]]
name = "core"
independent = true
depends_on = []
reason = "Core can be reviewed against main."
```
MD
  cat > "${plan_dir}/slice-1-api.md" <<'MD'
# Slice 1

- Objective: API
- Dependencies: none
- Affected files: internal/api.go
MD
  cat > "${plan_dir}/slice-2-docs.md" <<'MD'
# Slice 2

- Objective: Docs
- Dependencies: none
- Affected files: README.md
MD
}

cleanup_repo() {
  cd "$PROJECT_ROOT"
  rm -rf "$_tmp"
}

printf '==> Ralph orchestrator PR strategy tests\n'

setup_repo
trap cleanup_repo EXIT HUP INT TERM

"$ORCHESTRATOR" --plan "$plan_dir" --dry-run > grouped.log
assert_contains "dry-run reports grouped strategy" "[DRY RUN] PR strategy: grouped" grouped.log
assert_contains "dry-run reports decision approval state" "[DRY RUN] PR strategy decision: selected=grouped, human approved=false" grouped.log
assert_contains "dry-run reports core group branch" "PR group core: branch feat/90/grouped-core, slices 1-api" grouped.log
assert_contains "dry-run reports docs group branch" "PR group docs-tests: branch feat/90/grouped-docs-tests, slices 2-docs" grouped.log
assert_not_exists "dry-run does not write orchestrator state" ".harness/state/orchestrator/orchestrator.json"

"$ORCHESTRATOR" --plan "$plan_dir" --dry-run --unified-pr > unified.log
assert_contains "--unified-pr aliases unified strategy" "[DRY RUN] PR strategy: unified" unified.log

if "$ORCHESTRATOR" --plan "$plan_dir" --dry-run --pr-strategy bogus > invalid.log 2>&1; then
  fail "invalid strategy exits non-zero"
else
  pass "invalid strategy exits non-zero"
fi
assert_contains "invalid strategy message" "must be one of grouped, stacked, unified" invalid.log

"$ORCHESTRATOR" --plan "$plan_dir" --dry-run --pr-strategy stacked > stacked-warning.log 2>&1
assert_contains "override mismatch warning includes both strategies" "Runtime --pr-strategy 'stacked' overrides manifest PR strategy 'grouped'." stacked-warning.log
assert_contains "stacked without dependency rationale warns" "Stacked PR strategy selected without dependency rationale" stacked-warning.log

"$RALPH" cleanup --plan "$plan_dir" --dry-run > cleanup.log
assert_contains "cleanup dry-run reports plan" "Cleanup plan: ${plan_dir}" cleanup.log
assert_contains "cleanup dry-run reports integration branch" "Integration branch: feat/90/grouped" cleanup.log

missing_plan="${_tmp}/missing-plan"
mkdir -p .harness/state/orchestrator
cat > .harness/state/orchestrator/orchestrator.json <<JSON
{
  "schema_version": 1,
  "plan": "${missing_plan}",
  "started": "2026-05-13T04:41:39Z",
  "max_parallel": 4,
  "max_iterations": 20,
  "unified_pr": false,
  "status": "running"
}
JSON

"$RALPH" cleanup --stale --older-than 0d --dry-run > stale-missing-dry-run.log
assert_contains "stale cleanup dry-run reports stale state" "Current orchestrator state is stale." stale-missing-dry-run.log
assert_contains "stale cleanup dry-run reports missing plan" "Plan missing: ${missing_plan}" stale-missing-dry-run.log
assert_contains "stale cleanup dry-run reports state removal" "[DRY RUN] Would remove stale orchestrator state: .harness/state/orchestrator" stale-missing-dry-run.log
assert_contains "stale cleanup dry-run skips branch cleanup" "[DRY RUN] Branch cleanup skipped because plan metadata is unavailable." stale-missing-dry-run.log
if [ -f .harness/state/orchestrator/orchestrator.json ]; then
  pass "stale cleanup dry-run preserves orchestrator state"
else
  fail "stale cleanup dry-run preserves orchestrator state"
fi

"$RALPH" cleanup --stale --older-than 0d > stale-missing-cleanup.log
assert_contains "stale cleanup archives missing-plan state" "Archived stale orchestrator state to .harness/state/loop-archive/" stale-missing-cleanup.log
assert_contains "stale cleanup removes missing-plan state" "Removed stale orchestrator state: .harness/state/orchestrator" stale-missing-cleanup.log
assert_contains "stale cleanup skips branch cleanup without metadata" "Branch cleanup skipped because plan metadata is unavailable." stale-missing-cleanup.log
assert_not_exists "stale cleanup removes orchestrator json" ".harness/state/orchestrator/orchestrator.json"

"$RALPH" status --json > stale-status.json
assert_contains "status no longer reports stale run" "\"error\":\"no_active_orchestrator\"" stale-status.json

if "$RALPH" cleanup --plan "$missing_plan" --dry-run > missing-explicit.log 2>&1; then
  fail "explicit missing plan exits non-zero"
else
  pass "explicit missing plan exits non-zero"
fi
assert_contains "explicit missing plan still fails clearly" "cleanup requires a Ralph Loop plan directory with _manifest.md: ${missing_plan}" missing-explicit.log

printf '\nRalph orchestrator PR strategy tests: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
