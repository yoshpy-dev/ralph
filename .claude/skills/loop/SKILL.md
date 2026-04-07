---
name: loop
description: Initialize a Ralph Loop session for autonomous multi-iteration work. Generates PROMPT.md and state files from a task-specific template, then provides instructions to run the loop externally. Invoke automatically when a task benefits from sustained autonomous iteration outside Claude Code.
allowed-tools: Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion
---
Set up a Ralph Loop for autonomous iteration outside Claude Code.

## Goals

- Turn a task into a self-contained loop that runs `cat PROMPT.md | claude -p` repeatedly
- Choose the right prompt template for the task type
- Leave the user ready to start the loop from their terminal

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

### Step 3 — 目的と計画ファイルの確認

Use **AskUserQuestion** to confirm the objective and optionally link a plan file.

- Pre-fill the question with an objective inferred from conversation context.
- If `docs/plans/active/` contains plan files, list them as options (plus "None" for no plan).
- If no plans exist, skip the plan selection and only confirm the objective.

### Step 4 — init スクリプト実行

Run the init script with the confirmed parameters:
```sh
./scripts/ralph-loop-init.sh <task-type> "<objective>" [plan-slug]
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
./scripts/ralph-loop.sh                          # basic
./scripts/ralph-loop.sh --verify                  # with verification
./scripts/ralph-loop.sh --verify --max-iterations 10  # bounded
```

## Output

- `.harness/state/loop/PROMPT.md` ready to run
- `.harness/state/loop/task.json` with metadata
- `.harness/state/loop/progress.log` initialized
- Terminal command for the user to start the loop

## After the loop

When the user returns after running the loop:
1. Read `.harness/state/loop/status` to check outcome
2. Read `.harness/state/loop/progress.log` for what happened
3. Suggest `/review` and `/verify` to validate the results

## Anti-bottleneck

When presenting AskUserQuestion choices, always pre-select or recommend the most likely option based on conversation context and the active plan. This minimizes user effort. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

- [prompts/general.md](prompts/general.md)
- [prompts/refactor.md](prompts/refactor.md)
- [prompts/test-coverage.md](prompts/test-coverage.md)
- [prompts/bugfix.md](prompts/bugfix.md)
- [prompts/docs.md](prompts/docs.md)
- [prompts/migration.md](prompts/migration.md)
- [Recipe: Ralph Loop](../../../docs/recipes/ralph-loop.md)
