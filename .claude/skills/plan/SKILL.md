---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Use when acceptance criteria, verification strategy, or affected areas need to be made explicit.
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
3. Choose one active plan file. If none exists, create one with `./scripts/new-feature-plan.sh <slug>` or from [template.md](template.md).
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

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown

## Additional resources

- [template.md](template.md)
