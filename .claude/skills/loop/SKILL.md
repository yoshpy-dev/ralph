---
name: loop
description: Initialize a Ralph Loop session for autonomous parallel-slice execution. Creates a directory-based plan and runs ralph-orchestrator.sh for multi-worktree parallel pipelines with unified PR. Invoke automatically when a task benefits from sustained autonomous iteration outside Claude Code.
allowed-tools: Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion
---
Set up a Ralph Loop for autonomous parallel-slice execution outside Claude Code.

## Goals

- Turn a task into a self-contained parallel pipeline that runs autonomously
- Set up a directory-based plan with slices for parallel execution
- Leave the user ready to start the orchestrator from their terminal

## Steps

### Step 1 — コンテキスト把握

Read `AGENTS.md` and scan `docs/plans/active/` to understand the current project state.

### Step 2 — タスクタイプ選択

Use **AskUserQuestion** to let the user pick a task type.

- Options: `general` / `refactor` / `test-coverage` / `bugfix` / `docs` / `migration`
- If the task type can be inferred from the conversation context or the active plan, place that option first with `(Recommended)` appended to the label.
- Descriptions for each option:
  - general — default for most tasks
  - refactor — restructuring without behaviour change
  - test-coverage — adding or improving tests
  - bugfix — diagnosing and fixing a bug
  - docs — documentation updates
  - migration — language, framework, or API migration

### Step 3 — 目的と計画ディレクトリの確認

Use **AskUserQuestion** to confirm the objective and link the plan directory.

- Pre-fill the question with an objective inferred from conversation context.
- If `docs/plans/active/` contains directory-based plans (with `_manifest.md`), list them as options.
- Ralph Loop requires a directory-based plan. If none exists, instruct the user to create one with `./scripts/new-ralph-plan.sh <slug> [issue] [slice-count]`.

### Step 3.5 — Git Worktree 作成

Create an isolated worktree for the loop session:

1. Read the active plan to extract metadata (type, issue number, slug).
2. Determine branch name: `<type>/<issue>/<slug>` (with issue) or `<type>/<slug>` (without issue).
3. Run `git worktree add .claude/worktrees/<slug> -b <branch-name>` to create the worktree.
4. Update the plan file: replace `Branch: TBD` (or any TBD variant) with the actual branch name.
5. All subsequent steps (init script, PROMPT.md generation, etc.) execute inside the worktree directory.

If already on a feature branch (not main/master), skip worktree creation and work in-place.

### Step 4 — init スクリプト実行

Run the init script with the confirmed parameters:
```sh
./scripts/ralph-loop-init.sh <task-type> "<objective>" <plan-directory>
```

### Step 5 — PROMPT.md の承認

Read the generated `.harness/state/loop/PROMPT.md` and display its contents. Then use **AskUserQuestion** to get approval:

- Options:
  1. **このまま実行** — proceed as-is
  2. **調整が必要** — user provides edits; apply them to PROMPT.md and re-display for confirmation
  3. **キャンセル** — abort the loop setup
- If the user chooses "調整が必要", edit PROMPT.md per their instructions, then re-ask for approval.

### Step 6 — 実行コマンドの提示

After approval, print the run command:

```sh
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --unified-pr
# Dry run to verify slice parsing
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --dry-run
# Bounded iterations
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --unified-pr --max-iterations 15
```

## Output

- `.harness/state/loop/PROMPT.md` ready to run
- `.harness/state/loop/task.json` with metadata
- `.harness/state/loop/progress.log` initialized
- Worktree path at `.claude/worktrees/<slug>` (if created)
- Terminal command for the user to start the loop

## After the loop

The orchestrator handles everything autonomously (parallel pipeline per slice → integration merge → unified PR). When the user returns:

1. Run `./scripts/ralph status` to check outcome
2. Read `.harness/state/orchestrator/orchestrator.json` for final state
3. If all slices are `complete` and the unified PR was created — show the PR URL.
4. If any slice is `stuck`, `repair_limit`, or `aborted` — review the failure context and help the user decide next steps (resume, abort, or manual intervention).
5. The orchestrator already creates the PR, so no further post-implementation pipeline is needed.
6. If worktrees were created, ask the user whether to keep or remove them.

## Anti-bottleneck

When presenting AskUserQuestion choices, always pre-select or recommend the most likely option based on conversation context and the active plan. This minimizes user effort. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

### Pipeline prompts
- [prompts/pipeline-inner.md](prompts/pipeline-inner.md) — Implementation agent
- [prompts/pipeline-review.md](prompts/pipeline-review.md) — Self-review + verify + test agent
- [prompts/pipeline-outer.md](prompts/pipeline-outer.md) — Sync-docs + codex-review + PR agent

### Scripts
- `scripts/ralph-orchestrator.sh` — Multi-worktree parallel orchestrator
- `scripts/ralph-pipeline.sh` — Per-slice pipeline (Inner/Outer Loop)
- `scripts/ralph` — CLI wrapper (plan/run/status/abort)

### Other
- [Recipe: Ralph Loop](../../../docs/recipes/ralph-loop.md)
