# Recipe: Ralph Loop

Autonomous multi-iteration coding with `claude -p` and file-system memory.

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

### Context strategy

Each `claude -p` invocation starts with zero chat history. The prompt instructs the agent to:

1. Read `progress.log` first (cross-iteration memory)
2. Read `task.json` for task metadata
3. Read `AGENTS.md` for project map
4. Check `git status` and `git log` for current state

This means the agent reconstructs context from files each iteration, avoiding stale context accumulation.

## Integration with the operating loop

```
/plan    →  Create plan in docs/plans/active/, select /loop flow
  ↓
/loop    →  Create Git Worktree, initialize Ralph Loop with plan reference
  ↓
Terminal: ./scripts/ralph-loop.sh --verify
  ↓
Return to Claude Code
  ↓
/review  →  Self-review the loop's diff
/verify  →  Spec compliance + static analysis
/test    →  Run behavioral tests
/pr      →  Create PR, archive plan
```

## Tips

- Start with `--max-iterations 5` to calibrate before long runs
- Always use `--verify` for code changes
- Review `progress.log` after the loop finishes — it tells the full story
- If the agent gets stuck, edit `PROMPT.md` with more specific guidance and restart
- For complex tasks, create a plan first (`/plan`) and pass the slug to init

## Customizing prompts

Edit `.harness/state/loop/PROMPT.md` directly after initialization. Common customizations:

- Add specific file paths to investigate
- Add constraints (e.g., "do not modify the public API")
- Add acceptance criteria
- Reference specific plan sections

## Archiving

When you re-initialize a loop, the previous state is automatically archived to `.harness/state/loop-archive/<timestamp>/`.
