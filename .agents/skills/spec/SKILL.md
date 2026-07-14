---
name: spec
description: Turn vague ideas or abstract prompts into detailed, actionable specifications through a grill-me-style decision-tree questioning loop with recommended answers, codebase exploration, web research, and interactive clarification with the user. Invoke when a repository-change request is too vague or abstract for /plan — missing objective, scope, or acceptance criteria. Do not invoke for reviews, Q&A or explanations, execution of an existing plan, trivial fixes, or when the user explicitly requests another skill.
---
Turn an abstract idea into a detailed specification.

## Role separation: /spec vs /plan

| Aspect | /spec | /plan |
|--------|-------|-------|
| Input | Vague idea, abstract prompt | Clear spec, defined task |
| Focus | **What** to build (requirements, constraints) | **How** to build (implementation strategy, files) |
| Output | Spec doc (`docs/specs/*.md`) / GitHub issue | Implementation plan (`docs/plans/active/*.md`) |
| Research | Codebase exploration, web search, best practices | Affected files, risk analysis |
| User interaction | Decision-tree questioning with recommended answers + clarification | Flow selection (standard/Ralph) + critical-fork resolution (convergent) |

`/spec` comes before `/plan`. Use `/spec` when the request is too vague for `/plan`.

## Goals

- Transform ambiguous requests into implementation-ready specifications
- Expand sparse inputs (even a one-line prompt) through decision-tree questioning with recommended answers
- Discover requirements through codebase analysis and best-practice research
- Resolve residual ambiguity through targeted user questions
- Produce a versioned spec that survives context loss
- Isolate every temporary or versioned spec artifact in a clean-base task
  worktree before writing it

## Steps

1. **Understand the request** (internal, no user interaction): Read the user's input. List what is clear and what is ambiguous or missing. This is an internal triage step — do NOT call `AskUserQuestion` here. The list feeds the next step.

2. **Interrogate the decision tree with research-backed questions**: Use `AskUserQuestion` iteratively to question the user about every aspect of the spec until there is shared understanding. Treat this as a grill-me-style planning loop, not as an open-ended ideation pass.
   - Walk the design tree branch by branch, resolving upstream decisions before downstream ones and making decision dependencies explicit.
   - Ask questions in batches of five, unless fewer than five unresolved decisions remain.
   - For every question, include the recommended answer and a short rationale so the user can accept, reject, or refine it.
   - When the user asks questions or challenges a recommendation, answer directly, update the decision map, and then continue the loop.
   - As the user answers, maintain an in-memory decision map/spec outline with resolved decisions, rejected alternatives, dependencies, and remaining branches.
   - Before asking any question, check whether codebase exploration can answer it. If the repository answers the question, investigate instead of asking, record the evidence, and move to the next unresolved decision.
   - Explore inline for existing related code, current patterns and conventions, impact areas, dependencies, and similar implementations.
   - When a branch depends on external best practices, current library/framework behavior, or reference implementations, use `WebSearch`/`WebFetch` or Context7 MCP as appropriate before asking the user to decide.
   - Continue until the user signals that the idea is sufficiently shaped, or until additional questions stop producing new implementation-relevant information.
   - Respect anti-bottleneck: ask only for genuine product, scope, or trade-off decisions that cannot be resolved from repo context, research, or a reasonable default.

3. **Clarify residual requirements**: Use `AskUserQuestion` only for ambiguity that remains after the decision-tree loop:
   - Underspecified aspects surfaced by exploration/research
   - Trade-off decisions informed by newly gathered context (e.g., simplicity vs extensibility)
   - Priority and scope boundaries not yet nailed down
   - User preferences and constraints tied to findings
   - Ask in batches of five with recommended answers when several residual decisions remain.
   - Purpose is **convergent** — narrow toward a single, implementable spec.

4. **Draft the spec** (in memory — do NOT write to file yet):
   - Compose the full spec content from [template.md](template.md) using findings from steps 2-3
   - Include trade-off analysis with rationale
   - List resolved and remaining open questions
   - Add references to research sources
   - Keep the draft in memory for user review in the next step

5. **Final review with user**: Present the draft spec to the user for approval before any file or issue creation.
   - Show a structured summary covering: overview, functional requirements (bullet points), acceptance criteria, in/out of scope, open questions
   - Use `AskUserQuestion` to ask:
     - Question: "Confirm the spec content above?"
     - Options:
       1. **Approve** — finalize the spec as shown
       2. **Needs changes** — apply feedback before finalizing
   - If the user selects "Needs changes": apply the feedback to the in-memory draft and repeat this step
   - If the user selects "Approve": proceed to step 6

6. **Output selection**: Use `AskUserQuestion` to ask:
   - Question: "The spec is ready. How should it be processed?"
   - Options (3 choices):
     1. **Issue-only** — create a GitHub issue and leave no repo-tracked spec file
     2. **Save spec file** — write `docs/specs/<date>-<slug>.md` and create a docs/spec PR
     3. **Save spec file and hand off to /plan** — write the spec, then enter implementation planning in the same worktree

7. **Ensure a spec task worktree before any output write**:
   - Derive a stable `<slug>` from the approved spec title.
   - Run `./scripts/ralph-worktree.sh ensure --id spec-<slug> --kind spec --branch docs/spec-<slug> --path .claude/worktrees/spec-<slug> --canonical-ref "<source request or issue URL>" --cleanup-policy <mode>`.
   - The helper must create the worktree from a clean default branch, refuse branch/path collisions unless they match an existing state record, and store local state under `$(git rev-parse --git-common-dir)/ralph/worktrees/`.
   - All output work below executes inside the returned worktree path.

8. **Execute the chosen path** (write only after user approval in step 5 and after step 7 worktree ensure):

   **Issue-only** (option 1):
   - Write the issue body to `.harness/state/specs/<date>-<slug>.md` inside the spec worktree.
   - Run `gh issue create --title "<spec title>" --body-file .harness/state/specs/<date>-<slug>.md`.
   - Treat the GitHub issue URL / issue number as the canonical reference for future `/plan` runs.
   - On successful issue creation, run `./scripts/ralph-worktree.sh cleanup --id spec-<slug> --force-branch` so the temporary worktree and local branch do not linger.
   - If `gh` fails, leave the worktree and local state in place so the user can retry or recover the issue body.

   **Save spec file** (option 2):
   - Ensure `docs/specs/` exists inside the spec worktree.
   - Write the approved spec to `docs/specs/<date>-<slug>.md`.
   - Commit with `docs: add <slug> spec`.
   - Create a docs/spec PR using the same PR title/body standards as `/pr`, but without requiring implementation review/test artifacts.
   - On successful PR creation and push verification, run `./scripts/ralph-worktree.sh cleanup --id spec-<slug> --force-branch`.

   **Save spec file and hand off to /plan** (option 3):
   - Ensure `docs/specs/` exists inside the spec worktree.
   - Write the approved spec to `docs/specs/<date>-<slug>.md`.
   - Commit with `docs: add <slug> spec`.
   - Invoke `Skill(skill="plan")` from the same worktree.
   - The spec file path may be recorded in the plan, but future resumes must prefer the canonical issue/spec reference over stale absolute worktree paths.

## Anti-bottleneck

Before asking the user a question, first check whether you can answer it by:
- Inspecting the codebase (existing patterns, conventions, tests)
- Reading existing docs or plans
- Running scripts or checks
- Choosing a reasonable default and documenting it

Only use `AskUserQuestion` for genuine ambiguity that cannot be resolved from repo context. See the `anti-bottleneck` skill for the full checklist.

## Output

- Spec file at `docs/specs/<date>-<slug>.md`
- Or GitHub issue with spec content and immediate spec worktree cleanup
- Or docs/spec PR with immediate spec worktree cleanup
- Or `/plan` skill invocation for immediate implementation planning in the same worktree

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
