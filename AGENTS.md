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

1. Explore
2. Plan (manual — creates plan, selects flow) [+ optional Codex plan advisory]
3. Work (auto — creates branch) or Loop (auto — creates worktree)
   - Loop standard: implementation-only, post-impl pipeline runs via subagents
   - Loop pipeline: full autonomous Inner/Outer Loop (implement → review → verify → test → docs → codex → PR)
   - Loop parallel slices: directory-based plan → multi-worktree → integration branch → sequential merge → unified PR
4. Self-review (auto — via `reviewer` subagent, or pipeline-internal)
5. Verify (auto — via `verifier` subagent, or pipeline-internal)
6. Test (auto — via `tester` subagent, or pipeline-internal)
7. Codex review (auto, optional — cross-model second opinion)
8. PR (auto — includes hand-off)
9. CI verify + human merge

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

- `docs/plans/active/` — current plans (single files or `<date>-<slug>/` directories with `_manifest.md` + `slice-*.md`)
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates (`feature-plan.md`, `ralph-loop-plan.md`, `ralph-loop-manifest.md`, `ralph-loop-slice.md`)
- `docs/reports/` — self-review, verify, test, walkthrough artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance
- `.claude/skills/` — on-demand workflows
- `.claude/agents/` — specialized subagents
- `.claude/hooks/` — deterministic runtime checks
- `packs/languages/` — language-specific depth
- `scripts/` — reusable verification and bootstrap scripts (includes `ralph` CLI, `ralph-pipeline.sh`, `ralph-orchestrator.sh`, `new-ralph-plan.sh`)
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

## Verification contract

Prefer this order:
1. spec compliance check against acceptance criteria
2. linters and type checks (static analysis)
3. documentation drift check
4. targeted runtime commands
5. screenshots, logs, traces, or metrics
6. structured manual walkthrough

Never say "done" without saying what was verified and what remains unverified.

## Test contract

Tests should produce artifacts:
- test execution results with pass/fail counts
- coverage metrics
- failure analysis with root causes
- regression check results
- explicit test gaps

Tests must pass before PR creation.

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
