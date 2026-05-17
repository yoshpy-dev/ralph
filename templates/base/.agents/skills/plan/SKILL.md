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
4. **Flow selection**: Use **AskUserQuestion** to ask the user which execution flow to use.
   - Question: "Which development flow do you want to use?"
   - Options:
     1. **Standard flow (/work)** — interactive implementation inside Claude Code (short to medium tasks)
     2. **Ralph Loop (/loop)** — directory-based plan with autonomous parallel slice execution (large, divisible tasks)
   - If the plan mentions large-scale refactoring, migration, test-coverage campaigns, or multi-file autonomous work, recommend Ralph Loop.
   - After the user chooses, proceed to step 5.
5. Choose a branch type (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`, `perf`, `release`, or `security`) and a stable slug from the request.
6. **Ensure the task worktree before creating the plan file**:
   - First run `./scripts/ralph-worktree.sh current`. If it returns a matching task state (for example from `/spec` handoff), adopt that worktree and do not create a second one.
   - Otherwise run `./scripts/ralph-worktree.sh ensure --id plan-<slug> --kind <standard|loop> --branch <type>/<slug> --path .claude/worktrees/<slug> --canonical-ref "<issue URL, spec reference, or request summary>" --cleanup-policy pr-success`, where the kind matches the flow selected in step 4.
   - The helper must create the worktree from a clean default branch and store state under `$(git rev-parse --git-common-dir)/ralph/worktrees/`.
   - If a matching state record exists, resume it. If the branch/path exists with mismatched state, stop instead of overwriting.
   - All subsequent plan creation and edits happen inside the returned worktree path.
7. Create one active plan file inside the task worktree based on the flow selected in step 4:
   - **Standard flow**: Create with `./scripts/new-feature-plan.sh --type <type> <slug> [issue-number]` or from [template.md](template.md).
   - **Ralph Loop**: Create with `./scripts/new-ralph-plan.sh --type <type> <slug> [issue-number] [slice-count]` to generate a directory-based plan structure under `docs/plans/active/<date>-<slug>/`.
   - Set `Branch:` to the branch recorded in the worktree state rather than leaving it as `TBD`.
8. Fill in:
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
9. **Critical forks (convergent)**: After the initial draft is in place, scan the plan for "critical forks" — implementation decisions that meet **all three** of:
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
   - The flow-level choice already made in step 4
   - Anything a reasonable default + explicit assumption would cover

   Purpose is **convergent** — narrow between enumerated options, not expand the design space. Divergent ideation belongs to `/spec`, not here.

10. Keep the plan high-level enough to avoid cascading low-level mistakes.
11. End with a short readiness checklist.
12. **Codex plan advisory (optional)**:
    a. Run `./scripts/codex-check.sh` via Bash.
    b. If exit 1 (not available): note "Codex not available — skipping plan advisory" and proceed to step 11.
    c. If exit 0 (available): invoke Codex to adversarially review the plan via Bash:
       `codex exec --sandbox read-only "You are an adversarial plan reviewer. Your job is to break confidence in this plan, not to validate it. Default to skepticism — assume the plan can fail in subtle, high-cost ways until evidence says otherwise. Review for: (1) blind spots and missing risks — what failure modes are not addressed? (2) scope concerns — too broad, too narrow, or poorly bounded? (3) acceptance criteria gaps — can each criterion be verified deterministically? (4) design decision weaknesses — are there simpler or safer alternatives? (5) rollback and partial-failure scenarios — what happens if implementation stalls halfway? Report only material findings. Each finding must answer: What can go wrong? Why is this plan vulnerable? What is the likely impact? What concrete change would reduce the risk? Number each finding with severity [HIGH/MEDIUM/LOW]. Prefer one strong finding over several weak ones. If the plan looks solid, say so directly with no findings. Here is the plan file to review: docs/plans/active/<plan-file>"`
    d. Present Codex findings to the user as a numbered list.
    e. If Codex returned no actionable findings: note "Codex: no findings" and proceed to step 11.
    f. If findings exist, use AskUserQuestion:
       - Question: "Codex returned findings on the plan. How do you want to proceed?"
       - Options:
         1. Update plan — edit plan per relevant findings, then re-display
         2. Acknowledge findings, continue — proceed without changes
    g. After user decision, proceed to step 13.
13. **Flow confirmation**: Confirm the flow selected in step 4 and state which skill to invoke next:
    - Standard flow → `/work`
    - Ralph Loop → `/loop`

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Task worktree path and branch recorded by `ralph-worktree.sh`
- Chosen execution flow (standard /work or Ralph Loop /loop)

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
