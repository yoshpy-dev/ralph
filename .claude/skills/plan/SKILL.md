---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Accepts an optional GitHub issue number or URL for context pre-fill. Resolves high-leverage implementation forks with the user before finalizing. Ensures a clean-base task worktree before writing plan artifacts.
---
Create or update a plan in `docs/plans/active/` from an isolated task
worktree.

## Goals

- Turn a request into a versioned plan that survives context loss
- Define acceptance criteria and evidence before deep implementation
- Make later review and verification cheaper
- Keep plan artifacts out of the default checkout by ensuring a task worktree
  before the first file write

## Steps

1. Read `AGENTS.md`, `CLAUDE.md`, relevant `.claude/rules/`, and existing active plans.
2. Inspect only enough code and docs to understand the request and blast radius.
3. If a GitHub issue number or URL is provided:
   a. `gh issue view <number> --json title,body,labels,number`
   b. Pre-fill: Objective from title, Related request from body, Related issue: #N
   c. If no issue provided: set "Related issue: N/A"
4. Choose a branch type (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`, `perf`, `release`, or `security`) and a stable slug from the request.
5. **Ensure the task worktree before creating the plan file**:
   - First run `./scripts/ralph-worktree.sh current`. If it returns a matching task state (for example from `/spec` handoff), adopt that worktree and do not create a second one.
   - Otherwise run `./scripts/ralph-worktree.sh ensure --id plan-<slug> --kind standard --branch <type>/<slug> --path .claude/worktrees/<slug> --canonical-ref "<issue URL, spec reference, or request summary>" --cleanup-policy pr-success`.
   - The helper must create the worktree from a clean default branch and store state under `$(git rev-parse --git-common-dir)/ralph/worktrees/`.
   - If a matching state record exists, resume it. If the branch/path exists with mismatched state, stop instead of overwriting.
   - All subsequent plan creation and edits happen inside the returned worktree path.
6. Create one active plan file inside the task worktree with `./scripts/new-feature-plan.sh --type <type> <slug> [issue-number]` or from [template.md](template.md). Set `Branch:` to the branch recorded in the worktree state rather than leaving it as `TBD`.
7. Fill in:
   - objective
   - scope and non-goals
   - assumptions
   - affected files or systems
   - acceptance criteria
   - implementation outline
   - verify plan (static analysis checks, spec compliance criteria, documentation drift checks, evidence to capture)
   - test plan (unit tests, integration tests, regression tests, edge cases, evidence to capture)
   - risk register
   - rollout or rollback notes
   - evidence targets
8. **Critical forks (convergent)**: After the initial draft is in place, scan the plan for "critical forks" — implementation decisions that meet **all three** of:
   - Two or more approaches differ materially in risk, cost, or rollback profile
   - The choice cannot be resolved from the codebase, existing `.claude/rules/`, docs, or a reasonable default
   - Reversing the decision mid-implementation would cost more than roughly one slice of rework

   For each critical fork identified:
   a. Use `AskUserQuestion` with one focused question and 2-4 concrete options. Each option must briefly state its pros/cons so the user can choose informedly.
   b. Record the chosen approach and its rationale in the plan's "Design decisions" section (see [template.md](template.md)).
   c. If the chosen option invalidates other plan sections (outline, risks, affected files), revise them before continuing.

   If no critical forks exist after scanning, write "Critical forks: None" in the Design decisions section and proceed.

   **Do NOT ask about**:
   - Stylistic or easily reversible choices (internal naming, helper placement inside an established module pattern)
   - Decisions already settled by `.claude/rules/`, `AGENTS.md`, or the upstream `/spec` output
   - Anything a reasonable default + explicit assumption would cover

   Purpose is **convergent** — narrow between enumerated options, not expand the design space. Divergent ideation belongs to `/spec`, not here.

9. Keep the plan high-level enough to avoid cascading low-level mistakes.
10. End with a short readiness checklist.
11. **Codex plan advisory (optional)**:
    a. Run `./scripts/codex-check.sh` via Bash.
    b. If exit 1 (not available): note "Codex not available — skipping plan advisory" and proceed to completion.
    c. If exit 0 (available): invoke Codex to adversarially review the plan via Bash:
       `codex exec --sandbox read-only "You are an adversarial plan reviewer. Your job is to break confidence in this plan, not to validate it. Default to skepticism — assume the plan can fail in subtle, high-cost ways until evidence says otherwise. Review for: (1) blind spots and missing risks — what failure modes are not addressed? (2) scope concerns — too broad, too narrow, or poorly bounded? (3) acceptance criteria gaps — can each criterion be verified deterministically? (4) design decision weaknesses — are there simpler or safer alternatives? (5) rollback and partial-failure scenarios — what happens if implementation stalls halfway? Report only material findings. Each finding must answer: What can go wrong? Why is this plan vulnerable? What is the likely impact? What concrete change would reduce the risk? Number each finding with severity [HIGH/MEDIUM/LOW]. Prefer one strong finding over several weak ones. If the plan looks solid, say so directly with no findings. Here is the plan file to review: docs/plans/active/<plan-file>"`
    d. Present Codex findings to the user as a numbered list.
    e. If Codex returned no actionable findings: note "Codex: no findings" and proceed to completion.
    f. If findings exist, use AskUserQuestion:
       - Question: "Codex returned findings on the plan. How do you want to proceed?"
       - Options:
         1. Update plan — edit plan per relevant findings, then re-display
         2. Acknowledge findings, continue — proceed without changes
    g. After user decision, state that `/work` is the next skill to invoke.

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Task worktree path and branch recorded by `ralph-worktree.sh`
- Next step: invoke `/work`

## Anti-bottleneck

Before asking the user for confirmation or choices during planning, first check whether the answer is available from the codebase, existing plans, docs, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

- [template.md](template.md)

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
