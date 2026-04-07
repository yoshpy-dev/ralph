---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Accepts an optional GitHub issue number or URL for context pre-fill. Creates a feature branch after plan approval. Manual trigger only.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Write, Edit, Bash
---
Create or update a plan in `docs/plans/active/`.

## Goals

- Turn a request into a versioned plan that survives context loss
- Define acceptance criteria and evidence before deep implementation
- Make later review and verification cheaper

## Steps

1. Read `AGENTS.md`, `CLAUDE.md`, relevant `.claude/rules/`, and existing active plans.
2. Inspect only enough code and docs to understand the request and blast radius.
2.5. If a GitHub issue number or URL is provided:
     a. `gh issue view <number> --json title,body,labels,number`
     b. Pre-fill: Objective from title, Related request from body, Related issue: #N
     c. If no issue provided: set "Related issue: N/A"
3. Choose one active plan file. If none exists, create one with `./scripts/new-feature-plan.sh <slug> [issue-number]` or from [template.md](template.md).
4. Fill in:
   - objective
   - scope and non-goals
   - assumptions
   - affected files or systems
   - acceptance criteria
   - implementation outline
   - verification strategy
   - risk register
   - rollout or rollback notes
   - evidence targets
5. Keep the plan high-level enough to avoid cascading low-level mistakes.
6. End with a short readiness checklist.
7. Create feature branch:
   a. Determine type from context (feat/fix/docs/refactor/chore).
   b. With issue: `git checkout -b <type>/<issue>/<slug>`
   c. Without issue: `git checkout -b <type>/<slug>`
   d. Slug = plan filename slug. Do NOT push yet.

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Feature branch created and checked out

## Anti-bottleneck

Before asking the user for confirmation or choices during planning, first check whether the answer is available from the codebase, existing plans, docs, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

- [template.md](template.md)
