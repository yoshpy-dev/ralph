#!/usr/bin/env sh
# Regression guard for the per-slice verification wording: the plan template
# and /plan step 7 ask for each slice's verification command, /implement hands
# that command to the implementer (or picks one for a plan that has none) and
# runs no full ./scripts/run-verify.sh in step 12 or the Completion gate, and
# the Validation Gate, model-routing, and subagent-policy say the same.
# Added by the verify-slices-lighter-self-review plan (issue #261).
#
# Each check reads the file with its line breaks folded into spaces, so
# re-wrapping a sentence does not break it. assert_contains is case-sensitive;
# assert_not_contains ignores case, so an old sentence stays out in either case.

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

# join_lines: stdin on one line, line breaks and runs of spaces as single
# spaces.
join_lines() {
  tr '\n' ' ' | tr -s ' '
}

assert_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if [ ! -f "$_file" ]; then
    record_fail "$_desc (no such file: $_file)"
  elif join_lines < "$_file" | grep -Fq -- "$_needle"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in $_file)"
  fi
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if [ ! -f "$_file" ]; then
    record_fail "$_desc (no such file: $_file)"
  elif join_lines < "$_file" | grep -Fiq -- "$_needle"; then
    record_fail "$_desc (unexpected: $_needle in $_file)"
  else
    record_pass "$_desc"
  fi
}

# block FILE START STOP: the lines from the first one that starts with START
# up to, not including, the next one that starts with STOP (fixed strings).
block() {
  awk -v start="$2" -v stop="$3" '
    !inside && index($0, start) == 1 { inside = 1; print; next }
    inside && index($0, stop) == 1 { exit }
    inside { print }
  ' "$1"
}

# assert_block_contains DESC FILE START STOP NEEDLE: NEEDLE is in that block
# (one step, bullet, or section), not only somewhere else in the file.
assert_block_contains() {
  _desc="$1"
  _file="$2"
  _start="$3"
  _stop="$4"
  _needle="$5"
  if [ ! -f "$_file" ]; then
    record_fail "$_desc (no such file: $_file)"
  elif block "$_file" "$_start" "$_stop" | join_lines | grep -Fq -- "$_needle"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in the block from \"$_start\" in $_file)"
  fi
}

# AC1: the plan template asks for a verification command per slice.
plan_template_files="
docs/plans/templates/feature-plan.md
templates/base/docs/plans/templates/feature-plan.md
.claude/skills/plan/template.md
.agents/skills/plan/template.md
templates/base/.claude/skills/plan/template.md
templates/base/.agents/skills/plan/template.md
"

for file in $plan_template_files; do
  assert_contains "$file shows a slice item that ends with its verify command" \
    "1. <slice> — verify: \`<command>\`" "$file"
  assert_contains "$file says not to narrow with RALPH_VERIFY_SCOPE=changed" \
    "Do not narrow with RALPH_VERIFY_SCOPE=changed" "$file"
  assert_contains "$file wraps gofmt -l in test -z" \
    "test -z \"\$(gofmt -l" "$file"
done

# AC2: /plan step 7 asks for each slice's verification command.
plan_skill_files="
.claude/skills/plan/SKILL.md
.agents/skills/plan/SKILL.md
templates/base/.claude/skills/plan/SKILL.md
templates/base/.agents/skills/plan/SKILL.md
"

for file in $plan_skill_files; do
  assert_block_contains "$file step 7 asks for each slice's verification command" \
    "$file" "7. Fill in:" "8. " \
    "implementation outline: one item per slice, each ending with that slice's verification command"
done

# AC3: /implement takes the slice command from the plan (D4 when there is
# none), inline slices do the mechanical self-check (D6), and step 12 and the
# Completion gate run no full run-verify.sh.
implement_skill_files="
.claude/skills/implement/SKILL.md
.agents/skills/implement/SKILL.md
templates/base/.claude/skills/implement/SKILL.md
templates/base/.agents/skills/implement/SKILL.md
"

step6="6. Implement in small slices"
step7="7. **Commit gate"
inline_start="   - **Inline slices**"
inline_stop="   - If verification fails"

for file in $implement_skill_files; do
  assert_block_contains "$file step 6 takes the slice command from the plan" \
    "$file" "$step6" "$step7" \
    "The exact verification commands are the slice's command from the plan's \`## Implementation outline\`"
  assert_block_contains "$file step 6 covers a plan with no command for the slice" \
    "$file" "$step6" "$step7" \
    "If the plan has no command for the slice (a plan approved before the template asked for one)"
  assert_block_contains "$file step 6 writes the picked command under Progress checklist" \
    "$file" "$step6" "$step7" \
    "pick the narrowest command that covers the slice's files and write it under the plan's \`## Progress checklist\` before dispatching"

  assert_block_contains "$file inline slices run the plan's command" \
    "$file" "$inline_start" "$inline_stop" \
    "run the slice's verification command from the plan (step 6)"
  assert_block_contains "$file inline slices do the mechanical self-check" \
    "$file" "$inline_start" "$inline_stop" \
    "then do the same mechanical self-check the implementer does before it commits"
  assert_block_contains "$file inline slices note the self-check under Progress checklist" \
    "$file" "$inline_start" "$inline_stop" \
    "and note the result under the plan's \`## Progress checklist\`"

  assert_block_contains "$file step 12 does not run the full run-verify.sh" \
    "$file" "12. Before presenting completion" "13. " \
    "Do not run the full \`./scripts/run-verify.sh\` here"
  assert_block_contains "$file Completion gate checks each slice's command" \
    "$file" "## Completion gate" "## " \
    "- [ ] Every slice's verification command passed"

  assert_not_contains "$file drops run-verify.sh as the inline slice check" \
    "run \`./scripts/run-verify.sh\` (or equivalent)" "$file"
  assert_not_contains "$file drops the full run-verify.sh from step 12" \
    "Before presenting completion, run \`./scripts/run-verify.sh\`" "$file"
  assert_not_contains "$file drops run-verify.sh from the Completion gate" \
    "\`./scripts/run-verify.sh\` exits 0" "$file"
done

# AC4: the Validation Gate, model-routing, and subagent-policy match.
commit_strategy_files="
.claude/rules/ralph/git-commit-strategy.md
templates/base/.claude/rules/ralph/git-commit-strategy.md
"

for file in $commit_strategy_files; do
  assert_block_contains "$file Validation Gate runs the slice's command from the plan" \
    "$file" "## Validation Gate" "## " \
    "2. Run the slice's verification command from the plan: the narrowest check that covers the files the slice touches"
  assert_block_contains "$file Validation Gate leaves the full checks to /verify and /test" \
    "$file" "## Validation Gate" "## " \
    "The full static and test checks run later, in \`/verify\` and \`/test\`"
  assert_not_contains "$file drops run-verify.sh as the slice check" \
    "Run \`./scripts/run-verify.sh\` (or equivalent)" "$file"
done

model_routing_files="
.claude/rules/ralph/model-routing.md
templates/base/.claude/rules/ralph/model-routing.md
"

for file in $model_routing_files; do
  assert_contains "$file delegation prompt asks for the mechanical self-check result" \
    "report contract (changed files, key decisions, verification evidence, the mechanical self-check result)" "$file"
  assert_contains "$file report contract lists the mechanical self-check" \
    "**Report contract (implementer → orchestrator):** changed files, decisions/deviations, verification evidence, the mechanical self-check result (unnecessary changes, typos, debug leftovers, secrets, files outside the handoff)" "$file"
  assert_contains "$file handoff row points at the plan's Implementation outline" \
    "| Exact verification commands | the slice's command from the plan's \`## Implementation outline\`" "$file"
  assert_not_contains "$file drops the bare copy-paste handoff row" \
    "| Exact verification commands | copy-paste ready |" "$file"
done

subagent_policy_files="
.claude/rules/ralph/subagent-policy.md
templates/base/.claude/rules/ralph/subagent-policy.md
"

for file in $subagent_policy_files; do
  assert_contains "$file handoff takes the slice command from the plan" \
    "exact verification commands (the slice's command from the plan's \`## Implementation outline\`)" "$file"
  assert_contains "$file implementer report includes the mechanical self-check" \
    "The implementer returns: changed files, decisions/deviations, verification evidence, the mechanical self-check result, commit-boundary evidence, and commit SHA." "$file"
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
