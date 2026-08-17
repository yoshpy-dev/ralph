---
name: work
description: Execute an approved plan in small coherent slices from an isolated task worktree, updating progress, evidence, and docs as implementation evolves. Invoke automatically after an approved plan exists and the task worktree is ready.
---
Work from the active plan, not from memory alone.

## Steps

1. **Resolve the target plan path** (must run before any branch or plan-file operations):
   - `/work` operates on single-file plans (`docs/plans/active/<date>-<slug>.md`) only.
   - Enumerate candidates: `.md` files directly under `docs/plans/active/` (excluding `.gitkeep`). **Ignore directories** — if a directory is the only entry, stop and report: this looks like a Ralph Loop directory plan (`_manifest.md` + `slice-*.md`); Ralph Loop's autonomous execution system was retired (see git history for the removed orchestration scripts, and `internal/org` for the current autonomous execution surface, `/org`). Convert the plan to a single-file plan via `/plan` and re-run `/work`, or use `/org` for autonomous multi-seat execution.
   - If exactly one candidate file exists, use it.
   - If multiple candidate files exist, ask via AskUserQuestion which plan this `/work` run targets, and use the selected path.
   - If none exist, stop and ask the user to run `/plan` first.
   - Downstream steps in this skill — and downstream skills (`/cross-review`, `/pr`) — MUST use this resolved path instead of rescanning `docs/plans/active/`.
2. **Resolve or resume the task worktree**, based on the plan resolved in Step 1:
   a. Read the resolved plan to extract metadata (type, issue number, slug).
   b. Determine branch name by running `./scripts/branch-name.sh from-plan <resolved-plan-path>`.
   c. Branch names must validate with `./scripts/branch-name.sh validate <branch-name>`. Allowed user-facing branch shapes are `<type>/<issue>/<slug>` (with issue) or `<type>/<slug>` (without issue), where `<type>` is one of `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`, `perf`, `release`, or `security`.
   d. Run `./scripts/ralph-worktree.sh current`. If it returns a state file for the current worktree, use that state id. Otherwise run `./scripts/ralph-worktree.sh ensure --id plan-<slug> --kind standard --branch <branch-name> --path .claude/worktrees/<slug> --plan-path <absolute-plan-path> --canonical-ref <issue/spec/request reference> --cleanup-policy pr-success` unless a matching task worktree state already exists.
   e. If this `/work` invocation is already running inside the returned worktree path, continue. If it is running outside that path, switch all subsequent commands and edits to the returned worktree path.
   f. If the resolved plan file is a legacy plan outside the task worktree, stop and migrate it intentionally instead of silently copying it; new `/plan` runs should already create plans inside the task worktree.
   g. Update the resolved plan file inside the task worktree: replace `Branch: TBD` (or any TBD variant) with the actual branch name.
3. **Pin the plan identity and initialize the pipeline cycle counter** (enforces the 2-cycle cap):
   a. Create `.harness/state/standard-pipeline/` if missing (`mkdir -p`). This directory is already covered by the existing `.harness/state/` gitignore.
   b. Write the Step-1 resolved absolute path plus worktree metadata to `.harness/state/standard-pipeline/active-plan.json` as `{"plan_path": "<absolute-path>", "worktree_path": "<absolute-worktree-path>", "branch": "<branch>", "base_sha": "<base-sha>", "worktree_state_id": "plan-<slug>", "created_at": "<UTC ISO8601>"}`. If the file already exists with a different `plan_path` or `worktree_state_id`, warn the user and ask whether to overwrite (resume) or abort.
   c. Handle `.harness/state/standard-pipeline/cycle-count.json`:
      - If the file is missing: initialize as `{"plan_path": "<absolute-path>", "cycle": 1}`.
      - If the file exists AND its `plan_path` matches the pinned plan: **preserve the existing counter** (do NOT reset to 1). This keeps the cap effective when the user resumes a plan after context compaction or a later session. Inform the user of the resumed cycle number.
      - If the file exists AND its `plan_path` differs from the pinned plan: warn and prompt via AskUserQuestion whether to reset the counter for the new plan or abort.
      - The counter reflects the **current** pipeline run (1 = first run, 2 = one re-run after cross-review ACTION_REQUIRED).
4. Read the current active plan using the path recorded in `active-plan.json`.
5. Confirm acceptance criteria, verify plan, and test plan before editing code.
6. Implement in small slices that can be reviewed and verified independently. **Delegate each implementation slice to the `implementer` subagent** (Claude Code: `Task(subagent_type="implementer")`; Codex: the `.codex/agents/implementer.toml` custom agent) with a structured handoff carrying: plan path, slice objective, acceptance criteria addressed, files in scope, exact verification commands, and commit message format. The main session (orchestrator) stays on decomposition, handoff authoring, report adjudication, and plan upkeep. Inline implementation is allowed only for (a) trivial single-file edits where a handoff costs more than the change, or (b) implementer dispatch failure (fall back inline and note the fallback in the report, same convention as the post-implementation pipeline). See `.claude/rules/ralph/model-routing.md` and `.claude/rules/ralph/subagent-policy.md`. Before dispatching, commit any outstanding plan/bookkeeping edits (or confirm they do not overlap the slice's files in scope) so the implementer starts from an unambiguous baseline.
7. **Commit gate after each slice (Validation Gate):**
   - **Delegated slices**: the implementer has already run the handoff's verification commands and committed. The orchestrator's gate is adjudication: check the returned report (verification evidence present, commit-boundary evidence shows no in-scope or unexpected dirt (pre-existing out-of-scope bookkeeping noted in the report is acceptable), commit SHA is the new branch `HEAD` — `git rev-parse HEAD` matches the reported SHA (an existing-but-stale SHA would still satisfy `git log -1`)), and spot-check by re-running a verification command when the report leaves doubt. Do not re-stage or re-commit the slice; one slice = one commit, owned by the implementer.
   - **Inline slices** (trivial edit / dispatch-failure fallback): run `./scripts/run-verify.sh` (or equivalent), stage the slice's files, and commit with conventional format: `<type>: <description>` — the original gate, unchanged.
   - If verification fails in either mode, fix before proceeding. Do not accumulate unverified changes. See `.claude/rules/ralph/git-commit-strategy.md`.
8. Update the plan's progress checklist while working.
9. If the task splits cleanly, delegate focused research or review to subagents.
10. If repeated failures occur, reduce scope, inspect evidence, and revise the plan instead of thrashing.
11. Keep docs, contracts, and tests aligned with behavior changes.
12. Before presenting completion, run `./scripts/run-verify.sh` or equivalent deterministic checks.
13. After criteria met, run the post-implementation pipeline per `.claude/rules/ralph/subagent-policy.md`:
    a. `reviewer` → `/self-review` — stop if CRITICAL findings
    b. `verifier` → `/verify` — stop if fail verdict
    c. `tester` → `/test` — stop if fail verdict
    d. `doc-maintainer` → `/sync-docs`
    In Claude Code, use `Task(subagent_type=...)` calls. In Codex, use the
    matching `.codex/agents/` custom agents and keep the same order. If a
    subagent dispatch fails, run that step inline and note the fallback in the
    report.
    e. **Invoke `/cross-review` via the Skill tool** (optional, inline — if the reviewer CLI is unavailable, skip to `/pr`). The skill reads `cycle-count.json` and enforces `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default 2). On re-run after ACTION_REQUIRED fixes, `/cross-review` increments `cycle-count.json`.
    f. **Invoke `/pr` via the Skill tool** — do NOT run `gh pr create` directly. The `/pr` skill enforces the Japanese template, pre-checks, plan archiving, and task worktree/local branch cleanup. On success, `/pr` deletes `.harness/state/standard-pipeline/active-plan.json` and `cycle-count.json`.

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
- Delegated slices over orchestrator-written code

## CLI execution modes

This skill runs under both Claude Code and Codex. The execution mode follows
the conventions in AGENTS.md and `.codex/AGENTS.override.md`.

| Aspect | Claude Code | Codex |
|--------|-------------|-------|
| Skill invocation | `/skill-name` slash command | `$skill-name` mention or the `/skills` menu (avoid the `/skill-name` form — it collides with built-ins) |
| Skill body path | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Subagent mechanism | `Task(subagent_type=...)` when a policy delegates | `.codex/agents/` custom agents when a policy delegates |
| Structured prompts | `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit reply |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
