---
name: work
description: Execute an approved plan in small coherent slices, updating progress, evidence, and docs as implementation evolves. Invoke automatically after an approved plan exists and the current branch is a feature branch.
---
Work from the active plan, not from memory alone.

## Steps

0. **Resolve the target plan path** (must run before any branch or plan-file operations):
   - `/work` is the standard-flow skill — it only operates on single-file plans (`docs/plans/active/<date>-<slug>.md`). Ralph Loop directory plans (`docs/plans/active/<date>-<slug>/`) must be handled by `/loop` instead.
   - Enumerate candidates: `.md` files directly under `docs/plans/active/` (excluding `.gitkeep`). **Ignore directories** — if a directory is the only entry, stop and ask the user to run `/loop` (Ralph Loop).
   - If exactly one candidate file exists, use it.
   - If multiple candidate files exist, ask via AskUserQuestion which plan this `/work` run targets, and use the selected path.
   - If none exist, stop and ask the user to run `/plan` first.
   - Downstream steps in this skill — and downstream skills (`/codex-review`, `/pr`) — MUST use this resolved path instead of rescanning `docs/plans/active/`.
0.5. **Create feature branch** (if not already on one), based on the plan resolved in Step 0:
   a. Read the resolved plan to extract metadata (type, issue number, slug).
   b. Determine branch name: `<type>/<issue>/<slug>` (with issue) or `<type>/<slug>` (without issue).
   c. If already on a feature branch (not main/master), skip creation.
   d. Otherwise, run `git checkout -b <branch-name>`.
   e. Update the resolved plan file: replace `Branch: TBD` (or any TBD variant) with the actual branch name.
0.7. **Pin the plan identity and initialize the pipeline cycle counter** (enforces the 2-cycle cap):
   a. Create `.harness/state/standard-pipeline/` if missing (`mkdir -p`). This directory is already covered by the existing `.harness/state/` gitignore.
   b. Write the Step-0 resolved absolute path to `.harness/state/standard-pipeline/active-plan.json` as `{"plan_path": "<absolute-path>", "created_at": "<UTC ISO8601>"}`. If the file already exists with a different `plan_path`, warn the user and ask whether to overwrite (resume) or abort.
   c. Handle `.harness/state/standard-pipeline/cycle-count.json`:
      - If the file is missing: initialize as `{"plan_path": "<absolute-path>", "cycle": 1}`.
      - If the file exists AND its `plan_path` matches the pinned plan: **preserve the existing counter** (do NOT reset to 1). This keeps the cap effective when the user resumes a plan after context compaction or a later session. Inform the user of the resumed cycle number.
      - If the file exists AND its `plan_path` differs from the pinned plan: warn and prompt via AskUserQuestion whether to reset the counter for the new plan or abort.
      - The counter reflects the **current** pipeline run (1 = first run, 2 = one re-run after Codex ACTION_REQUIRED).
1. Read the current active plan using the path recorded in `active-plan.json`.
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
   e. **Invoke `/codex-review` via the Skill tool** (optional, inline — if Codex CLI unavailable, skip to `/pr`). The skill reads `cycle-count.json` and enforces `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default 2). On re-run after ACTION_REQUIRED fixes, `/codex-review` increments `cycle-count.json`.
   f. **Invoke `/pr` via the Skill tool** — do NOT run `gh pr create` directly. The `/pr` skill enforces the Japanese template, pre-checks, and plan archiving. On success, `/pr` deletes `.harness/state/standard-pipeline/active-plan.json` and `cycle-count.json`.

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
