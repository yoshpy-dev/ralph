# Recipe: Ralph Loop

Autonomous multi-iteration coding with `claude -p` and file-system memory.

## Flow overview

| | 標準フロー (`/work`) | Ralph Loop (`/loop`) |
|---|---|---|
| **トリガー** | `/work` skill | `/loop` skill → ターミナルで `ralph run` |
| **実装** | Claude Code セッション内で対話的 | `claude -p` で自律実行 × N slice |
| **ブランチ** | `git checkout -b` | `git worktree add` × N |
| **post-impl 実行モデル** | subagent Task calls (`reviewer`, `verifier`, `tester`, `doc-maintainer`) | `claude -p` × 専用プロンプト (`pipeline-self-review.md`, `pipeline-verify.md`, `pipeline-test.md`, `pipeline-outer.md`) |
| **パイプライン順序** | `/self-review` → `/verify` → `/test` → `/sync-docs` → `/codex-review` → `/pr` | 同一 |
| **レポート出力** | `docs/reports/` | `docs/reports/` + `.harness/state/pipeline/` (dual-write) |
| **ユースケース** | 短〜中規模、対話的 | 大規模、分割可能、並列自律 |

### Decision flow

```
/plan (フロー選択)
  ├── 標準フロー → /work → 対話的実装 → subagent pipeline → /pr
  └── Ralph Loop → /loop → セットアップ → ターミナルで ralph run
        → orchestrator: worktree × N → pipeline × N (parallel)
        → integration merge → unified PR
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
# 1. Create a directory-based plan with slices
./scripts/new-ralph-plan.sh <slug> [issue] [slice-count]

# 2. Edit the plan: _manifest.md + slice-*.md files
$EDITOR docs/plans/active/<date>-<slug>/

# 3. Set up via /loop skill in Claude Code (or manually via ralph-loop-init.sh)

# 4. Run the orchestrator
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/ --unified-pr

# 5. Check results
./scripts/ralph status
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
| Slice timeout | `RALPH_SLICE_TIMEOUT` seconds per slice (default 1800 = 30 min) |
| Signal handling | Separate INT/TERM and EXIT traps with `_INTERRUPTED` flag for clean signal/exit discrimination |
| Numeric validation | All numeric config values validated at startup |
| Verification | `--verify` flag runs `run-verify.sh` after each iteration |
| Prompt rules | Safety rules embedded in every template (no sudo, no force push) |

### Configuration via environment variables

All Ralph pipeline settings are centralized in `scripts/ralph-config.sh`. Override any default via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RALPH_MODEL` | `claude-opus-4-7` | Claude model name |
| `RALPH_EFFORT` | `xhigh` | Effort level for `claude -p` |
| `RALPH_PERMISSION_MODE` | `bypassPermissions` | Permission mode for `claude -p` |
| `RALPH_MAX_ITERATIONS` | `20` | Total iteration cap across all cycles |
| `RALPH_MAX_INNER_CYCLES` | `10` | Max Inner Loop cycles before escalation |
| `RALPH_MAX_OUTER_CYCLES` | `2` | Max Outer Loop cycles (total pipeline runs) before escalation |
| `RALPH_MAX_REPAIR_ATTEMPTS` | `5` | Max fix attempts per failing test |
| `RALPH_MAX_PARALLEL` | `4` | Max concurrent worktree pipelines |
| `RALPH_SLICE_TIMEOUT` | `1800` | Per-slice timeout in seconds |
| `RALPH_STANDARD_MAX_PIPELINE_CYCLES` | `2` | (Standard flow only) Max post-implementation pipeline runs before requiring user confirmation |

Priority: CLI argument > environment variable > default value.

Example:
```sh
RALPH_MODEL=sonnet RALPH_SLICE_TIMEOUT=3600 ./scripts/ralph run --plan <dir> --unified-pr
```

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
  - Integration pipeline on merged branch (--skip-pr --fix-all)
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
  implement (claude -p) → self-review (claude -p) → verify (claude -p) → test (claude -p)
  → if tests fail: retry (up to --max-inner-cycles)

Outer Loop (after tests pass):
  sync-docs (claude -p) → codex-review → PR (claude -p)
  → if codex ACTION_REQUIRED: regress to Inner Loop
```

Each post-implementation agent (self-review, verify, test) runs as a dedicated `claude -p` invocation with a single-responsibility prompt. Agents execute scripts internally (e.g., `run-static-verify.sh`, `run-test.sh`) and produce structured analysis — not just exit codes. Reports are dual-written to `.harness/state/pipeline/` and `docs/reports/`, with machine-readable sidecar signal files for pass/fail detection.

### When to use Ralph Loop

- Large-scale features or refactors that can be split into independent slices
- Test coverage campaigns across many files
- Migration work (dependency, framework, API)
- When you want the full cycle handled autonomously without returning to Claude Code

## Archiving

When you re-initialize a loop, the previous state is automatically archived to `.harness/state/loop-archive/<timestamp>/`.
