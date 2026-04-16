# harness-engineering-scaffolding-template

A language-agnostic scaffold for practicing harness engineering with **Claude Code first** and **cross-vendor portability second**.

This repository is intentionally designed as a **map, not a manual**:
- `AGENTS.md` is the vendor-neutral map for any coding agent.
- `CLAUDE.md` imports `AGENTS.md` and adds Claude Code specific guidance.
- `.claude/rules/` keeps conditional guidance out of the always-on context.
- `.claude/skills/` provides on-demand workflows for plan, work, self-review, verify, and harness auditing.
- `.claude/hooks/` adds deterministic runtime guardrails where instructions alone are not enough.
- `packs/languages/` provides opt-in language specializations without hard-coding the core scaffold to one stack.

## Why this scaffold exists

Strong harnesses are not just prompts. They combine:
1. **A small always-on map**
2. **On-demand workflows**
3. **Deterministic checks on the execution path**
4. **Evidence-backed review and verification**
5. **Optional escalation to subagents, worktrees, or agent teams when the task truly needs them**

The default philosophy here is:

- Start simple
- Add complexity only when the model or task demands it
- Keep plans, decisions, tech debt, and evidence in the repo
- Promote recurring mistakes from prose into scripts, rules, tests, hooks, or CI

## What is inside

```text
.
├── AGENTS.md
├── CLAUDE.md
├── .claude/
│   ├── settings.json
│   ├── hooks/
│   ├── skills/
│   ├── agents/
│   └── rules/
├── cmd/
│   └── ralph-tui/          # TUI entrypoint (Go)
├── internal/
│   ├── state/              # Pipeline state reader
│   ├── watcher/            # File system watcher
│   ├── ui/                 # Bubble Tea TUI components
│   └── action/             # CLI action executor
├── docs/
│   ├── research/
│   ├── architecture/
│   ├── quality/
│   ├── plans/
│   ├── specs/
│   ├── reports/
│   ├── evidence/
│   ├── tech-debt/
│   ├── recipes/
│   ├── roadmap/
│   └── references/
├── packs/
│   └── languages/
└── scripts/
```

## Quick start

1. Initialize the project (cleans template artifacts, bootstraps hooks and directories).

   ```sh
   ./scripts/init-project.sh
   ```

2. Edit these files first:
   - `AGENTS.md`
   - `CLAUDE.md`
   - `.claude/rules/*.md`
   - `packs/languages/*/verify.sh` or `scripts/verify.local.sh`

3. Create your first plan.

   ```sh
   # Standard flow
   ./scripts/new-feature-plan.sh login-form

   # Ralph Loop (directory-based plan with parallel slices)
   ./scripts/new-ralph-plan.sh login-form N/A 3
   ```

4. In Claude Code, follow the loop:
   - `/spec` (optional) → `/plan` → `/work` (or `/loop`) → `/self-review` → `/verify` → `/test` → `/sync-docs` → `/codex-review` (optional) → `/pr`

5. Before claiming a task is done, run:

   ```sh
   ./scripts/run-verify.sh
   ```

## Operating loop

This scaffold assumes the following default loop. `/spec` is the only manual trigger; all other steps are auto-invoked.

1. **Spec** (manual, optional — `/spec`)
   - Use when the request is too vague for `/plan`
   - Explores the codebase, researches best practices, and clarifies requirements interactively
   - Produces a spec file in `docs/specs/` and optionally creates a GitHub issue
   - Can hand off directly to `/plan` after completion

2. **Plan** (auto — `/plan`)
   - Create or refresh a file-backed plan in `docs/plans/active/`
   - Define acceptance criteria, contracts, risks, and verification
   - Optionally link a GitHub issue for context pre-fill
   - Select execution flow: standard (`/work`) or Ralph Loop (`/loop`)

3. **Work** (auto — `/work`) or **Loop** (auto — `/loop`)
   - `/work`: creates a branch (`git checkout -b`) and implements interactively in Claude Code
   - `/loop`: creates a Git Worktree and sets up autonomous iteration via `claude -p`

4. **Self-review** (auto — `/self-review`)
   - Produce a written review artifact (diff quality only)
   - Prefer read-only reviewer agents for audit tasks

5. **Verify** (auto — `/verify`)
   - Check spec compliance against acceptance criteria
   - Run static analysis and documentation drift checks
   - Record results in `docs/reports/`

6. **Test** (auto — `/test`)
   - Run behavioral tests (unit, integration, regression)
   - Tests must pass before PR creation

7. **Sync docs** (auto — `/sync-docs`)
   - Sync plans, docs, and instruction files after behavior changes
   - Check for documentation drift across AGENTS.md, CLAUDE.md, rules, and README

8. **Codex review** (auto, optional — `/codex-review`)
   - Cross-model second opinion on the diff using Codex CLI
   - Silently skipped if Codex is unavailable
   - Findings are advisory — user decides whether to act

9. **PR** (auto — `/pr`)
   - Create a pull request with structured summary
   - Archive finished plans from `active/` to `archive/`
   - Include walkthrough for large diffs

10. **CI + Human merge**
    - `verify.yml` runs `run-verify.sh` on the PR
    - Human reviews and merges

## Hook configuration

`.claude/settings.json` ships with all hooks pre-configured:

- Session start context
- Prompt-level reminders
- Bash guardrails
- Edit/write verification reminders
- Tool failure feedback
- Compaction checkpoints
- Session end summary

Customize by editing `.claude/settings.json` directly. Use `.claude/settings.local.json` for personal overrides (gitignored).

## Language packs

The core scaffold stays stack-agnostic. Language-specific depth lives in `packs/languages/`.

Included starter packs:
- `typescript/`
- `python/`
- `rust/`
- `golang/`
- `dart/` (Flutter support included)
- `_template/` for new packs

Add a new pack with:

```sh
./scripts/new-language-pack.sh go
```

Then wire it into:
- `packs/languages/<name>/verify.sh`
- `.claude/rules/<name>.md`
- project build/test/tooling

## Ralph Loop (autonomous parallel execution)

For large tasks that can be split into independent slices, the Ralph Loop runs parallel pipelines across multiple Git worktrees. Each slice gets its own `claude -p` pipeline that handles the full lifecycle autonomously (implement → self-review → verify → test → sync-docs → codex-review). Completed slices are sequentially merged into an integration branch, and a unified PR is created.

```sh
# Create a directory-based plan with slices
./scripts/new-ralph-plan.sh my-feature N/A 3

# Run the orchestrator
./scripts/ralph run --plan docs/plans/active/2026-01-01-my-feature/ --unified-pr

# Check progress (launches TUI when available, table output otherwise)
./scripts/ralph status

# Table output only (skip TUI)
./scripts/ralph status --no-tui

# JSON output
./scripts/ralph status --json

# Retry a failed/stuck slice
./scripts/ralph retry <slice-name>

# Abort a single slice
./scripts/ralph abort --slice <slice-name>

# Abort all slices
./scripts/ralph abort

# Build the TUI binary (requires Go 1.22+)
./scripts/build-tui.sh
```

Or use the `/loop` skill inside Claude Code for interactive setup.

When a TUI binary (`bin/ralph-tui`) is available and the terminal is a TTY, `ralph status` launches a Lazygit-style 4-pane interface for real-time slice monitoring, log tailing, and interactive retry/abort. If the binary is missing or outdated, it falls back to the existing table output.

Safety rails include iteration limits, stuck detection (3 consecutive no-change iterations), Inner/Outer Loop architecture with repair attempt caps, slice timeout detection, signal handlers for clean shutdown, and hook parity checks. All pipeline settings (model, effort, permission mode, iteration caps, timeouts) are configurable via environment variables through `scripts/ralph-config.sh`.

See `docs/recipes/ralph-loop.md` for the full guide.

## Portability model

This scaffold deliberately separates:
- **Portable instruction map**: `AGENTS.md`
- **Claude-native control plane**: `CLAUDE.md`, `.claude/rules/`, `.claude/skills/`, `.claude/hooks/`, `.claude/agents/`
- **Language packs**: `packs/languages/`
- **Deterministic scripts and CI**: `scripts/`, `.github/workflows/`

That gives you a base you can keep if you later add Codex, Gemini CLI, or another coding agent.

## Recommended adoption order

See `docs/roadmap/harness-maturity-model.md`, but the short version is:

1. Map + verify
2. Plan/work/self-review/verify skills
3. Deterministic hooks
4. Path-scoped rules and subagents
5. Worktrees and agent teams for genuinely parallel tasks
6. Evaluator loops and richer observability only when the task complexity earns the cost

## Important defaults

- Keep `AGENTS.md` short
- Keep `CLAUDE.md` shorter
- Move topic-specific guidance to `.claude/rules/`
- Move workflow-specific guidance to `.claude/skills/`
- Prefer evidence over confidence
- Do not rely on prose for hard guarantees
- Treat human attention as the scarcest resource in the system

## Useful files to inspect first

- `docs/research/approach-comparison.md`
- `docs/architecture/design-principles.md`
- `.claude/skills/plan/SKILL.md`
- `.claude/skills/verify/SKILL.md`
- `.claude/hooks/pre_bash_guard.sh`
- `scripts/run-verify.sh`
- `docs/roadmap/harness-maturity-model.md`

## License

MIT
