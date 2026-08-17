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
