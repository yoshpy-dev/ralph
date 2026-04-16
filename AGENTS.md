# AGENTS.md

This repository is a scaffold for harness engineering.

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

1. Spec (manual, optional — refines vague ideas into detailed specifications via codebase exploration, web research, and user clarification → `docs/specs/` or GitHub issue)
2. Plan (auto — creates plan, selects flow) [+ optional Codex plan advisory]
3. **標準フロー**: Work (auto — creates branch, interactive implementation)
   **Ralph Loop**: Loop (auto — directory-based plan → `ralph-orchestrator.sh` → multi-worktree parallel → integration branch → integration pipeline → unified PR)
4. Self-review (auto — via `reviewer` subagent, or pipeline-internal)
5. Verify (auto — via `verifier` subagent, or pipeline-internal)
6. Test (auto — via `tester` subagent, or pipeline-internal)
7. Sync-docs (auto — via `doc-maintainer` subagent, or pipeline-internal)
8. Codex-review (auto, optional — cross-model second opinion)
9. PR (auto — includes hand-off)
10. CI verify + human merge

Steps 4–9 run via subagents in 標準フロー. In Ralph Loop, they are handled internally by the pipeline scripts.

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

- `cmd/ralph-tui/` — Go entrypoint for Bubble Tea TUI (`main.go`, `version.go`)
- `internal/state/` — pipeline state reader (checkpoint, orchestrator, manifest parsing)
- `internal/watcher/` — fsnotify-based file watcher with polling fallback
- `internal/ui/` — Bubble Tea model, layout, panes, keybindings, styles
- `internal/action/` — CLI action executor (retry, abort via `scripts/ralph`)
- `docs/specs/` — spec files produced by `/spec` (`<date>-<slug>.md`)
- `docs/plans/active/` — current plans (single files for standard flow; `<date>-<slug>/` directories with `_manifest.md` + `slice-*.md` for Ralph Loop)
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates (`feature-plan.md`, `ralph-loop-manifest.md`, `ralph-loop-slice.md`)
- `docs/reports/` — self-review, verify, test, walkthrough artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance
- `.claude/skills/` — on-demand workflows
- `.claude/agents/` — specialized subagents
- `.claude/hooks/` — deterministic runtime checks
- `packs/languages/` — language-specific depth
- `scripts/` — reusable verification and bootstrap scripts (includes `ralph` CLI, `ralph-config.sh`, `ralph-pipeline.sh`, `ralph-orchestrator.sh`, `new-ralph-plan.sh`, `build-tui.sh`)
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
