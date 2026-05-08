# AGENTS.md

This repository hosts `ralph`, a CLI for harness engineering. Run `ralph init` to scaffold a new project from this source.

Treat this file as a **map**:
- short
- stable
- cross-vendor
- easy to verify against the repo

## Mission

Build coding-agent workflows that are:
- reliable
- inspectable
- evidence-backed
- easy to extend
- cheap by default, richer only when needed

## Primary loop

1. Spec (manual, optional — refines vague ideas into detailed specifications via iterative brainstorming, codebase exploration, web research, and user clarification → `docs/specs/` or GitHub issue)
2. Plan (auto — creates plan, selects flow) [+ optional Codex plan advisory]
3. **標準フロー**: Work (auto — creates branch, interactive implementation)
   **Ralph Loop**: Loop (auto — directory-based plan → `ralph-orchestrator.sh` → multi-worktree parallel → integration branch → integration pipeline → unified PR)
4. Self-review (auto — via `reviewer` subagent, or pipeline-internal)
5. Verify (auto — via `verifier` subagent, or pipeline-internal)
6. Test (auto — via `tester` subagent, or pipeline-internal)
7. Sync-docs (auto — via `doc-maintainer` subagent, or pipeline-internal)
8. Cross-review (auto, optional — cross-model second opinion via the other CLI: Claude → Codex; Codex → Claude)
9. PR (auto — includes hand-off)
10. CI verify + human merge

Steps 4–9 run via subagents in 標準フロー. In Ralph Loop, they are handled internally by the pipeline scripts.

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

- `cmd/ralph/` — Go entrypoint for the ralph CLI (cobra root, ldflags injection, go:embed wiring)
- `cmd/ralph-tui/` — Legacy TUI entrypoint (to be removed in Phase 9)
- `internal/cli/` — cobra subcommands (init, upgrade, run, status, retry, abort, doctor, pack, version)
- `internal/scaffold/` — go:embed template system, manifest TOML, file render with SHA256 hashes
- `internal/upgrade/` — hash-based diff engine, conflict resolution (auto-update, conflict, add, remove)
- `internal/config/` — ralph.toml parser with defaults
- `internal/state/` — pipeline state reader (checkpoint, orchestrator, manifest parsing)
- `internal/watcher/` — fsnotify-based file watcher with polling fallback
- `internal/ui/` — Bubble Tea model, layout, panes, keybindings, styles
- `internal/action/` — CLI action executor (retry, abort)
- `templates/` — go:embed source: base scaffold, language packs
- `docs/specs/` — spec files produced by `/spec` (`<date>-<slug>.md`)
- `docs/plans/active/` — current plans (single files for standard flow; `<date>-<slug>/` directories with `_manifest.md` + `slice-*.md` for Ralph Loop)
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates (`feature-plan.md`, `ralph-loop-manifest.md`, `ralph-loop-slice.md`)
- `docs/reports/` — self-review, verify, test, walkthrough artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance (read by both CLIs)
- `.claude/skills/` — Claude-side on-demand workflows
- `.claude/agents/` — specialized subagents (Claude only)
- `.claude/hooks/` — deterministic runtime checks
  - `check_mojibake.sh` + `mojibake-allowlist` — temporary U+FFFD detection guard for Claude Code SSE mojibake (remove once upstream Issue #43746 ships)
- `.agents/skills/` — Codex-side skill bodies (mirrors `.claude/skills/`; kept in lock-step by `scripts/check-skill-sync.sh`)
- `.codex/` — Codex project config for this meta-repo (`config.toml`, `hooks/`, `AGENTS.override.md`, `README.md`); same shape as `templates/base/.codex/` so ralph dogfoods the parity it ships
- `templates/base/.codex/` — `ralph init` source for the same surface; root `.codex/` and template `.codex/` are kept identical via `scripts/check-sync.sh` (no KNOWN_DIFFS today)
- `packs/languages/` — language-specific depth (also copied to `templates/packs/` for embedding)
- `scripts/` — reusable verification and bootstrap scripts (includes legacy `ralph` shell CLI, `ralph-config.sh`, `ralph-pipeline.sh`, `ralph-orchestrator.sh`, `install.sh`, drift gate `check-skill-sync.sh`, Codex availability probe `codex-check.sh`)
- `docs/recipes/` — hands-on recipes (Codex setup, Ralph Loop, language packs, worktrees)
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

## Verification & test contracts

See `docs/quality/definition-of-done.md` for full checklists.

Key rule: never say "done" without saying what was verified and what remains unverified. Tests must pass before PR creation.

## Hard rules

- Keep this file short
- Keep `CLAUDE.md` short
- Move detailed topic guidance into `.claude/rules/`
- Move step-by-step workflows into `.claude/skills/`
- Promote repeated mistakes into hooks, tests, CI, or scripts
- Do not expand plans into brittle low-level instructions unless the task truly needs it
- Keep names grep-able and boundaries explicit
- Update docs when behavior, contracts, or workflows change

## Human escalation boundaries

Escalate to a human only for:
- irreversible destructive actions
- secrets or credentials you cannot access
- product or design judgment that cannot be verified from repo context
- external approvals that are genuinely required

Everything else should first attempt self-verification.
