# AGENTS.md

This repository hosts `ralph`, a CLI for harness engineering. Run `ralph init` to scaffold a new project from this source.

Treat this file as a **map**:
- short
- stable
- cross-vendor
- easy to verify against the repo

<!-- BEGIN RALPH MANAGED (ralph:agents-md) -->
## Mission

Build coding-agent workflows that are:
- reliable
- inspectable
- evidence-backed
- easy to extend
- cheap by default, richer only when needed

## Primary loop

This project ships two independent execution surfaces:

- **development harness** — the interactive standard flow below, used for all spec/plan/work/review changes.
- **org runtime** — autonomous multi-seat execution (`ralph org spawn/send/wait/...`) for tasks that need a coordinating `lead` plus role seats running outside a single interactive session.

The development harness:

1. Spec (auto, optional — refines vague ideas into detailed specifications via decision-tree questioning, codebase exploration, web research, and user clarification)
2. Plan (auto — ensures a clean-base task worktree, creates plan) [+ optional Codex plan advisory]
3. Work (auto — resumes task worktree, interactive implementation)
4. Self-review (auto — via `reviewer` subagent, or pipeline-internal)
5. Verify (auto — via `verifier` subagent, or pipeline-internal)
6. Test (auto — via `tester` subagent, or pipeline-internal)
7. Sync-docs (auto — via `doc-maintainer` subagent, or pipeline-internal)
8. Cross-review (auto, optional — cross-model second opinion via the other agent: Claude → Codex; Codex → Claude)
9. PR (auto — includes hand-off)
10. CI verify + human merge

All repo writes in spec/plan/work flows must happen inside a task worktree
created from a clean default branch.

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Verification & test contracts

Key rule: never say "done" without saying what was verified and what remains unverified. Tests must pass before PR creation.

## Hard rules

- Keep this file short
- Keep `CLAUDE.md` short
- Move detailed topic guidance into `.claude/rules/ralph/`
- Move step-by-step workflows into `.claude/skills/`
- Promote repeated mistakes into hooks, tests, CI, or scripts
- Do not expand plans into brittle low-level instructions unless the task truly needs it
- Keep names grep-able and boundaries explicit
- Update docs when behavior, contracts, or workflows change

Detailed topic guidance lives in `.claude/rules/ralph/`; step-by-step
workflows live in `.claude/skills/`.
<!-- END RALPH MANAGED -->

## Org runtime pointers (meta-repo specific)

The self-review, verify, test, and sync-docs steps run through
phase-specific subagents for both Claude Code and Codex; `/cross-review`
and `/pr` remain inline. Local task state for spec/plan/work lives under
`$(git rev-parse --git-common-dir)/ralph/worktrees/`; PR success cleans up
the task worktree and local branch while leaving the remote PR branch
intact.

See `internal/org/`, `docs/specs/2026-08-01-org-runtime.md`, and
`.claude/rules/ralph/agent-messaging.md` for the org runtime's implementation
and protocol contract.

## Repo map

- `cmd/ralph/` — Go entrypoint for the ralph CLI (cobra root, ldflags injection, go:embed wiring)
- `internal/cli/` — cobra subcommands (init, upgrade, eject, adopt, status, doctor, pack, insights, org, version)
- `internal/scaffold/` — go:embed template system, manifest TOML (v3 ownership fields: `layout`/`owner` core|fork|seed|block/`forked_from_version`, opt-in write API, legacy read compat), file render with SHA256 hashes
- `internal/upgrade/` — overlay engine wired into `ralph upgrade` for v2 layouts: core replace planner with ordered ops + commit barrier, managed block engine, `.claude/settings.json` 3-way merge against the `.ralph/core/settings.ralph.json` snapshot, advisory diff, upgrade report writer (`docs/reports/upgrade-*.md`); legacy layouts migrate via a confirmed one-time flow (internal/cli/migrate.go) before chaining into this engine (spec: docs/specs/2026-08-17-overlay-scaffold-v2.md)
- `internal/config/` — ralph.toml parser with defaults
- `internal/org/` — org runtime mechanism layer: envelope validation, seat saga manifest (flock-serialized), receipts, permission-mode envelopes, herdr/agmsg driver adapters, role prompt templates (go:embed), typed message protocol, org report generation, two-layer watchdog (pulse watch + on-demand watcher) (spec: docs/specs/2026-08-01-org-runtime.md; protocol rule: .claude/rules/ralph/agent-messaging.md)
- `internal/insights/` — insight event/receipt readers, aggregation, and report backfill for `ralph insights`
- `templates/` — go:embed source: base scaffold, language packs
- `docs/specs/` — spec files produced by `/spec` (`<date>-<slug>.md`)
- `docs/plans/active/` — current plans (`<date>-<slug>.md` single files)
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates (`feature-plan.md`)
- `docs/reports/` — self-review, verify, test, sync-docs, cross-review triage, walkthrough artifacts
- `docs/insights/` — committed insight events (`events/<date>-<slug>.jsonl`); schema in `docs/insights/README.md`; consumed by `ralph insights`
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/ralph/` — path-scoped ralph guidance (read by both agents); language pack rules also render here as `<lang>.md`
- `.claude/skills/` — Claude-side on-demand workflows
- `.claude/agents/` — Claude Code subagent definitions
- `.claude/hooks/` — deterministic runtime controls; `settings.json` points each event at `ralph-dispatch.sh <event>`, which fans out to `.claude/hooks/<event>.d/` (core), `.ralph/local/hooks/<event>.d/` (downstream local, committed), then `.claude/hooks/local/<event>.d/` (downstream local, gitignored)
  - `check_mojibake.sh` + `mojibake-allowlist` — temporary U+FFFD detection guard for Claude Code SSE mojibake (remove once upstream Issue #43746 ships)
- `.ralph/core/` — generation sources consumed by `ralph init` (e.g. `AGENTS.core.md`, the managed-block content for `AGENTS.md`; `settings.ralph.json`, the settings 3-way-merge snapshot consumed by `ralph upgrade`)
- `.ralph/local/` — downstream extension points (`hooks/<event>.d/`, `verify.d/`, `test.d/`) that `scripts/run-verify.sh` and the hooks dispatcher execute after core processing (`hooks/<event>.d/` wiring runs under both Claude Code and Codex — both route through `ralph-dispatch.sh`)
- `.agents/skills/` — Codex-side skill bodies (mirrors `.claude/skills/`; regenerated by `scripts/sync-skills.sh`, drift-checked by `scripts/check-skill-sync.sh`)
- `.codex/` — Codex project config for this meta-repo (`config.toml`, `agents/`, `hooks/`, `AGENTS.override.md`, `README.md`); `agents/` contains Codex custom agent definitions; same shape as `templates/base/.codex/` so ralph dogfoods the parity it ships
- `templates/base/.codex/` — `ralph init` source for the same surface; root `.codex/` and template `.codex/` are kept identical via `scripts/check-sync.sh` (no KNOWN_DIFFS today)
- `packs/languages/` — language-specific depth (also copied to `templates/packs/` for embedding)
- `scripts/` — reusable verification and bootstrap scripts (includes `ralph-common.sh` (shared shell helpers: ts/log/default_branch/detect_active_plan_dir); `ralph-config.sh`, `ralph-worktree.sh`, `xreview-helpers.sh` (cross-review driver helpers: `detect_base_branch` / `pick_reviewer` / `count_triage_findings`), `install.sh`, skills-mirror generator `sync-skills.sh`, drift gate `check-skill-sync.sh`, artifact retention GC `gc-artifacts.sh`, insight-event appender `insights-append.sh`, Codex availability probe `codex-check.sh`)
- `docs/recipes/` — hands-on recipes (Codex setup, language packs, worktrees)
- `.harness/state/` — runtime state, not canonical truth

## Planning contract

Every non-trivial task should have:
- objective
- scope and non-goals
- affected files or modules
- acceptance criteria
- verify plan (static analysis, spec compliance, doc drift)
- test plan (unit, integration, regression, edge cases)
- risk register
- rollout or rollback note
- evidence targets

## Review contract

Reviews should produce artifacts, not only chat output:
- findings with severity (diff quality only)
- evidence
- merge or no-merge recommendation
- follow-ups
- known gaps

## Verification & test contracts (meta-repo addition)

See `docs/quality/definition-of-done.md` for full checklists.

## Human escalation boundaries

Escalate to a human only for:
- irreversible destructive actions
- secrets or credentials you cannot access
- product or design judgment that cannot be verified from repo context
- external approvals that are genuinely required

Everything else should first attempt self-verification.
