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
2. Plan (manual — creates branch)
3. Work (auto)
4. Review (auto)
5. Verify (auto)
6. PR (auto — includes hand-off)
7. CI verify + human merge

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

- `docs/plans/active/` — current plans
- `docs/plans/archive/` — completed plans
- `docs/reports/` — review, verify, walkthrough artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance
- `.claude/skills/` — on-demand workflows
- `.claude/agents/` — specialized subagents
- `.claude/hooks/` — deterministic runtime checks
- `packs/languages/` — language-specific depth
- `scripts/` — reusable verification and bootstrap scripts
- `.harness/state/` — runtime state, not canonical truth

## Planning contract

Every non-trivial task should have:
- objective
- scope and non-goals
- affected files or modules
- acceptance criteria
- verification strategy
- risk register
- rollout or rollback note
- evidence targets

## Review contract

Reviews should produce artifacts, not only chat output:
- findings with severity
- evidence
- merge or no-merge recommendation
- follow-ups
- known gaps

## Verification contract

Prefer this order:
1. tests
2. linters and type checks
3. targeted runtime commands
4. screenshots, logs, traces, or metrics
5. structured manual walkthrough

Never say "done" without saying what was verified and what remains unverified.

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
