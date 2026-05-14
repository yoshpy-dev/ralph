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

### Step 1 — Context survey

Read `AGENTS.md` and scan `docs/plans/active/` to understand the current project state.

### Step 2 — Task type selection

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

### Step 3 — Confirm objective and plan directory

Use **AskUserQuestion** to confirm the objective and link the plan directory.

- Pre-fill the question with an objective inferred from conversation context.
- If `docs/plans/active/` contains directory-based plans (with `_manifest.md`), list them as options.
- Ralph Loop requires a directory-based plan. If none exists, instruct the user to create one with `./scripts/new-ralph-plan.sh --type <type> <slug> [issue] [slice-count]`.

### Step 3.5 — Git worktree creation

Create an isolated worktree for the loop session:

1. Read the active plan to extract metadata (type, issue number, slug).
2. Determine branch name by running `./scripts/branch-name.sh from-plan <manifest-path>`.
   Branch names must validate with `./scripts/branch-name.sh validate <branch-name>`.
3. Run `git worktree add .claude/worktrees/<slug> -b <branch-name>` to create the worktree.
4. Update the plan file: replace `Branch: TBD` (or any TBD variant) with the actual branch name.
5. All subsequent steps (init script, PROMPT.md generation, etc.) execute inside the worktree directory.

If already on a feature branch (not main/master), skip worktree creation and work in-place.

### Step 4 — Run init script

Run the init script with the confirmed parameters:
```sh
./scripts/ralph-loop-init.sh <task-type> "<objective>" <plan-directory>
```

### Step 5 — Approve PROMPT.md

Read the generated `.harness/state/loop/PROMPT.md` and display its contents. Then use **AskUserQuestion** to get approval:

- Options:
  1. **Proceed as-is** — run with the current PROMPT.md
  2. **Adjust** — user provides edits; apply them to PROMPT.md and re-display for confirmation
  3. **Cancel** — abort the loop setup
- If the user chooses "Adjust", edit PROMPT.md per their instructions, then re-ask for approval.

### Step 6 — Print run command

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

The orchestrator handles everything autonomously (parallel pipeline per slice → integration merge → **integration pipeline** (`--skip-pr --fix-all`) → unified PR). The integration pipeline runs `ralph-pipeline.sh` on the merged branch following the canonical post-implementation order (`/self-review` → `/verify` → `/test` → `/sync-docs` → `/cross-review`) to catch cross-module issues and fix ALL findings before PR creation. When the user returns:

1. Run `./scripts/ralph status` to check outcome
2. Read `.harness/state/orchestrator/orchestrator.json` for final state
3. If all slices are `complete` and the unified PR was created — show the PR URL.
4. If any slice is `stuck`, `repair_limit`, or `aborted` — review the failure context and help the user decide next steps (resume, abort, or manual intervention).
5. The orchestrator already creates the PR, so no further post-implementation pipeline is needed.
6. If worktrees were created, ask the user whether to keep or remove them.

## Anti-bottleneck

When presenting AskUserQuestion choices, always pre-select or recommend the most likely option based on conversation context and the active plan. This minimizes user effort. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

### Pipeline prompts (used by `ralph-pipeline.sh` as `claude -p` inputs)

Each prompt is a standalone `claude -p` invocation — the Ralph Loop equivalent of a subagent in `/work`:

| Prompt | Phase | Equivalent `/work` subagent |
|--------|-------|-----------------------------|
| [pipeline-inner.md](prompts/pipeline-inner.md) | Implementation (Inner Loop) | — (interactive in `/work`) |
| [pipeline-self-review.md](prompts/pipeline-self-review.md) | Self-review (Inner Loop) | `reviewer` |
| [pipeline-verify.md](prompts/pipeline-verify.md) | Verify (Inner Loop) | `verifier` |
| [pipeline-test.md](prompts/pipeline-test.md) | Test (Inner Loop) | `tester` |
| [pipeline-outer.md](prompts/pipeline-outer.md) | Sync-docs (Outer Loop) | `doc-maintainer` |

### Scripts
- `scripts/ralph-orchestrator.sh` — Multi-worktree parallel orchestrator
- `scripts/ralph-pipeline.sh` — Per-slice pipeline (Inner/Outer Loop)
- `scripts/ralph` — CLI wrapper (plan/run/status/abort)

### Other
- [Recipe: Ralph Loop](../../../docs/recipes/ralph-loop.md)

## CLI execution modes

This skill runs under both Claude Code and Codex. The execution mode follows
the conventions in AGENTS.md and `.codex/AGENTS.override.md`.

| Aspect | Claude Code | Codex |
|--------|-------------|-------|
| Skill invocation | `/skill-name` slash command | `$skill-name` mention or the `/skills` menu (avoid the `/skill-name` form — it collides with built-ins) |
| Skill body path | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Subagents | Parallel calls via `Task(subagent_type=...)` | Sequential inline execution — chained within a single agent |
| Structured prompts | `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit reply |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

### Loop driver selection (Phase 2 / issue #44)

The CLI invoked by `ralph-pipeline.sh` per slice is controlled by a **driver**.
Selection has two layers:

1. `[loop] driver = "claude" | "codex"` in `ralph.toml` (static default; **honoured only via the Go binary `ralph run`**)
2. Environment variable `RALPH_LOOP_DRIVER=claude|codex` (runtime override, takes precedence over TOML, and also works through the shell wrapper `./scripts/ralph`)

The shell wrappers (`./scripts/ralph`, `scripts/ralph-orchestrator.sh`) do not
read `ralph.toml`, so if you want to enable the Codex driver via TOML only,
either use the Go binary `ralph run` or export `RALPH_LOOP_DRIVER` from your
shell profile / direnv. `ralph doctor` prints the effective value and its
source (`env` / `toml` / `default`) on one line, so you can confirm which
layer took effect.

Under the Codex driver, `RALPH_CODEX_SANDBOX` (default `workspace-write`) and
`RALPH_CODEX_APPROVAL_POLICY` (default `on-failure`) are also reported.

Example of running the flow under the Codex driver:

```sh
codex trust .                                   # one-time
ralph doctor                                    # confirm "codex" on the Loop driver line
RALPH_LOOP_DRIVER=codex ./scripts/ralph run \
  --plan docs/plans/active/<date>-<slug>/ --unified-pr
```

`/cross-review` always uses the **opposite** CLI as the reviewer. When
driver=claude, the reviewer is the existing `codex exec review`; when
driver=codex, the reviewer is `claude -p --permission-mode auto` invoked with
`.claude/skills/cross-review/prompts/adversarial-claude.md`. The `Driver:` /
`Reviewer:` header in the triage report and the `driver` / `reviewer` fields
in the `report_event "cross-review"` JSONL record which pair actually ran.

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
