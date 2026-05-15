# Recipe: Ralph Loop

Autonomous multi-iteration coding with `claude -p` and file-system memory.

## Flow overview

| | 標準フロー (`/work`) | Ralph Loop (`/loop`) |
|---|---|---|
| **トリガー** | `/work` skill | `/loop` skill → ターミナルで `ralph run` |
| **実装** | Claude Code セッション内で対話的 | `claude -p` で自律実行 × N slice |
| **ブランチ / worktree** | clean-base task worktree | task worktree + `git worktree add` × N slice worktrees |
| **post-impl 実行モデル** | subagent Task calls (`reviewer`, `verifier`, `tester`, `doc-maintainer`) | `claude -p` × 専用プロンプト (`pipeline-self-review.md`, `pipeline-verify.md`, `pipeline-test.md`, `pipeline-outer.md`) |
| **パイプライン順序** | `/self-review` → `/verify` → `/test` → `/sync-docs` → `/cross-review` → `/pr` | 同一 |
| **レポート出力** | `docs/reports/` | `docs/reports/` + `.harness/state/pipeline/` (dual-write) |
| **ユースケース** | 短〜中規模、対話的 | 大規模、分割可能、並列自律 |

### Decision flow

```
/plan (clean-base task worktree 作成 + フロー選択)
  ├── 標準フロー → /work → 対話的実装 → subagent pipeline → /pr → task worktree cleanup
  └── Ralph Loop → /loop → セットアップ → ターミナルで ralph run
        → orchestrator: worktree × N → pipeline × N (parallel)
        → integration merge → grouped PRs (default) / unified PR (fallback)
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
# 1. Create or resume a clean-base task worktree via /plan, then create a directory-based plan with slices
./scripts/new-ralph-plan.sh --type <type> <slug> [issue] [slice-count]

# 2. Edit the plan inside the task worktree: _manifest.md + slice-*.md files
$EDITOR docs/plans/active/<date>-<slug>/

# 3. Set up via /loop skill in Claude Code (or manually via ralph-loop-init.sh)

# 4. Run the orchestrator
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/

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
| `RALPH_LOOP_DRIVER` | `claude` | Which driver runs `ralph-pipeline.sh` per slice (`claude` or `codex`). Phase 2 / issue #44. |
| `RALPH_CODEX_SANDBOX` | `workspace-write` | `codex exec -s` value when driver=codex (`read-only` / `workspace-write` / `danger-full-access`) |
| `RALPH_CODEX_APPROVAL_POLICY` | `on-failure` | Codex `approval_policy` override (`untrusted` / `on-failure` / `on-request` / `never`) |
| `RALPH_CLAUDE_REVIEWER_MODEL` | `claude-opus-4-7` | Model used by `claude -p` when it plays adversarial reviewer (driver=codex cross-review path) |

Priority: CLI argument > environment variable > `ralph.toml` > default value. The
loop-driver knobs also accept `[loop] driver = "..."` etc. in `ralph.toml`,
which `ralph run` propagates to the orchestrator only when the env var is
unset.

Example:
```sh
RALPH_MODEL=sonnet RALPH_SLICE_TIMEOUT=3600 ./scripts/ralph run --plan <dir>
```

### Running Loop under the Codex driver

Phase 2 (issue #44) wires `ralph-pipeline.sh` through a driver-aware
dispatcher. To drive a Loop run from Codex:

1. **Trust the project once.** Codex silently ignores `.codex/config.toml`,
   `[features]`, and `[hooks]` until you run `codex trust .` from the repo
   root. Without trust the per-slice pipeline still works but project-level
   hooks won't fire.
2. **Confirm setup.** `ralph doctor` prints `Loop driver: <effective>
   (source: env|toml|default)` so you can verify the switch took. When
   driver=codex it also shows the active `codex_sandbox` and
   `codex_approval_policy`.
3. **Start the orchestrator with the Codex driver.**

   The shell wrapper picks the driver from the **environment only**:

   ```sh
   RALPH_LOOP_DRIVER=codex ./scripts/ralph run \
     --plan docs/plans/active/<date>-<slug>/
   ```

   To make the choice persistent without re-typing the env var, you have two
   options:

   - Use the **Go binary** entrypoint, which reads `ralph.toml` and propagates
     the result to the orchestrator:

     ```toml
     # ralph.toml
     [loop]
     driver = "codex"
     codex_sandbox = "workspace-write"
     codex_approval_policy = "on-failure"
     ```

     ```sh
     ralph run --plan docs/plans/active/<date>-<slug>/
     ```

   - Or export `RALPH_LOOP_DRIVER` in your shell profile / direnv so every
     invocation of `./scripts/ralph` inherits it.

   Important asymmetry: the shell wrapper (`./scripts/ralph` and
   `scripts/ralph-orchestrator.sh`) does **not** parse `ralph.toml` itself.
   `[loop] driver = "codex"` alone, paired with `./scripts/ralph run`, will
   silently fall back to the claude driver. `ralph doctor` reports the
   effective driver and source so you can confirm which knob actually took
   effect.

   The env var always wins when both are set, so a one-shot
   `RALPH_LOOP_DRIVER=claude ./scripts/ralph run ...` easily reverts to Claude
   for a single run.

What changes inside the pipeline:

- Per-slice agent calls go through `codex exec -s <sandbox>
  -c approval_policy=<policy> --output-last-message <log>.last -` (stdin
  prompt). The wrapper synthesises `<log>.json` so existing parsers work
  unchanged.
- Preflight Probe 5 inspects `codex exec --help` to confirm
  `--output-last-message`, `-s`, and `-c` are present; missing flags fail
  the probe instead of silently degrading.
- The cross-review reviewer is **inverted**: with driver=codex the pipeline
  invokes `claude -p --permission-mode auto` against
  `.claude/skills/cross-review/prompts/adversarial-claude.md` so the
  cross-model gate is preserved. The triage report's `Driver:` / `Reviewer:`
  header records which pair ran.

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
/plan    →  Ensure clean-base task worktree and create directory-based plan (docs/plans/active/<date>-<slug>/)
            using ./scripts/new-ralph-plan.sh --type <type> <slug> [issue] [slice-count]
  ↓
/loop    →  Set up the Ralph Loop session from the task worktree
  ↓
Terminal: ./scripts/ralph run --plan docs/plans/active/<date>-<slug>/
  ↓
Orchestrator handles:
  - Uses the task worktree as the control worktree
  - Creates worktree per slice (.claude/worktrees/<slug>)
  - Runs ralph-pipeline.sh in each worktree (parallel where no deps)
  - Uses the manifest PR strategy decision as the source of truth
  - Warns when runtime `--pr-strategy` overrides the recorded decision
  - Sequential merge to typed branch from ./scripts/branch-name.sh from-plan
  - Integration pipeline on merged branch (--skip-pr --fix-all)
  - Grouped PRs from manifest `pr_groups` by default
  - Temporary integration branch cleanup after grouped/stacked PR creation succeeds
  - Unified PR from the typed integration branch only when `pr_strategy = "unified"` or `--pr-strategy unified`
  ↓
Return to Claude Code: check ./scripts/ralph status
```

## Tips

- Start with `--max-iterations 5` to calibrate before long runs
- Always use `--verify` for code changes
- Record the PR strategy decision before running: AI recommends `grouped`, `stacked`, or `unified`; human approval at plan approval time makes it final.
- Use `stacked` only when the manifest records a real dependency chain between PR groups.
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
./scripts/ralph run --plan docs/plans/active/<date>-<slug>/
./scripts/ralph run --plan <dir> --dry-run      # validate setup first
./scripts/ralph run --plan <dir> --max-iterations 15  # bounded
./scripts/ralph run --plan <dir> --pr-strategy unified # fallback single PR
./scripts/ralph status                          # check progress
./scripts/ralph cleanup --plan <dir>             # remove retained diagnostics after inspection
./scripts/ralph abort                           # safely stop and archive state
```

### Inner / Outer Loop architecture (per slice)

```
Inner Loop (per cycle):
  implement (claude -p) → self-review (claude -p) → verify (claude -p) → test (claude -p)
  → if tests fail: retry (up to --max-inner-cycles)

Outer Loop (after tests pass):
  sync-docs (claude -p) → cross-review → PR (claude -p)
  → if cross-review ACTION_REQUIRED: regress to Inner Loop
```

Each post-implementation agent (self-review, verify, test) runs as a dedicated `claude -p` invocation with a single-responsibility prompt. Agents execute scripts internally (e.g., `run-static-verify.sh`, `run-test.sh`) and produce structured analysis — not just exit codes. Reports are dual-written to `.harness/state/pipeline/` and `docs/reports/`, with machine-readable sidecar signal files for pass/fail detection.

### When to use Ralph Loop

- Large-scale features or refactors that can be split into independent slices
- Test coverage campaigns across many files
- Migration work (dependency, framework, API)
- When you want the full cycle handled autonomously without returning to Claude Code

## Archiving

When you re-initialize a loop, the previous state is automatically archived to `.harness/state/loop-archive/<timestamp>/`.
