# Recipe: Ralph Loop

Autonomous multi-iteration coding with `claude -p` and file-system memory.

## Flow overview

| Flow | Trigger | Script | Branch | Post-impl pipeline | Use case |
|------|---------|--------|--------|--------------------|----------|
| 標準フロー | `/work` | (Claude Code session) | `git checkout -b` | subagents | 短〜中規模、対話的 |
| Ralph Loop | `/loop` | `ralph-orchestrator.sh` | `git worktree add` × N | pipeline-internal × N | 大規模、分割可能、並列自律 |

### Decision flow

```
/plan
  ├── "標準フロー (/work)" → /work → 対話的実装 → subagents → /pr
  └── "Ralph Loop (/loop)" → /loop
        directory-based plan → ralph-orchestrator.sh
        → pipeline × N (parallel) → integration merge → unified PR
```

## What is it

The Ralph Loop is a pattern where a shell script repeatedly pipes a prompt file into `claude -p`, letting the agent iterate on a task across many fresh-context invocations. The file system (git, progress logs, state files) serves as the agent's persistent memory.

Named after Geoffrey Huntley's original `while :; do cat PROMPT.md | claude -p; done` pattern.

## When to use

- Greenfield implementation that benefits from sustained autonomous work
- Refactoring across many files where each step should be independently verifiable
- Test coverage campaigns
- Bug hunts that need systematic diagnosis
- Documentation sweeps
- Migration work (dependency, framework, API)

## When NOT to use

- Quick fixes (just use Claude Code directly)
- Tasks requiring heavy human judgment at each step
- Work that needs hooks for safety (hooks do not fire in `-p` mode)

## Quick start

```sh
# 1. Initialize the loop
./scripts/ralph-loop-init.sh general "Implement user authentication"

# 2. Review the generated prompt
cat .harness/state/loop/PROMPT.md

# 3. Run the loop
./scripts/ralph-loop.sh --verify --max-iterations 10

# 4. Check results
cat .harness/state/loop/status
cat .harness/state/loop/progress.log
```

## Using the /loop skill

Inside Claude Code, run `/loop` to interactively set up a Ralph Loop session. The skill will:

1. Determine the right task type (general, refactor, test-coverage, bugfix, docs, migration)
2. Run the init script
3. Let you review and customize the PROMPT.md
4. Give you the terminal command to start

## Task types

| Type | Template | Best for |
|------|----------|----------|
| `general` | General-purpose iteration | Most tasks |
| `refactor` | Behaviour-preserving restructuring | Code reorganization |
| `test-coverage` | Adding and improving tests | Coverage campaigns |
| `bugfix` | Diagnose-first, fix-second | Bug investigation |
| `docs` | Code-verified documentation | Doc sweeps |
| `migration` | Backward-compatible migration steps | Upgrades |

## How it works

### Flow

```
ralph-loop-init.sh
  ├── Archives previous loop state (if any)
  ├── Selects prompt template by task type
  ├── Generates PROMPT.md with objective injected
  └── Creates task.json, progress.log, status

ralph-loop.sh
  └── while iteration < max:
        ├── cat PROMPT.md | claude -p | tee iteration-NNN.log
        ├── Check for <promise>COMPLETE</promise> → stop
        ├── Check for <promise>ABORT</promise> → stop
        ├── Stuck detection (3x empty git diff) → stop
        └── Optional: run-verify.sh
```

### State directory

All loop state lives in `.harness/state/loop/`:

| File | Purpose |
|------|---------|
| `PROMPT.md` | Piped to `claude -p` each iteration |
| `task.json` | Task metadata (objective, type, plan, timestamps) |
| `progress.log` | Append-only log — the agent's cross-iteration memory |
| `status` | Current state: pending, running, complete, stuck, aborted, max_iterations |
| `stuck.count` | Consecutive no-change counter |
| `iteration-NNN.log` | Full output of each iteration |

### Safety rails

| Rail | Mechanism |
|------|-----------|
| Iteration limit | `--max-iterations N` (default 20) |
| Stuck detection | 3 consecutive iterations with no git diff → auto-stop |
| Completion gate | Agent must output `<promise>COMPLETE</promise>` explicitly |
| Abort signal | Agent can output `<promise>ABORT</promise>` when blocked |
| Verification | `--verify` flag runs `run-verify.sh` after each iteration |
| Prompt rules | Safety rules embedded in every template (no sudo, no force push) |

### Commit verification

The orchestrator monitors uncommitted changes after each iteration:

- After every iteration, `git status --porcelain` is checked
- If uncommitted changes are detected, a warning is logged to stdout and `progress.log`
- Warnings use the `> [orchestrator]` prefix for easy filtering
- The loop does NOT stop on uncommitted changes (advisory only)
- A summary of uncommitted warnings is printed at the end of the loop

This ensures the agent's commit behavior is visible and auditable without blocking progress.

### Context strategy

Each `claude -p` invocation starts with zero chat history. The prompt instructs the agent to:

1. Read `progress.log` first (cross-iteration memory)
2. Read `task.json` for task metadata
3. Read `AGENTS.md` for project map
4. Check `git status` and `git log` for current state

This means the agent reconstructs context from files each iteration, avoiding stale context accumulation.

## Integration with the operating loop

```
/plan    →  Create directory-based plan (docs/plans/active/<date>-<slug>/)
            using ./scripts/new-ralph-plan.sh <slug> [issue] [slice-count]
  ↓
/loop    →  Set up the Ralph Loop session
  ↓
Terminal: ./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --unified-pr
  ↓
Orchestrator handles:
  - Creates worktree per slice (.claude/worktrees/<slug>)
  - Runs ralph-pipeline.sh in each worktree (parallel where no deps)
  - Sequential merge to integration/<slug> branch
  - Unified PR from integration branch
  ↓
Return to Claude Code: check ./scripts/ralph status
```

## Tips

- Start with `--max-iterations 5` to calibrate before long runs
- Always use `--verify` for code changes
- Review `progress.log` after the loop finishes — it tells the full story
- If the agent gets stuck, edit `PROMPT.md` with more specific guidance and restart
- For complex tasks, create a plan first (`/plan`) and pass the slug to init
- The orchestrator checks for uncommitted changes after each iteration — if you see warnings in the summary, review `progress.log` for details and consider adding more specific commit instructions to `PROMPT.md`

## Customizing prompts

Edit `.harness/state/loop/PROMPT.md` directly after initialization. Common customizations:

- Add specific file paths to investigate
- Add constraints (e.g., "do not modify the public API")
- Add acceptance criteria
- Reference specific plan sections

## Pipeline architecture

Each slice in the Ralph Loop runs a full Inner/Outer Loop pipeline autonomously:

```sh
# Use the ralph CLI
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --unified-pr
./scripts/ralph run --plan <dir> --dry-run      # validate setup first
./scripts/ralph run --plan <dir> --unified-pr --max-iterations 15  # bounded
./scripts/ralph status                          # check progress
./scripts/ralph abort                           # safely stop and archive state
```

### Inner / Outer Loop architecture (per slice)

```
Inner Loop (per cycle):
  implement → self-review → verify → test
  → if tests fail: retry (up to --max-inner-cycles)

Outer Loop (after tests pass):
  sync-docs → codex-review → PR
  → if codex ACTION_REQUIRED: regress to Inner Loop
```

### When to use Ralph Loop

- Large-scale features or refactors that can be split into independent slices
- Test coverage campaigns across many files
- Migration work (dependency, framework, API)
- When you want the full cycle handled autonomously without returning to Claude Code

## Archiving

When you re-initialize a loop, the previous state is automatically archived to `.harness/state/loop-archive/<timestamp>/`.
