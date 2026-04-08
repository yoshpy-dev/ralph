---
name: work
description: Execute an approved plan in small coherent slices, updating progress, evidence, and docs as implementation evolves. Invoke automatically after an approved plan exists and the current branch is a feature branch.
---
Work from the active plan, not from memory alone.

## Steps

0. **Create feature branch** (if not already on one):
   a. Read the active plan in `docs/plans/active/` to extract metadata (type, issue number, slug).
   b. Determine branch name: `<type>/<issue>/<slug>` (with issue) or `<type>/<slug>` (without issue).
   c. If already on a feature branch (not main/master), skip creation.
   d. Otherwise, run `git checkout -b <branch-name>`.
   e. Update the plan file: replace `Branch: TBD` (or any TBD variant) with the actual branch name.
1. Read the current active plan in `docs/plans/active/`.
2. Confirm acceptance criteria, verify plan, and test plan before editing code.
3. Implement in small slices that can be reviewed and verified independently.
3a. **Commit after each verified slice** (Validation Gate):
    - Run `./scripts/run-verify.sh` (or equivalent) after completing a slice.
    - If verification passes, stage the slice's files and commit with conventional format: `<type>: <description>`.
    - If verification fails, fix before committing. Do not accumulate unverified changes.
    - See `.claude/rules/git-commit-strategy.md` for the full policy.
4. Update the plan's progress checklist while working.
5. If the task splits cleanly, delegate focused research or review to subagents.
6. If repeated failures occur, reduce scope, inspect evidence, and revise the plan instead of thrashing.
7. Keep docs, contracts, and tests aligned with behavior changes.
8. Before presenting completion, run `./scripts/run-verify.sh` or equivalent deterministic checks.
9. After criteria met, delegate the post-implementation pipeline to subagents per `.claude/rules/subagent-policy.md`:
   a. `Task(subagent_type="reviewer")` → `/self-review` — stop if CRITICAL findings
   b. `Task(subagent_type="verifier")` → `/verify` — stop if fail verdict
   c. `Task(subagent_type="tester")` → `/test` — stop if fail verdict
   d. `Task(subagent_type="doc-maintainer")` → `/sync-docs`
   e. `/codex-review` (optional, inline — findings are triaged before user presentation)
   f. **Invoke `/pr` via the Skill tool** — do NOT run `gh pr create` directly. The `/pr` skill enforces the Japanese template, pre-checks, and plan archiving.

## Scope discipline

- Work only on items listed in the plan's scope. If you discover work outside scope, record it in the plan's open questions or tech-debt, do not implement it.
- Each slice should map to one or more acceptance criteria. If a slice does not advance any criterion, question whether it belongs.

## Plan drift detection

- Before each major slice, re-read the plan to confirm alignment.
- If your implementation diverges from the plan (new files, changed interfaces, different approach), update the plan FIRST with a deviation note before continuing.
- Never silently drift. The plan is the contract.

## Uncertainty management

- If you encounter ambiguity that the plan does not address, check repo context and subagents first (anti-bottleneck).
- If still uncertain, record the uncertainty in the plan's open questions and make the smallest safe choice.
- Do not make large irreversible decisions under uncertainty — flag them.

## Completion gate

Do NOT present a task as complete unless ALL of the following are true:

- [ ] `./scripts/run-verify.sh` exits 0 (or a project-specific verifier passes)
- [ ] Each slice is individually committed with a conventional commit message
- [ ] The active plan's progress checklist is fully updated
- [ ] Any discovered tech debt is recorded in `docs/tech-debt/`

If verification has not run, say so explicitly instead of claiming done.

## Anti-bottleneck

Before asking the user for confirmation, next steps, or choices, first check whether you can resolve the question through verification, repo context, subagents, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

## Strong defaults

- One slice at a time
- Evidence before confidence
- Versioned plan over chat-only plan
- Smaller diffs over heroic rewrites
