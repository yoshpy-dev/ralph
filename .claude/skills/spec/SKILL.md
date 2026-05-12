---
name: spec
description: Turn vague ideas or abstract prompts into detailed, actionable specifications through iterative brainstorming, codebase exploration, web research, and interactive clarification with the user. Manual trigger only.
disable-model-invocation: true
allowed-tools: Task, Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion, WebSearch, WebFetch, Skill
---
Turn an abstract idea into a detailed specification.

## Role separation: /spec vs /plan

| Aspect | /spec | /plan |
|--------|-------|-------|
| Input | Vague idea, abstract prompt | Clear spec, defined task |
| Focus | **What** to build (requirements, constraints) | **How** to build (implementation strategy, files) |
| Output | Spec doc (`docs/specs/*.md`) / GitHub issue | Implementation plan (`docs/plans/active/*.md`) |
| Research | Codebase exploration, web search, best practices | Affected files, risk analysis |
| User interaction | Iterative brainstorming (divergent) + clarification (convergent) | Flow selection (standard/Ralph) + critical-fork resolution (convergent) |

`/spec` comes before `/plan`. Use `/spec` when the request is too vague for `/plan`.

## Goals

- Transform ambiguous requests into implementation-ready specifications
- Expand sparse inputs (even a one-line prompt) through iterative brainstorming
- Discover requirements through codebase analysis and best-practice research
- Resolve residual ambiguity through targeted user questions
- Produce a versioned spec that survives context loss

## Steps

1. **Understand the request** (internal, no user interaction): Read the user's input. List what is clear and what is ambiguous or missing. This is an internal triage step — do NOT call `AskUserQuestion` here. The list feeds the next step.

2. **Brainstorm to expand the idea**: Use `AskUserQuestion` iteratively to widen the problem space before converging. This step is mandatory when the input is sparse (e.g., a single sentence like "I want to achieve X"); skip or shorten when the input is already detailed.
   - Explore these axes one at a time (repeat as needed, no iteration cap):
     - Purpose and background (why this matters, the problem to solve)
     - Target users and usage scenarios
     - Alternative approaches (other ways to solve it, and their trade-offs)
     - Success criteria (what "done" looks like)
     - Scope boundaries and priority (what is explicitly out, MVP scope)
     - Known constraints (technical, time, team)
   - Purpose is **divergent** (expand options), distinct from step 5 which is **convergent** (resolve remaining ambiguity).
   - Continue until the user signals that the idea is sufficiently shaped, or until further questions stop yielding new information.
   - Respect anti-bottleneck: before asking, check whether the repo already answers the question.

3. **Explore the codebase**: Use `Task(subagent_type="Explore")` to investigate:
   - Existing code related to the request
   - Current patterns and conventions
   - Potential impact areas and dependencies
   - Similar implementations that already exist

4. **Research best practices**: Use `WebSearch` and `WebFetch` to investigate:
   - Industry best practices for the problem domain
   - Common approaches and trade-offs
   - Reference implementations in well-known projects
   - Relevant library/framework documentation (via Context7 MCP if applicable)

5. **Clarify residual requirements**: Use `AskUserQuestion` actively to resolve any ambiguity that remains after brainstorming and research:
   - Underspecified aspects surfaced by exploration/research
   - Trade-off decisions informed by newly gathered context (e.g., simplicity vs extensibility)
   - Priority and scope boundaries not yet nailed down
   - User preferences and constraints tied to findings
   - Repeat this step as many times as needed. Do not guess when you can ask.
   - Purpose is **convergent** — narrow toward a single, implementable spec.

6. **Draft the spec** (in memory — do NOT write to file yet):
   - Compose the full spec content from [template.md](template.md) using findings from steps 2-5
   - Include trade-off analysis with rationale
   - List resolved and remaining open questions
   - Add references to research sources
   - Keep the draft in memory for user review in the next step

7. **Final review with user**: Present the draft spec to the user for approval before any file or issue creation.
   - Show a structured summary covering: overview, functional requirements (bullet points), acceptance criteria, in/out of scope, open questions
   - Use `AskUserQuestion` to ask:
     - Question: "Confirm the spec content above?"
     - Options:
       1. **Approve** — finalize the spec as shown
       2. **Needs changes** — apply feedback before finalizing
   - If the user selects "Needs changes": apply the feedback to the in-memory draft and repeat this step
   - If the user selects "Approve": proceed to step 8

8. **Output selection**: Use `AskUserQuestion` to ask:
   - Question: "The spec is ready. How should it be processed?"
   - Options (4 choices):
     1. **Create a GitHub issue** — file the spec as an issue
     2. **Save the spec file** — write markdown only to `docs/specs/`
     3. **Save the spec file and create a GitHub issue** — do both
     4. **Save the spec file and hand off to /plan** — save, then enter the implementation planning phase

9. **Execute the chosen path** (write only after user approval in step 7):

   **File save** (options 2, 3, 4):
   - Ensure `docs/specs/` directory exists
   - Write the approved spec to `docs/specs/<date>-<slug>.md`

   **GitHub issue creation** (options 1, 3):
   - Save the spec file first (always, as fallback), then run: `gh issue create --title "<spec title>" --body-file docs/specs/<date>-<slug>.md`
   - If `gh` fails (auth error, network issue):
     a. Warn the user: "GitHub issue creation failed. The spec has been saved to docs/specs/<date>-<slug>.md."

   **Plan transition** (option 4):
   - After saving the spec file, invoke: `Skill(skill="plan")`
   - The spec file path should be referenced in the plan's "Related request" field

## Anti-bottleneck

Before asking the user a question, first check whether you can answer it by:
- Inspecting the codebase (existing patterns, conventions, tests)
- Reading existing docs or plans
- Running scripts or checks
- Choosing a reasonable default and documenting it

Only use `AskUserQuestion` for genuine ambiguity that cannot be resolved from repo context. See the `anti-bottleneck` skill for the full checklist.

## Output

- Spec file at `docs/specs/<date>-<slug>.md`
- Optionally: GitHub issue with spec content
- Optionally: `/plan` skill invocation for immediate implementation planning

## CLI execution modes

This skill runs under both Claude Code and Codex. The execution mode follows
the conventions in AGENTS.md and `.codex/AGENTS.override.md`.

| Aspect | Claude Code | Codex |
|--------|-------------|-------|
| Skill invocation | `/skill-name` slash command | `$skill-name` mention or the `/skills` menu (avoid the `/skill-name` form — it collides with built-ins) |
| Skill body path | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Subagents | Parallel calls via `Task(subagent_type=...)` | Sequential inline execution — chained within a single agent |
| Structured prompts | `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit reply |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
