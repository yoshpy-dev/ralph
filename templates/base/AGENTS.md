# AGENTS.md

Treat this file as a **map** that both Claude Code and Codex read:
- short
- stable
- cross-vendor
- easy to verify against the repo

This file is the shared source of truth. Claude-only details live in
`CLAUDE.md`. Codex-only details live in `.codex/AGENTS.override.md` and
`.codex/README.md`.

## Mission

Build coding-agent workflows that are:
- reliable
- inspectable
- evidence-backed
- easy to extend

## Primary loop

1. Spec (manual, optional — refines vague ideas into detailed specifications via iterative brainstorming, codebase exploration, web research, and user clarification → `docs/specs/` or GitHub issue)
2. Plan (auto — creates plan, selects flow) [+ optional Codex plan advisory]
3. **Standard flow**: Work (auto — creates branch, interactive implementation)
   **Ralph Loop**: Loop (auto — directory-based plan → `ralph-orchestrator.sh` → multi-worktree parallel → integration branch → integration pipeline → unified PR)
4. Self-review (auto)
5. Verify (auto)
6. Test (auto)
7. Sync-docs (auto)
8. Cross-review (auto, optional — cross-model second opinion via the other CLI: Claude → Codex; Codex → Claude)
9. PR (auto — includes hand-off)
10. CI verify + human merge

In Claude Code's standard flow, steps 4–7 run via subagents (`reviewer`,
`verifier`, `tester`, `doc-maintainer`). In Codex they run sequentially in
one agent. In Ralph Loop they are handled internally by the pipeline scripts.

## Skill invocation

| CLI | How to invoke a skill | Notes |
|-----|------------------------|-------|
| Claude Code | `/skill-name` slash command | Set in `.claude/skills/<name>/SKILL.md` frontmatter |
| Codex | `$skill-name` mention or `/skills` menu | `/skill-name` collides with Codex built-ins (e.g. `/plan`) — do not use |

Both CLIs read the same skill bodies. Claude reads `.claude/skills/`, Codex
reads `.agents/skills/`. `scripts/check-skill-sync.sh` keeps the two trees in
lock-step (body, name, description, implicit-invocation policy).

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

<!-- Update this section to reflect your project's structure -->

- `docs/specs/` — spec files produced by `/spec`
- `docs/plans/active/` — current plans
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates
- `docs/reports/` — self-review, verify, test, walkthrough artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance (read by both CLIs)
- `.claude/skills/` — Claude Code skill bodies
- `.claude/agents/` — Claude Code subagent definitions
- `.claude/hooks/` — Claude Code runtime hooks
- `.agents/skills/` — Codex skill bodies (mirrors `.claude/skills/`)
- `.codex/` — Codex project config, hooks, override docs
- `scripts/` — reusable verification, hook, and bootstrap scripts (shared)

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

Key rule: never say "done" without saying what was verified and what remains
unverified. Tests must pass before PR creation.

## Codex setup checklist

If you intend to drive ralph from Codex, finish this once per project before
starting any flow:

1. Install Codex CLI (>= 0.128.0).
2. `codex trust .` — without trust, `.codex/config.toml`, `[features]`, and
   `[hooks]` are silently ignored.
3. `ralph doctor` — confirms CLI presence, project trust, `codex_hooks` flag,
   and at least one effective `[hooks]` entry.

See `.codex/README.md` for the full guide.

## Hard rules

- Keep this file short
- Keep `CLAUDE.md` short
- Move detailed topic guidance into `.claude/rules/` (read by both CLIs)
- Move step-by-step workflows into `.claude/skills/` and mirror in `.agents/skills/`
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
