#!/usr/bin/env sh
# Regression guard for the per-slice verification wording: the plan template
# and /plan step 7 ask for each slice's verification command, /implement hands
# that command to the implementer (or picks one for a plan that has none) and
# runs no full ./scripts/run-verify.sh in step 12 or the Completion gate, its
# step 7 reads the mechanical self-check result item by item and dispatches a
# slice that stopped without a commit again as a resume, and the Validation
# Gate, model-routing, and subagent-policy say the same.
# Added by the verify-slices-lighter-self-review plan (issue #261).
#
# Each check reads the file with its line breaks folded into spaces, so
# re-wrapping a sentence does not break it. The helper names say the case
# handling: *_exact is case-sensitive, *_any_case ignores case (so an old
# sentence stays out in either case). tests/test-self-review-scope.sh has
# helpers named assert_contains and assert_not_contains whose case handling
# differs; do not copy across under those names.

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

# assert_joined_contains_exact: case-sensitive, on the joined file.
assert_joined_contains_exact() {
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

# assert_joined_lacks_any_case: ignores case, on the joined file.
assert_joined_lacks_any_case() {
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

# assert_block_contains_exact DESC FILE START STOP NEEDLE: case-sensitive;
# NEEDLE is in that block (one step, bullet, or section), not only somewhere
# else in the file.
assert_block_contains_exact() {
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
  assert_joined_contains_exact "$file shows a slice item that ends with its verify command" \
    "1. <slice> — verify: \`<command>\`" "$file"
  assert_joined_contains_exact "$file says not to narrow with RALPH_VERIFY_SCOPE=changed" \
    "Do not narrow with RALPH_VERIFY_SCOPE=changed" "$file"
  assert_joined_contains_exact "$file wraps gofmt -l in test -z" \
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
  assert_block_contains_exact "$file step 7 asks for each slice's verification command" \
    "$file" "7. Fill in:" "8. " \
    "implementation outline: one item per slice, each ending with that slice's verification command"
done

# The five mechanical items, worded as in the canonical list in
# model-routing.md's report contract.
mechanical_items="unnecessary changes, typos and copy-paste errors, debug leftovers, secrets, and files outside the slice"

# AC3: /implement takes the slice command from the plan (D4 when there is
# none), the gate reads the implementer's self-check result, inline slices do
# the mechanical self-check on the staged diff (D6), and step 12 and the
# Completion gate run no full run-verify.sh.
implement_skill_files="
.claude/skills/implement/SKILL.md
.agents/skills/implement/SKILL.md
templates/base/.claude/skills/implement/SKILL.md
templates/base/.agents/skills/implement/SKILL.md
"

step6="6. Implement in small slices"
step7="7. **Commit gate"
delegated_start="   - **Delegated slices**"
inline_start="   - **Inline slices**"
inline_stop="   - If verification fails"

for file in $implement_skill_files; do
  assert_block_contains_exact "$file step 6 takes the slice command from the plan, the narrowest check" \
    "$file" "$step6" "$step7" \
    "The exact verification commands are the slice's command from the plan's \`## Implementation outline\` (the item's \"— verify: \`<command>\`\"), the narrowest check that covers the slice's files."
  assert_block_contains_exact "$file step 6 covers a plan with no command for the slice" \
    "$file" "$step6" "$step7" \
    "If the plan has no command for the slice (a plan approved before the template asked for one)"
  assert_block_contains_exact "$file step 6 writes the picked command under Progress checklist" \
    "$file" "$step6" "$step7" \
    "pick the narrowest command that covers the slice's files and write it under the plan's \`## Progress checklist\` before dispatching (for an inline slice, before implementing it)"
  assert_block_contains_exact "$file step 6 lets a resume hand over uncommitted changes" \
    "$file" "$step6" "$step7" \
    "so the implementer starts from an unambiguous baseline; the uncommitted changes a resume hands over (step 7) are the one exception."

  assert_block_contains_exact "$file delegated gate reads a result per self-check item" \
    "$file" "$delegated_start" "$inline_start" \
    "mechanical self-check result present, with a result for each of its five items and no item found and not fixed"
  assert_block_contains_exact "$file delegated gate fixes an item found and not fixed first" \
    "$file" "$delegated_start" "$inline_start" \
    "If an item is found and not fixed, have it fixed as its own slice (a new handoff and a new commit) before the next slice."
  assert_block_contains_exact "$file delegated gate covers the last slice" \
    "$file" "$delegated_start" "$inline_start" \
    "For the last slice, do it before step 13."
  assert_block_contains_exact "$file delegated gate covers a stop without a commit on a failure or a secret" \
    "$file" "$delegated_start" "$inline_start" \
    "If the implementer stopped without committing (a verification failure it could not fix within scope, or a secret in the staged diff it could not remove within scope), there is no new commit to check"
  assert_block_contains_exact "$file delegated gate takes a secret out of the worktree first" \
    "$file" "$delegated_start" "$inline_start" \
    "First take a secret out of the worktree (edit the value out, or delete a new untracked file that holds it): the session-end and pre-compaction \`wip:\` checkpoints stage everything with \`git add -A\`."
  assert_block_contains_exact "$file delegated gate dispatches a stopped slice again as a resume" \
    "$file" "$delegated_start" "$inline_start" \
    "Then narrow the slice or revise the plan (step 10) if needed, and dispatch the slice again as a resume, with a handoff that names the uncommitted changes the stopped implementer left in the files in scope as the slice's own, so the new implementer takes them over instead of stopping at its check for pre-existing changes."
  assert_block_contains_exact "$file delegated gate discards a dropped file's changes in index and worktree" \
    "$file" "$delegated_start" "$inline_start" \
    "When narrowing drops a file from the slice, discard its changes (\`git restore --staged --worktree -- <file>\` for a tracked file; delete an untracked one) or give them a slice of their own."

  assert_block_contains_exact "$file inline slices run the plan's command" \
    "$file" "$inline_start" "$inline_stop" \
    "run the slice's verification command from the plan (step 6)"
  assert_block_contains_exact "$file inline slices stage, then self-check the staged diff" \
    "$file" "$inline_start" "$inline_stop" \
    "then stage the slice's files and do the same mechanical self-check the implementer does on the staged diff (\`git diff --cached\`): $mechanical_items."
  assert_block_contains_exact "$file inline slices re-run the command after a fix" \
    "$file" "$inline_start" "$inline_stop" \
    "If the check changed anything, re-run the verification command and re-stage."
  assert_block_contains_exact "$file inline slices note each item, a secret by file and line, and take a secret out of the worktree" \
    "$file" "$inline_start" "$inline_stop" \
    "Note the result for each item (clean, fixed, or found and not fixed; name a secret by its file and line, never by its value) under the plan's \`## Progress checklist\`; fix an item found and not fixed as its own slice before the next slice (for the last slice, before step 13). If a secret cannot be removed within scope, do not commit: unstage it, take it out of the worktree as for a delegated slice, and stop. Then commit"
  assert_joined_lacks_any_case "$file drops the self-check before staging" \
    "Then stage the slice's files and commit" "$file"

  assert_block_contains_exact "$file step 12 does not run the full run-verify.sh" \
    "$file" "12. Before presenting completion" "13. " \
    "Do not run the full \`./scripts/run-verify.sh\` here"
  assert_block_contains_exact "$file Completion gate checks each slice's command" \
    "$file" "## Completion gate" "## " \
    "- [ ] Every slice's verification command passed"
  assert_block_contains_exact "$file Completion gate checks /verify and /test" \
    "$file" "## Completion gate" "## " \
    "- [ ] \`/verify\` and \`/test\` (step 13) passed"
  assert_block_contains_exact "$file Completion gate names both kinds of checks" \
    "$file" "## Completion gate" "## " \
    "If a slice's verification or the pipeline's checks have not run, say so explicitly instead of claiming done."
  assert_joined_lacks_any_case "$file drops the ambiguous closing sentence" \
    "If verification has not run, say so explicitly" "$file"

  assert_joined_lacks_any_case "$file drops run-verify.sh as the inline slice check" \
    "run \`./scripts/run-verify.sh\` (or equivalent)" "$file"
  assert_joined_lacks_any_case "$file drops the full run-verify.sh from step 12" \
    "Before presenting completion, run \`./scripts/run-verify.sh\`" "$file"
  assert_joined_lacks_any_case "$file drops run-verify.sh from the Completion gate" \
    "\`./scripts/run-verify.sh\` exits 0" "$file"
done

# AC4: the Validation Gate, model-routing, and subagent-policy match.
commit_strategy_files="
.claude/rules/ralph/git-commit-strategy.md
templates/base/.claude/rules/ralph/git-commit-strategy.md
"

for file in $commit_strategy_files; do
  assert_block_contains_exact "$file Validation Gate runs the slice's command from the plan" \
    "$file" "## Validation Gate" "## " \
    "2. Run the slice's verification command from the plan: the narrowest check that covers the files the slice touches"
  assert_block_contains_exact "$file Validation Gate leaves the full checks to /verify and /test" \
    "$file" "## Validation Gate" "## " \
    "The full static and test checks run later, in \`/verify\` and \`/test\`"
  assert_joined_lacks_any_case "$file drops run-verify.sh as the slice check" \
    "Run \`./scripts/run-verify.sh\` (or equivalent)" "$file"
done

model_routing_files="
.claude/rules/ralph/model-routing.md
templates/base/.claude/rules/ralph/model-routing.md
"

for file in $model_routing_files; do
  assert_joined_contains_exact "$file delegation prompt asks for the mechanical self-check result" \
    "report contract (changed files, key decisions, verification evidence, the mechanical self-check result)" "$file"
  assert_joined_contains_exact "$file report contract asks for the mechanical self-check" \
    "**Report contract (implementer → orchestrator):** changed files, decisions/deviations, verification evidence, the mechanical self-check result, commit-boundary evidence" "$file"
  assert_joined_contains_exact "$file report contract holds the canonical list" \
    "The mechanical self-check has five items, and this is the canonical list that every other mention repeats word for word: $mechanical_items." "$file"
  assert_joined_contains_exact "$file report contract gives a result per item and names a secret by file and line" \
    "The result says, for each item, clean, fixed (what), or found and not fixed (what and why), and names a secret by its file and line, never by its value." "$file"
  assert_joined_contains_exact "$file report contract fixes an item found and not fixed first" \
    "When an item is found and not fixed, the orchestrator has it fixed as its own slice before the next slice (\`/implement\` step 7)." "$file"
  assert_joined_contains_exact "$file report contract covers the last slice and a resume after a secret" \
    "For the last slice, that happens before step 13. A secret that cannot be removed within scope is never committed: the implementer stops without a commit, and the orchestrator first takes the secret out of the worktree, narrows the slice if needed, and dispatches it again as a resume, whose handoff names the uncommitted changes the stopped slice left in the files in scope as the slice's own (\`/implement\` step 7)." "$file"
  assert_joined_lacks_any_case "$file drops the old item wording" \
    "files outside the handoff)" "$file"
  assert_joined_contains_exact "$file handoff row points at the plan's Implementation outline" \
    "| Exact verification commands | the slice's command from the plan's \`## Implementation outline\`" "$file"
  assert_joined_contains_exact "$file handoff table has the resume row" \
    "| Resume (only when dispatching a stopped slice again) | the uncommitted changes the stopped slice left in the files in scope, named as the slice's own (\`/implement\` step 7) |" "$file"
  assert_joined_lacks_any_case "$file drops the bare copy-paste handoff row" \
    "| Exact verification commands | copy-paste ready |" "$file"
done

subagent_policy_files="
.claude/rules/ralph/subagent-policy.md
templates/base/.claude/rules/ralph/subagent-policy.md
"

for file in $subagent_policy_files; do
  assert_joined_contains_exact "$file handoff takes the slice command from the plan" \
    "exact verification commands (the slice's command from the plan's \`## Implementation outline\`)" "$file"
  assert_joined_contains_exact "$file implementer report includes the mechanical self-check" \
    "The implementer returns: changed files, decisions/deviations, verification evidence, the mechanical self-check result, commit-boundary evidence, and commit SHA." "$file"
done

# The Codex README's implementer row repeats the five items word for word.
codex_readme_files="
.codex/README.md
templates/base/.codex/README.md
"

for file in $codex_readme_files; do
  assert_joined_contains_exact "$file implementer row lists the five items in the canonical words" \
    "does a mechanical self-check on the staged diff ($mechanical_items)" "$file"
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
