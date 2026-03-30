# coding-agent-harness-scaffold

A language-agnostic scaffold for practicing harness engineering with **Claude Code first** and **cross-vendor portability second**.

This repository is intentionally designed as a **map, not a manual**:
- `AGENTS.md` is the vendor-neutral map for any coding agent.
- `CLAUDE.md` imports `AGENTS.md` and adds Claude Code specific guidance.
- `.claude/rules/` keeps conditional guidance out of the always-on context.
- `.claude/skills/` provides on-demand workflows for plan, work, review, verify, and harness auditing.
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
│   ├── settings.minimal.example.json
│   ├── settings.advanced.example.json
│   ├── hooks/
│   ├── skills/
│   ├── agents/
│   └── rules/
├── docs/
│   ├── research/
│   ├── architecture/
│   ├── quality/
│   ├── plans/
│   ├── reports/
│   ├── evidence/
│   ├── recipes/
│   ├── roadmap/
│   └── references/
├── packs/
│   └── languages/
├── scripts/
└── examples/
```

## Quick start

1. Copy the minimal hook profile.

   ```sh
   cp .claude/settings.minimal.example.json .claude/settings.json
   ```

2. Bootstrap local runtime folders.

   ```sh
   ./scripts/bootstrap.sh
   ```

3. Edit these files first:
   - `AGENTS.md`
   - `CLAUDE.md`
   - `.claude/rules/*.md`
   - `packs/languages/*/verify.sh` or `scripts/verify.local.sh`

4. Create your first plan.

   ```sh
   ./scripts/new-feature-plan.sh login-form
   ```

5. In Claude Code, follow the loop:
   - `/plan`
   - `/work`
   - `/review`
   - `/verify`

6. Before claiming a task is done, run:

   ```sh
   ./scripts/run-verify.sh
   ```

## Operating loop

This scaffold assumes the following default loop:

1. **Explore**
   - Read relevant code, docs, rules, and open plans
   - Decide whether the task is small enough to stay single-session

2. **Plan**
   - Create or refresh a file-backed plan in `docs/plans/active/`
   - Define acceptance criteria, contracts, risks, and verification

3. **Work**
   - Implement in small coherent slices
   - Update plan progress as you go
   - Keep evidence and docs aligned with code changes

4. **Review**
   - Produce a written review artifact, not just a verbal "looks good"
   - Prefer read-only reviewer agents for audit tasks

5. **Verify**
   - Run deterministic checks first
   - Record verification results in `docs/reports/`
   - Note any remaining coverage gaps explicitly

6. **Hand off**
   - Provide a concise walkthrough when the diff is large or subtle
   - Archive finished plans from `active/` to `archive/`

## Minimal vs advanced profiles

- `settings.minimal.example.json`
  - Session start context
  - Bash guardrails
  - Edit/write reminders
  - Failure feedback
  - Session end summary

- `settings.advanced.example.json`
  - Adds prompt-level reminders
  - Adds compaction checkpoints
  - Adds instruction-load observability

Keep the minimal profile first. Only adopt the advanced profile after the base loop is already useful.

## Language packs

The core scaffold stays stack-agnostic. Language-specific depth lives in `packs/languages/`.

Included starter packs:
- `typescript/`
- `python/`
- `rust/`
- `_template/` for new packs

Add a new pack with:

```sh
./scripts/new-language-pack.sh go
```

Then wire it into:
- `packs/languages/<name>/verify.sh`
- `.claude/rules/<name>.md`
- project build/test/tooling

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
2. Plan/work/review/verify skills
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
